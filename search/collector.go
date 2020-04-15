package search

import (
	"context"
	"sort"

	bsearch "github.com/blevesearch/bleve/search"
	search "github.com/dfuse-io/search"
)

type trxResult struct {
	id       string
	blockNum uint64
}

func Collect(ctx context.Context, lowBlockNum, highBlockNum uint64, results bsearch.DocumentMatchCollection) (out []search.SearchMatch, err error) {
	trxs := make(map[string][]uint16)
	var trxList []*trxResult

	for _, el := range results {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		blockNum, trxID, actionIdx, skip := ExplodeEOSDocumentID(el.ID)
		if skip {
			continue
		}

		if blockNum < lowBlockNum || blockNum > highBlockNum {
			continue
		}

		if _, found := trxs[trxID]; !found {
			trxList = append(trxList, &trxResult{
				id:       trxID,
				blockNum: blockNum,
			})
		}

		trxs[trxID] = append(trxs[trxID], actionIdx)
	}

	for _, trx := range trxList {
		actions := trxs[trx.id]
		sort.Slice(actions, func(i, j int) bool { return actions[i] < actions[j] })

		out = append(out, &EOSSearchMatch{
			TrxIDPrefix:   trx.id,
			ActionIndexes: actions,
			BlockNumber:   trx.blockNum,
		})
	}

	return out, nil
}
