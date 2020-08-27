package filtering

import (
	"fmt"

	"github.com/dfuse-io/bstream"
	pbcodec "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/codec/v1"
	"go.uber.org/zap"
)

type BlockFilter struct {
	IncludeProgram              *CELFilter
	ExcludeProgram              *CELFilter
	SystemActionsIncludeProgram *CELFilter
}

func NewBlockFilter(includeProgramCode, excludeProgramCode, systemActionsIncludeProgramCode string) (*BlockFilter, error) {
	includeFilter, err := newCELFilterInclude(includeProgramCode)
	if err != nil {
		return nil, fmt.Errorf("include filter: %w", err)
	}

	excludeFilter, err := newCELFilterExclude(excludeProgramCode)
	if err != nil {
		return nil, fmt.Errorf("exclude filter: %w", err)
	}

	saIncludeFilter, err := newCELFilterSystemActionsInclude(systemActionsIncludeProgramCode)
	if err != nil {
		return nil, fmt.Errorf("system actions include filter: %w", err)
	}

	return &BlockFilter{
		IncludeProgram:              includeFilter,
		ExcludeProgram:              excludeFilter,
		SystemActionsIncludeProgram: saIncludeFilter,
	}, nil
}

// TransformInPlace received a `bstream.Block` pointer, unpack it's native counterpart, a `pbcodec.Block` pointer
// in our case and transforms it in place, modifiying the pointed object. This means that future `ToNative()` calls
// on the bstream block will return a filtered version of this block.
//
// *Important* This method expect that the caller will peform the transformation in lock step, there is no lock
//             performed by this method. It's the caller responsibility to deal with concurrency issues.
func (f *BlockFilter) TransformInPlace(blk *bstream.Block) error {
	// Don't decode the bstream block at all so we save a costly unpacking when both filters are no-op filters
	if f.IncludeProgram.IsNoop() && f.ExcludeProgram.IsNoop() {
		return nil
	}

	block := blk.ToNative().(*pbcodec.Block)
	if !block.FilteringApplied {
		f.transfromInPlace(block)
		return nil
	}

	if block.FilteringIncludeFilterExpr != f.IncludeProgram.code ||
		block.FilteringExcludeFilterExpr != f.ExcludeProgram.code ||
		block.FilteringSystemActionsIncludeFilterExpr != f.SystemActionsIncludeProgram.code {
		panic(fmt.Sprintf("different block filter already applied, include [applied %q, trying %q], exclude [applied %q, trying %q] and system include [applied %q, trying %q]",
			block.FilteringIncludeFilterExpr,
			f.IncludeProgram.code,
			block.FilteringExcludeFilterExpr,
			f.ExcludeProgram.code,
			block.FilteringSystemActionsIncludeFilterExpr,
			f.SystemActionsIncludeProgram.code,
		))
	}
	return nil

}

