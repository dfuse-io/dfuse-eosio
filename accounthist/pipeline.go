package accounthist

import (
	"context"
	"fmt"

	"github.com/dfuse-io/bstream"
	"github.com/dfuse-io/bstream/forkable"
)

func (ws *Service) SetupSource() error {
	ctx := context.Background()

	lastProcessedBlock, err := ws.GetLastProcessedBlock(ctx)
	if err != nil {
		return fmt.Errorf("could not get last processed block: %w", err)
	}

	gateType := bstream.GateExclusive

	if lastProcessedBlock <= bstream.GetProtocolFirstStreamableBlock {
		lastProcessedBlock = bstream.GetProtocolFirstStreamableBlock
		gateType = bstream.GateInclusive
	}

	// WARN: this is IRREVERSIBLE ONLY, we'll need to check start block when
	// we want some reversible segment support.

	gate := bstream.NewBlockNumGate(lastProcessedBlock, gateType, ws, bstream.GateOptionWithLogger(zlog))
	gate.MaxHoldOff = 1000

	forkableHandler := forkable.New(gate,
		forkable.WithLogger(zlog),
		forkable.WithFilters(forkable.StepIrreversible),
	)

	var filterPreprocessFunc bstream.PreprocessFunc
	if ws.blockFilter != nil {
		filterPreprocessFunc = func(blk *bstream.Block) (interface{}, error) {
			return nil, ws.blockFilter(blk)
		}
	}

	fs := bstream.NewFileSource(
		ws.blocksStore,
		lastProcessedBlock,
		2, // parallel download count
		filterPreprocessFunc,
		forkableHandler,
		bstream.FileSourceWithLogger(zlog),
	)

	ws.source = fs

	return nil
}
