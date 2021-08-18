package search

import (
	"context"
	"sort"
	"strings"

	bsearch "github.com/blevesearch/bleve/search"
	"github.com/streamingfast/search"
	"go.uber.org/zap"
)

type testTrxResult struct {
	id       string
	blockNum uint64
}

var TestMatchCollector = func(ctx context.Context, lowBlockNum, highBlockNum uint64, results bsearch.DocumentMatchCollection) (out []search.SearchMatch, err error) {
	trxs := make(map[string][]uint16)
	var trxList []*testTrxResult

	for _, el := range results {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		blockNum, trxID, actionIdx, skip := testExplodeDocumentID(el.ID)
		if skip {
			continue
		}

		if blockNum < lowBlockNum || blockNum > highBlockNum {
			continue
		}

		if _, found := trxs[trxID]; !found {
			trxList = append(trxList, &testTrxResult{
				id:       trxID,
				blockNum: blockNum,
			})
		}

		trxs[trxID] = append(trxs[trxID], actionIdx)
	}

	for _, trx := range trxList {
		actions := trxs[trx.id]
		sort.Slice(actions, func(i, j int) bool { return actions[i] < actions[j] })

		out = append(out, &SearchMatch{
			TrxIDPrefix:   trx.id,
			ActionIndexes: actions,
			BlockNumber:   trx.blockNum,
		})
	}

	return out, nil
}

func testExplodeDocumentID(ref string) (blockNum uint64, trxID string, actionIdx uint16, skip bool) {
	var err error
	chunks := strings.Split(ref, ":")
	chunksCount := len(chunks)
	if chunksCount != 3 || chunks[0] == "meta" { // meta, flatten, etc.
		skip = true
		return
	}

	blockNum32, err := fromHexUint32(chunks[0])
	if err != nil {
		zlog.Panic("woah, block num invalid?", zap.Error(err))
	}

	blockNum = uint64(blockNum32)

	trxID = chunks[1]
	actionIdx, err = fromHexUint16(chunks[2])
	if err != nil {
		zlog.Panic("woah, action index invalid?", zap.Error(err))
	}

	return
}
