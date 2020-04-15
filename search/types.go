package search

import (
	"strings"

	"github.com/dfuse-io/bstream"
	pbdeos "github.com/dfuse-io/pbgo/dfuse/codecs/deos"
	pbsearch "github.com/dfuse-io/pbgo/dfuse/search/v1"
	"github.com/dfuse-io/search"
)

func init() {
	search.GetSearchMatchFactory = func() search.SearchMatch {
		return &EOSSearchMatch{}
	}
}

type EOSSearchMatch struct {
	TrxIDPrefix   string   `json:"prefix"` // ID prefix
	ActionIndexes []uint16 `json:"acts"`   // Action indexes within the transactions
	BlockNumber   uint64   `json:"blk"`    // Current block for this trx
	Index         uint64   `json:"idx"`    // Index of the matching transaction within a block (depends on order of sort)
}

func (m *EOSSearchMatch) BlockNum() uint64 {
	return m.BlockNumber
}

func (m *EOSSearchMatch) GetIndex() uint64 {
	return m.Index
}

func (m *EOSSearchMatch) TransactionIDPrefix() string {
	return m.TrxIDPrefix
}

func (m *EOSSearchMatch) SetIndex(index uint64) {
	m.Index = index
}

func (m *EOSSearchMatch) FillProtoSpecific(match *pbsearch.SearchMatch, block *bstream.Block) error {
	eosMatch := &pbsearch.EOSMatch{}
	match.Specific = &pbsearch.SearchMatch_Eos{
		Eos: eosMatch,
	}

	if block != nil {
		eosMatch.Block = m.buildBlockTrxPayload(block)
		if m.TrxIDPrefix == "" {
			return nil
		}
	}

	eosMatch.ActionIndexes = uint16to32s(m.ActionIndexes)

	return nil
}

func (m *EOSSearchMatch) buildBlockTrxPayload(block *bstream.Block) *pbsearch.EOSBlockTrxPayload {
	blk := block.ToNative().(*pbdeos.Block)

	if m.TrxIDPrefix == "" {
		return &pbsearch.EOSBlockTrxPayload{
			BlockHeader: blk.Header,
			BlockID:     blk.ID(),
		}
	}

	for _, trx := range blk.TransactionTraces {
		fullTrxID := trx.Id
		if !strings.HasPrefix(fullTrxID, m.TrxIDPrefix) {
			continue
		}

		out := &pbsearch.EOSBlockTrxPayload{}
		out.BlockHeader = blk.Header
		out.BlockID = blk.Id
		out.Trace = trx
		return out
	}

	// FIXME (MATT): Is this even possible?
	return nil
}
