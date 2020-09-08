package filtering

import (
	"container/heap"
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

type kv struct {
	Key   string
	Value int
}

type actorMap map[string]int

func (m actorMap) add(actor string) {
	if elem, ok := m[actor]; ok {
		elem++
		return
	}
	m[actor] = 1
}

func getHeap(m map[string]int) *KVHeap {
	h := &KVHeap{}
	heap.Init(h)
	for k, v := range m {
		heap.Push(h, kv{k, v})
	}
	return h
}

type KVHeap []kv

func (h KVHeap) Len() int           { return len(h) }
func (h KVHeap) Less(i, j int) bool { return h[i].Value > h[j].Value }
func (h KVHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }

func (h *KVHeap) Push(x interface{}) {
	*h = append(*h, x.(kv))
}

func (h *KVHeap) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}

func getTop5ActorsForTrx(trx *pbcodec.TransactionTrace) (topActors []string) {
	var actors actorMap
	actors = make(map[string]int)
	for _, action := range trx.ActionTraces {
		actors.add(action.Receiver)
		actors.add(action.Account())
		for _, auth := range action.Action.Authorization {
			actors.add(auth.Actor)
		}
	}
	kvHeap := getHeap(actors)
	for i := 0; i < 5; i++ {
		if kvHeap.Len() == 0 {
			break
		}
		act := kvHeap.Pop()
		topActors = append(topActors, act.(kv).Key)
	}
	return
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
		var trxTop5Actors []string //per transaction
		getTrxTop5Actors := func() []string {
			if trxTop5Actors == nil {
				trxTop5Actors = getTop5ActorsForTrx(trxTrace)
			}
			return trxTop5Actors
		}

		for _, actTrace := range trxTrace.ActionTraces {
			passes, isSystem := f.shouldProcess(trxTrace, actTrace, getTrxTop5Actors)
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
			trxTop5Actors = nil
			getTrxTop5Actors = func() []string {
				if trxTop5Actors == nil {
					trxTop5Actors = getTop5ActorsForTrx(trxTrace.FailedDtrxTrace)
				}
				return trxTop5Actors
			}
			for _, actTrace := range trxTrace.FailedDtrxTrace.ActionTraces {
				passes, isSystem := f.shouldProcess(trxTrace.FailedDtrxTrace, actTrace, getTrxTop5Actors)
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

func (f *BlockFilter) shouldProcess(trxTrace *pbcodec.TransactionTrace, actTrace *pbcodec.ActionTrace, trxTop5ActorsGetter func() []string) (pass bool, isSystem bool) {
	activation := actionTraceActivation{trace: actTrace, trxScheduled: trxTrace.Scheduled, trxActionCount: len(trxTrace.ActionTraces), trxTop5ActorsGetter: trxTop5ActorsGetter}
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

func (f *BlockFilter) String() string {
	return fmt.Sprintf("[include: %s, exclude: %s, system: %s]", f.IncludeProgram.code, f.ExcludeProgram.code, f.SystemActionsIncludeProgram.code)
}
