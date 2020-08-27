package accounthist

import (
	"context"
	"fmt"

	"github.com/dfuse-io/bstream"
	"github.com/dfuse-io/bstream/forkable"
	pbaccounthist "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/accounthist/v1"
	pbcodec "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/codec/v1"
	"github.com/golang/protobuf/proto"
)

func (ws *Service) SetupSource() error {
	ctx := context.Background()

	// Retrieved lastProcessedBlock must be in the shard's range, and that shouldn't
	// change across invocations, or in the lifetime of the database.
	lastProcessedBlock, err := ws.GetLastProcessedBlock(ctx)
	if err != nil {
		return fmt.Errorf("could not get last processed block: %w", err)
	}

	gateType := bstream.GateExclusive

	if ws.startBlockNum != 0 && lastProcessedBlock < ws.startBlockNum {
		lastProcessedBlock = ws.startBlockNum
		gateType = bstream.GateInclusive
	}

	if lastProcessedBlock <= bstream.GetProtocolFirstStreamableBlock {
		lastProcessedBlock = bstream.GetProtocolFirstStreamableBlock
		gateType = bstream.GateInclusive
	}

	fileSourceStartBlockNum, previousIrreversibleID, err := ws.tracker.ResolveStartBlock(ctx, lastProcessedBlock)
	if err != nil {
		return err
	}

	// WARN: this is IRREVERSIBLE ONLY

	gate := bstream.NewBlockNumGate(lastProcessedBlock, gateType, ws, bstream.GateOptionWithLogger(zlog))
	gate.MaxHoldOff = 1000

	options := []forkable.Option{
		forkable.WithLogger(zlog),
		forkable.WithFilters(forkable.StepIrreversible),
	}
	if previousIrreversibleID != "" {
		options = append(options, forkable.WithInclusiveLIB(bstream.NewBlockRef(previousIrreversibleID, fileSourceStartBlockNum)))
	}

	forkableHandler := forkable.New(gate, options...)

	preprocFunc := func(blk *bstream.Block) (interface{}, error) {
		if ws.blockFilter != nil {
			if err := ws.blockFilter(blk); err != nil {
				return nil, err
			}
		}

		out := map[uint64][]byte{}
		// Go through `blk`, loop all those transaction traces, all those actions
		// and proto marshal them all in parallel
		block := blk.ToNative().(*pbcodec.Block)
		for _, tx := range block.TransactionTraces() {
			if tx.HasBeenReverted() {
				continue
			}
			for _, act := range tx.ActionTraces {
				if act.Receipt == nil {
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

	// another filter func:
	// return map[uint64][]byte{}

	fs := bstream.NewFileSource(
		ws.blocksStore,
		fileSourceStartBlockNum,
		2, // parallel download count
		preprocFunc,
		forkableHandler,
		bstream.FileSourceWithLogger(zlog),
		//bstream.FileSourceParallelPreprocessing(12),
	)

	ws.source = fs

	return nil
}
