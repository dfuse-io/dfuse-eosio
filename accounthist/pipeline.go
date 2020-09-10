package accounthist

import (
	"context"
	"fmt"

	"github.com/dfuse-io/bstream"
	"github.com/dfuse-io/bstream/forkable"
	pbaccounthist "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/accounthist/v1"
	pbcodec "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/codec/v1"
	"github.com/golang/protobuf/proto"
	"go.uber.org/zap"
)

func (ws *Service) SetupSource() error {
	ctx := context.Background()

	checkpoint, err := ws.resolveCheckpoint(ctx)
	if err != nil {
		return fmt.Errorf("unable to resolve shard checkpoint: %w", err)
	}
	ws.lastCheckpoint = checkpoint

	startProcessingBlockNum, fileSourceStartBlockNum, fileSourceStartBlockId, gateType, err := ws.resolveStartBlock(ctx, checkpoint)
	if err != nil {
		return fmt.Errorf("unable to resolve start block: %w", err)
	}

	ws.setupPipeline(startProcessingBlockNum, fileSourceStartBlockNum, fileSourceStartBlockId, gateType)

	return nil
}

func (ws *Service) setupPipeline(startProcessingBlockNum, fileSourceStartBlockNum uint64, fileSourceStartBlockId string, gateType bstream.GateType) {
	zlog.Info("setting up pipeline",
		zap.Uint64("start_processing_block_num", startProcessingBlockNum),
		zap.Uint64("file_source_start_block_num", fileSourceStartBlockNum),
		zap.String("file_source_start_block_id", fileSourceStartBlockId),
		zap.String("gate_type", gateType.String()),
	)

	// WARN: this is IRREVERSIBLE ONLY
	options := []forkable.Option{
		forkable.WithLogger(zlog),
		forkable.WithFilters(forkable.StepIrreversible),
	}

	if fileSourceStartBlockId != "" {
		zlog.Info("file source start block id defined, adding a with inclusive LIB option ",
			zap.String("file_source_start_block_id", fileSourceStartBlockId),
		)
		options = append(options, forkable.WithInclusiveLIB(bstream.NewBlockRef(fileSourceStartBlockId, fileSourceStartBlockNum)))
	}

	gate := bstream.NewBlockNumGate(startProcessingBlockNum, gateType, ws, bstream.GateOptionWithLogger(zlog))
	forkableHandler := forkable.New(gate, options...)

	fs := bstream.NewFileSource(
		ws.blocksStore,
		fileSourceStartBlockNum,
		2,
		preprocessingFunc(ws.blockFilter),
		forkableHandler,
		bstream.FileSourceWithLogger(zlog),
	)

	ws.source = fs
}

func (ws *Service) resolveCheckpoint(ctx context.Context) (*pbaccounthist.ShardCheckpoint, error) {
	if ws.shardNum != 0 {
		checkpoint := newShardCheckpoint(ws.startBlockNum)
		zlog.Info("starting a none shard-0, thus ignoring checkout and starting at the beginning",
			zap.Int("shard_num", int(ws.shardNum)),
			zap.Reflect("checkpoint", checkpoint),
		)
		return checkpoint, nil
	}

	// Retrieved lastProcessedBlock must be in the shard's range, and that shouldn't
	// change across invocations, or in the lifetime of the database.
	checkpoint, err := ws.GetShardCheckpoint(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetching shard checkpoint: %w", err)
	}

	if checkpoint != nil {
		zlog.Info("found checkpoint",
			zap.Int("shard_num", int(ws.shardNum)),
			zap.Reflect("checkpoint", checkpoint),
		)
		return checkpoint, nil
	}

	checkpoint = newShardCheckpoint(ws.startBlockNum)
	zlog.Info("starting without checkpoint",
		zap.Int("shard_num", int(ws.shardNum)),
		zap.Reflect("checkpoint", checkpoint),
	)
	return checkpoint, nil
}

func (ws *Service) resolveStartBlock(ctx context.Context, checkpoint *pbaccounthist.ShardCheckpoint) (startProcessingBlockNum uint64, fileSourceStartBlockNum uint64, fileSourceStartBlockId string, gateType bstream.GateType, err error) {
	if checkpoint.LastWrittenBlockId != "" {
		zlog.Info("resolving start blocks from checkpoint last written block",
			zap.Reflect("checkpoint", checkpoint),
		)
		return checkpoint.LastWrittenBlockNum, checkpoint.LastWrittenBlockNum, checkpoint.LastWrittenBlockId, bstream.GateExclusive, nil
	}

	zlog.Info("checkpoint does not have a last written block, resolving start blocks with tracker",
		zap.Reflect("checkpoint", checkpoint),
	)

	fsStartNum, previousIrreversibleID, err := ws.tracker.ResolveStartBlock(ctx, checkpoint.InitialStartBlock)
	if err != nil {
		return 0, 0, "", 0, fmt.Errorf("unable to resolve start block with tracker: %w", err)
	}

	return checkpoint.InitialStartBlock, fsStartNum, previousIrreversibleID, bstream.GateExclusive, nil
}

func newShardCheckpoint(startBlock uint64) *pbaccounthist.ShardCheckpoint {
	if startBlock <= bstream.GetProtocolFirstStreamableBlock {
		startBlock = bstream.GetProtocolFirstStreamableBlock
	}
	return &pbaccounthist.ShardCheckpoint{InitialStartBlock: startBlock}
}

func preprocessingFunc(blockFilter func(blk *bstream.Block) error) bstream.PreprocessFunc {
	return func(blk *bstream.Block) (interface{}, error) {
		if blockFilter != nil {
			if err := blockFilter(blk); err != nil {
				return nil, err
			}
		}

		out := map[uint64][]byte{}
		// Go through `blk`, loop all those transaction traces, all those actions
		// and proto marshal them all in parallel
		block := blk.ToNative().(*pbcodec.Block)
		for _, trxTrace := range block.TransactionTraces() {
			if trxTrace.HasBeenReverted() {
				continue
			}

			actionMatcher := block.FilteringActionMatcher(trxTrace)
			for _, act := range trxTrace.ActionTraces {
				if !actionMatcher.Matched(act.ExecutionIndex) || act.Receipt == nil {
					continue
				}

				acctData := &pbaccounthist.ActionRow{Version: 0, ActionTrace: act}
				rawTrace, err := proto.Marshal(acctData)
				if err != nil {
					return nil, err
				}

				out[act.Receipt.GlobalSequence] = rawTrace
			}
		}

		return out, nil
	}
}
