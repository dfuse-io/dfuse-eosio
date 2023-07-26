package injector

import (
	"context"
	"fmt"

	pbaccounthist "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/accounthist/v1"
	pbcodec "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/codec/v1"
	"github.com/golang/protobuf/proto"
	"github.com/streamingfast/bstream"
	"github.com/streamingfast/bstream/forkable"
	"go.uber.org/zap"
)

func (i *Injector) SetupSource(ignoreCheckpointOnLaunch bool) error {
	ctx := context.Background()

	checkpoint, err := i.resolveCheckpoint(ctx, ignoreCheckpointOnLaunch)
	if err != nil {
		return fmt.Errorf("unable to resolve shard checkpoint: %w", err)
	}
	i.lastCheckpoint = checkpoint

	startProcessingBlockNum, fileSourceStartBlockNum, fileSourceStartBlockId, gateType, err := i.resolveStartBlock(ctx, checkpoint)
	if err != nil {
		return fmt.Errorf("unable to resolve start block: %w", err)
	}

	i.setupPipeline(startProcessingBlockNum, fileSourceStartBlockNum, fileSourceStartBlockId, gateType)

	return nil
}

func (i *Injector) setupPipeline(startProcessingBlockNum, fileSourceStartBlockNum uint64, fileSourceStartBlockId string, gateType bstream.GateType) {
	zlog.Info("setting up pipeline",
		zap.Uint64("start_processing_block_num", startProcessingBlockNum),
		zap.Uint64("file_source_start_block_num", fileSourceStartBlockNum),
		zap.String("file_source_start_block_id", fileSourceStartBlockId),
		zap.String("gate_type", gateType.String()),
	)

	// WARN: this is IRREVERSIBLE ONLY
	options := []forkable.Option{
		forkable.WithLogger(zlog),
		forkable.WithFilters(forkable.StepNew | forkable.StepIrreversible),
	}

	if fileSourceStartBlockId != "" {
		zlog.Info("file source start block id defined, adding a with inclusive LIB option ",
			zap.String("file_source_start_block_id", fileSourceStartBlockId),
		)
		options = append(options, forkable.WithInclusiveLIB(bstream.NewBlockRef(fileSourceStartBlockId, fileSourceStartBlockNum)))
	}

	gate := bstream.NewBlockNumGate(startProcessingBlockNum, gateType, i, bstream.GateOptionWithLogger(zlog))
	forkableHandler := forkable.New(gate, options...)

	fs := bstream.NewFileSource(
		i.blocksStore,
		fileSourceStartBlockNum,
		2,
		PreprocessingFunc(i.BlockFilter),
		forkableHandler,
		bstream.FileSourceWithLogger(zlog),
	)

	i.source = fs
}

func (i *Injector) resolveCheckpoint(ctx context.Context, ignoreCheckpointOnLaunch bool) (*pbaccounthist.ShardCheckpoint, error) {
	if ignoreCheckpointOnLaunch {
		checkpoint := newShardCheckpoint(i.startBlockNum)
		zlog.Info("ignoring checkpoint on launch starting without a checkpoint",
			zap.Int("shard_num", int(i.ShardNum)),
			zap.Reflect("checkpoint", checkpoint),
		)
		return checkpoint, nil
	}

	// Retrieved lastProcessedBlock must be in the shard's range, and that shouldn't
	// change across invocations, or in the lifetime of the database.
	checkpoint, err := i.GetShardCheckpoint(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetching shard checkpoint: %w", err)
	}

	if checkpoint != nil {
		zlog.Info("found checkpoint",
			zap.Int("shard_num", int(i.ShardNum)),
			zap.Reflect("checkpoint", checkpoint),
		)
		i.startedFromCheckpoint = true
		return checkpoint, nil
	}

	checkpoint = newShardCheckpoint(i.startBlockNum)
	zlog.Info("starting without checkpoint",
		zap.Int("shard_num", int(i.ShardNum)),
		zap.Reflect("checkpoint", checkpoint),
	)
	return checkpoint, nil
}

func (i *Injector) resolveStartBlock(ctx context.Context, checkpoint *pbaccounthist.ShardCheckpoint) (startProcessingBlockNum uint64, fileSourceStartBlockNum uint64, fileSourceStartBlockId string, gateType bstream.GateType, err error) {
	if checkpoint.LastWrittenBlockId != "" {
		zlog.Info("resolving start blocks from checkpoint last written block",
			zap.Reflect("checkpoint", checkpoint),
		)
		return checkpoint.LastWrittenBlockNum, checkpoint.LastWrittenBlockNum, checkpoint.LastWrittenBlockId, bstream.GateExclusive, nil
	}

	zlog.Info("checkpoint does not have a last written block, resolving start blocks with tracker",
		zap.Reflect("checkpoint", checkpoint),
	)

	fsStartNum, previousIrreversibleID, err := i.tracker.ResolveStartBlock(ctx, checkpoint.InitialStartBlock)
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

func PreprocessingFunc(blockFilter func(blk *bstream.Block) error) bstream.PreprocessFunc {
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
