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

	var startProcessingBlockNum uint64
	gateType := bstream.GateInclusive

	// Retrieved lastProcessedBlock must be in the shard's range, and that shouldn't
	// change across invocations, or in the lifetime of the database.
	checkpoint, err := ws.GetShardCheckpoint(ctx)
	if err != nil {
		return fmt.Errorf("fetching shard checkpoint: %w", err)
	}

	if checkpoint == nil || ws.shardNum != 0 {
		startBlock := ws.startBlockNum
		if startBlock <= bstream.GetProtocolFirstStreamableBlock {
			startBlock = bstream.GetProtocolFirstStreamableBlock
		}
		zlog.Info("starting without checkpoint", zap.Int("shard_num", int(ws.shardNum)), zap.Uint64("block_num", startBlock))
		checkpoint = &pbaccounthist.ShardCheckpoint{
			InitialStartBlock: startBlock,
		}
		startProcessingBlockNum = startBlock
	} else {
		zlog.Info("starting from checkpoint", zap.Int("shard_num", int(ws.shardNum)), zap.String("block_id", checkpoint.LastWrittenBlockId), zap.Uint64("block_num", checkpoint.LastWrittenBlockNum))
		startProcessingBlockNum = checkpoint.LastWrittenBlockNum
		gateType = bstream.GateExclusive
	}
	checkpoint.TargetStopBlock = ws.stopBlockNum
	ws.lastCheckpoint = checkpoint

	// WARN: this is IRREVERSIBLE ONLY

	options := []forkable.Option{
		forkable.WithLogger(zlog),
		forkable.WithFilters(forkable.StepIrreversible),
	}

	var fileSourceStartBlockNum uint64
	if checkpoint.LastWrittenBlockId != "" {
		options = append(options, forkable.WithInclusiveLIB(bstream.NewBlockRef(checkpoint.LastWrittenBlockId, checkpoint.LastWrittenBlockNum)))
		fileSourceStartBlockNum = checkpoint.LastWrittenBlockNum
	} else {
		fsStartNum, previousIrreversibleID, err := ws.tracker.ResolveStartBlock(ctx, checkpoint.InitialStartBlock)
		if err != nil {
			return err
		}

		if previousIrreversibleID != "" {
			options = append(options, forkable.WithInclusiveLIB(bstream.NewBlockRef(previousIrreversibleID, fsStartNum)))
		}
		fileSourceStartBlockNum = fsStartNum
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

	return nil
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