func (f *BlockFilter) transfromInPlace(block *pbcodec.Block) {
	block.FilteringApplied = true
	block.FilteringIncludeFilterExpr = f.IncludeProgram.code
	block.FilteringExcludeFilterExpr = f.ExcludeProgram.code
	block.FilteringSystemActionsIncludeFilterExpr = f.SystemActionsIncludeProgram.code

	var filteredTrxTrace []*pbcodec.TransactionTrace
	filteredExecutedInputActionCount := uint32(0)
	filteredExecutedTotalActionCount := uint32(0)

	excludedTransactionIds := map[string]bool{}
	for _, trxTrace := range block.UnfilteredTransactionTraces {
		trxTraceAddedToFiltered := false
		trxTraceExcluded := true
		for _, actTrace := range trxTrace.ActionTraces {
			passes, isSystem := f.shouldProcess(trxTrace, actTrace)
			if !passes {
				continue
			}

			actTrace.FilteringMatched = true
			actTrace.FilteringMatchedSystemActionFilter = isSystem
			filteredExecutedTotalActionCount++
			if actTrace.IsInput() {
				filteredExecutedInputActionCount++
			}

			if !trxTraceAddedToFiltered {
				filteredTrxTrace = append(filteredTrxTrace, trxTrace)
				trxTraceAddedToFiltered = true
				trxTraceExcluded = false
			}
		}

		if trxTrace.FailedDtrxTrace != nil {
			for _, actTrace := range trxTrace.FailedDtrxTrace.ActionTraces {
				passes, isSystem := f.shouldProcess(trxTrace.FailedDtrxTrace, actTrace)
				if !passes {
					continue
				}

				actTrace.FilteringMatched = true
				actTrace.FilteringMatchedSystemActionFilter = isSystem
				if !trxTraceAddedToFiltered {
					filteredTrxTrace = append(filteredTrxTrace, trxTrace)
					trxTraceAddedToFiltered = true
					trxTraceExcluded = false
				}
			}
		}

		if trxTraceExcluded {
			excludedTransactionIds[trxTrace.Id] = true
		}
	}

	var filteredTrx []*pbcodec.TransactionReceipt
	var filteredImplicitTrxOp []*pbcodec.TrxOp

	// If there is no exclusion, there is nothing to do, so just run when we have at least one exclusion
	if len(excludedTransactionIds) > 0 {
		if traceEnabled {
			zlog.Debug("filtering excluded transaction traces, let's filter out excluded one from transaction arrays", zap.Int("excluded_count", len(excludedTransactionIds)))
		}

		for _, trx := range block.UnfilteredTransactions {
			if _, isExcluded := excludedTransactionIds[trx.Id]; !isExcluded {
				filteredTrx = append(filteredTrx, trx)
			}
		}

		for _, trxOp := range block.UnfilteredImplicitTransactionOps {
			if _, isExcluded := excludedTransactionIds[trxOp.TransactionId]; !isExcluded {
				filteredImplicitTrxOp = append(filteredImplicitTrxOp, trxOp)
			}
		}

		if traceEnabled {
			zlog.Debug("filtered transactions",
				zap.Int("original_trx", len(block.UnfilteredTransactions)),
				zap.Int("original_implicit_trx", len(block.UnfilteredImplicitTransactionOps)),
				zap.Int("filtered_trx", len(filteredTrx)),
				zap.Int("filtered_implicit_trx", len(filteredImplicitTrxOp)),
			)
		}
	} else {
		filteredTrx = block.UnfilteredTransactions
		filteredImplicitTrxOp = block.UnfilteredImplicitTransactionOps
	}

	block.UnfilteredTransactions = nil
	block.UnfilteredTransactionTraces = nil
	block.UnfilteredImplicitTransactionOps = nil

	block.FilteredTransactions = filteredTrx
	block.FilteredTransactionCount = uint32(len(filteredTrx))

	block.FilteredTransactionTraces = filteredTrxTrace
	block.FilteredTransactionTraceCount = uint32(len(filteredTrxTrace))
	block.FilteredExecutedInputActionCount = filteredExecutedInputActionCount
	block.FilteredExecutedTotalActionCount = filteredExecutedTotalActionCount

	block.FilteredImplicitTransactionOps = filteredImplicitTrxOp
}

func (f *BlockFilter) shouldProcess(trxTrace *pbcodec.TransactionTrace, actTrace *pbcodec.ActionTrace) (pass bool, isSystem bool) {
	activation := actionTraceActivation{trace: actTrace, trxScheduled: trxTrace.Scheduled, trxActionCount: len(trxTrace.ActionTraces)}
	// If the include program does not match, there is nothing more to do here
	if !f.IncludeProgram.match(&activation) {
		if f.SystemActionsIncludeProgram.match(&activation) {
			return true, true
		}
		return false, false
	}

	// At this point, the inclusion expr matched, let's check it was included but should be now excluded based on the exclusion filter
	if f.ExcludeProgram.match(&activation) {
		if f.SystemActionsIncludeProgram.match(&activation) {
			return true, true
		}
		return false, false
	}

	// We are included and NOT excluded, this transaction trace/action trace match the block filter
	return true, false
}
