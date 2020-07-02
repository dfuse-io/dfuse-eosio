package filtering

import (
	"fmt"

	pbcodec "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/codec/v1"
)

type BlockFilter struct {
	IncludeProgram *CELFilter
	ExcludeProgram *CELFilter
}

func NewBlockFilter(includeProgramCode, excludeProgramCode string) (*BlockFilter, error) {
	includeFilter, err := newCELFilterInclude(includeProgramCode)
	if err != nil {
		return nil, fmt.Errorf("include filter: %w", err)
	}

	excludeFilter, err := newCELFilterExclude(excludeProgramCode)
	if err != nil {
		return nil, fmt.Errorf("exclude filter: %w", err)
	}

	return &BlockFilter{
		IncludeProgram: includeFilter,
		ExcludeProgram: excludeFilter,
	}, nil
}

func (f *BlockFilter) TransformInPlace(block *pbcodec.Block) {
	block.FilteringApplied = true
	block.FilteringIncludeFilterExpr = f.IncludeProgram.code
	block.FilteringExcludeFilterExpr = f.ExcludeProgram.code

	var filteredTrxTrace []*pbcodec.TransactionTrace
	filteredExecutedInputActionCount := uint32(0)
	filteredExecutedTotalActionCount := uint32(0)

	for _, trxTrace := range block.UnfilteredTransactionTraces {
		trxTraceAddedToFiltered := false
		for _, actTrace := range trxTrace.ActionTraces {
			if !f.shouldProcess(trxTrace, actTrace) {
				continue
			}

			actTrace.FilteringMatched = true
			filteredExecutedTotalActionCount++
			if actTrace.IsInput() {
				filteredExecutedInputActionCount++
			}

			if !trxTraceAddedToFiltered {
				filteredTrxTrace = append(filteredTrxTrace, trxTrace)
				trxTraceAddedToFiltered = true
			}
		}

		if trxTrace.FailedDtrxTrace != nil {
			for _, actTrace := range trxTrace.FailedDtrxTrace.ActionTraces {
				if !f.shouldProcess(trxTrace.FailedDtrxTrace, actTrace) {
					continue
				}

				actTrace.FilteringMatched = true
				if !trxTraceAddedToFiltered {
					filteredTrxTrace = append(filteredTrxTrace, trxTrace)
					trxTraceAddedToFiltered = true
				}
			}
		}
	}

	block.UnfilteredTransactionTraces = nil
	block.FilteredTransactionTraces = filteredTrxTrace
	block.FilteredTransactionTraceCount = uint32(len(filteredTrxTrace))
	block.FilteredExecutedInputActionCount = filteredExecutedInputActionCount
	block.FilteredExecutedTotalActionCount = filteredExecutedTotalActionCount
}

func (f *BlockFilter) shouldProcess(trxTrace *pbcodec.TransactionTrace, actTrace *pbcodec.ActionTrace) bool {
	activation := actionTraceActivation{trace: actTrace, trxScheduled: trxTrace.Scheduled}
	// If the include program does not match, there is nothing more to do here
	if !f.IncludeProgram.match(&activation) {
		return false
	}

	// At this point, the inclusion expr matched, let's check it was included but should be now excluded based on the exclusion filter
	if f.ExcludeProgram.match(&activation) {
		return false
	}

	// We are included and NOT excluded, this transaction trace/action trace match the block filter
	return true
}
