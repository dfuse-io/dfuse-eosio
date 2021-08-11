package search

import (
	"strings"

	"github.com/streamingfast/bstream"
	pbcodec "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/codec/v1"
	pbsearcheos "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/search/v1"
	pbsearch "github.com/streamingfast/pbgo/dfuse/search/v1"
	"github.com/golang/protobuf/ptypes"
)

type SearchMatch struct {
	TrxIDPrefix   string   `json:"prefix"` // ID prefix
	ActionIndexes []uint16 `json:"acts"`   // Action indexes within the transactions
	BlockNumber   uint64   `json:"blk"`    // Current block for this trx
	Index         uint64   `json:"idx"`    // Index of the matching transaction within a block (depends on order of sort)
}

func (m *SearchMatch) BlockNum() uint64 {
	return m.BlockNumber
}

func (m *SearchMatch) GetIndex() uint64 {
	return m.Index
}

func (m *SearchMatch) TransactionIDPrefix() string {
	return m.TrxIDPrefix
}

func (m *SearchMatch) SetIndex(index uint64) {
	m.Index = index
}

func (m *SearchMatch) FillProtoSpecific(match *pbsearch.SearchMatch, block *bstream.Block) (err error) {
	eosMatch := &pbsearcheos.Match{}

	if block != nil {
		eosMatch.Block = m.buildBlockTrxPayload(block)
		if m.TrxIDPrefix == "" {
			match.ChainSpecific, err = ptypes.MarshalAny(eosMatch)
			return err
		}
	}

	eosMatch.ActionIndexes = uint16to32s(m.ActionIndexes)

	match.ChainSpecific, err = ptypes.MarshalAny(eosMatch)
	return err
}

func (m *SearchMatch) buildBlockTrxPayload(block *bstream.Block) *pbsearcheos.BlockTrxPayload {
	blk := block.ToNative().(*pbcodec.Block)

	if m.TrxIDPrefix == "" {
		return &pbsearcheos.BlockTrxPayload{
			BlockHeader: blk.Header,
			BlockID:     blk.ID(),
		}
	}

	for _, trx := range blk.TransactionTraces() {
		fullTrxID := trx.Id
		if !strings.HasPrefix(fullTrxID, m.TrxIDPrefix) {
			continue
		}

		out := &pbsearcheos.BlockTrxPayload{}
		out.BlockHeader = blk.Header
		out.BlockID = blk.Id
		out.Trace = trx
		return out
	}

	// FIXME (MATT): Is this even possible?
	return nil
}
