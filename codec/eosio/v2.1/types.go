package eosio

import (
	"github.com/eoscanada/eos-go"
	"github.com/eoscanada/eos-go/ecc"
)

// BlockState
//
// File hierarchy:
//   - https://github.com/EOSIO/eos/blob/v2.1.0/libraries/chain/include/eosio/chain/block_header_state.hpp#L57
//   - https://github.com/EOSIO/eos/blob/v2.1.0/libraries/chain/include/eosio/chain/block_header_state.hpp#L126
//   - https://github.com/EOSIO/eos/blob/v2.1.0/libraries/chain/include/eosio/chain/block_state.hpp#L10
type BlockState struct {
	// From 'struct block_header_state_common'
	BlockNum                         uint32                         `json:"block_num"`
	DPoSProposedIrreversibleBlockNum uint32                         `json:"dpos_proposed_irreversible_blocknum"`
	DPoSIrreversibleBlockNum         uint32                         `json:"dpos_irreversible_blocknum"`
	ActiveSchedule                   *eos.ProducerAuthoritySchedule `json:"active_schedule"`
	BlockrootMerkle                  *eos.MerkleRoot                `json:"blockroot_merkle,omitempty"`
	ProducerToLastProduced           []eos.PairAccountNameBlockNum  `json:"producer_to_last_produced,omitempty"`
	ProducerToLastImpliedIRB         []eos.PairAccountNameBlockNum  `json:"producer_to_last_implied_irb,omitempty"`
	ValidBlockSigningAuthorityV2     *eos.BlockSigningAuthority     `json:"valid_block_signing_authority,omitempty"`
	ConfirmCount                     []uint8                        `json:"confirm_count,omitempty"`

	// From 'struct block_header_state'
	BlockID                   eos.Checksum256                   `json:"id"`
	Header                    *eos.SignedBlockHeader            `json:"header,omitempty"`
	PendingSchedule           *eos.PendingSchedule              `json:"pending_schedule"`
	ActivatedProtocolFeatures *eos.ProtocolFeatureActivationSet `json:"activated_protocol_features,omitempty" eos:"optional"`
	AdditionalSignatures      []ecc.Signature                   `json:"additional_signatures"`

	// From 'struct block_state'
	// Type changed in v2.1.x
	SignedBlock *SignedBlock `json:"block,omitempty" eos:"optional"`
	// Validated   bool         `json:"validated"`
}

// BlockState
//
// File hierarchy:
//   - https://github.com/EOSIO/eos/blob/v2.1.0/libraries/chain/include/eosio/chain/block.hpp#L135
type SignedBlock struct {
	eos.SignedBlockHeader
	// Added in v2.1.x
	// PruneState uint8 `json:"prune_state"`
	// Type changed in v2.1.x
	Transactions    []*TransactionReceipt `json:"transactions"`
	BlockExtensions []*eos.Extension      `json:"block_extensions"`
}

// TransactionTrace
//
// File hierarchy:
//   - https://github.com/EOSIO/eos/blob/v2.1.0/libraries/chain/include/eosio/chain/trace.hpp#L51
type TransactionTrace struct {
	ID              eos.Checksum256               `json:"id"`
	BlockNum        uint32                        `json:"block_num"`
	BlockTime       eos.BlockTimestamp            `json:"block_time"`
	ProducerBlockID eos.Checksum256               `json:"producer_block_id" eos:"optional"`
	Receipt         *eos.TransactionReceiptHeader `json:"receipt,omitempty" eos:"optional"`
	Elapsed         eos.Int64                     `json:"elapsed"`
	NetUsage        eos.Uint64                    `json:"net_usage"`
	Scheduled       bool                          `json:"scheduled"`
	ActionTraces    []*ActionTrace                `json:"action_traces"`
	AccountRamDelta *AccountDelta                 `json:"account_ram_delta" eos:"optional"`
	FailedDtrxTrace *TransactionTrace             `json:"failed_dtrx_trace,omitempty" eos:"optional"`
	Except          *eos.Except                   `json:"except,omitempty" eos:"optional"`
	ErrorCode       *eos.Uint64                   `json:"error_code,omitempty" eos:"optional"`
}

// TransactionTrace
//
// File hierarchy:
//   - https://github.com/EOSIO/eos/blob/v2.1.0/libraries/chain/include/eosio/chain/trace.hpp#L22
type ActionTrace struct {
	ActionOrdinal                          eos.Varuint32           `json:"action_ordinal"`
	CreatorActionOrdinal                   eos.Varuint32           `json:"creator_action_ordinal"`
	ClosestUnnotifiedAncestorActionOrdinal eos.Varuint32           `json:"closest_unnotified_ancestor_action_ordinal"`
	Receipt                                *eos.ActionTraceReceipt `json:"receipt,omitempty" eos:"optional"`
	Receiver                               eos.AccountName         `json:"receiver"`
	Action                                 *eos.Action             `json:"act"`
	ContextFree                            bool                    `json:"context_free"`
	ElapsedUs                              eos.Int64               `json:"elapsed"`
	Console                                eos.SafeString          `json:"console"`
	TransactionID                          eos.Checksum256         `json:"trx_id"`
	BlockNum                               uint32                  `json:"block_num"`
	BlockTime                              eos.BlockTimestamp      `json:"block_time"`
	ProducerBlockID                        eos.Checksum256         `json:"producer_block_id" eos:"optional"`
	AccountRAMDeltas                       []AccountDelta          `json:"account_ram_deltas"`
	// Added in 2.1.x
	// AccountDiskDeltas []AccountDelta `json:"account_disk_deltas"`
	Except    *eos.Except `json:"except,omitempty" eos:"optional"`
	ErrorCode *eos.Uint64 `json:"error_code,omitempty" eos:"optional"`
	// Added in 2.1.x
	ReturnValue eos.HexBytes `json:"return_value"`
}

type AccountDelta struct {
	Account eos.AccountName `json:"account"`
	Delta   eos.Int64       `json:"delta"`
}

type TransactionReceipt struct {
	eos.TransactionReceiptHeader
	Transaction eos.TransactionWithID `json:"trx"`
}

var TransactionVariant = eos.NewVariantDefinition([]eos.VariantType{
	{Name: "transaction_id", Type: eos.Checksum256{}},
	{Name: "packed_transaction", Type: (*PackedTransaction)(nil)},
})

type Transaction struct {
	eos.BaseVariant
}

func (r *Transaction) UnmarshalBinary(decoder *eos.Decoder) error {
	return r.BaseVariant.UnmarshalBinaryVariant(decoder, TransactionVariant)
}

type PackedTransaction struct {
	Compression       eos.CompressionType `json:"compression"`
	PrunableData      *PrunableData       `json:"prunable_data"`
	PackedTransaction eos.HexBytes        `json:"packed_trx"`
}

var PrunableDataVariant = eos.NewVariantDefinition([]eos.VariantType{
	{Name: "full_legacy", Type: (*PackedTransactionPrunableFullLegacy)(nil)},
	{Name: "none", Type: (*PackedTransactionPrunableNone)(nil)},
	{Name: "partial", Type: (*PackedTransactionPrunablePartial)(nil)},
	{Name: "full", Type: (*PackedTransactionPrunableFull)(nil)},
})

type PackedTransactionPrunableNone struct {
	Digest eos.Checksum256 `json:"digest"`
}

type PackedTransactionPrunablePartial struct {
	Signatures          []ecc.Signature `json:"signatures"`
	ContextFreeSegments []*Segment      `json:"context_free_segments"`
}

type PackedTransactionPrunableFull struct {
	Signatures          []ecc.Signature `json:"signatures"`
	ContextFreeSegments []eos.HexBytes  `json:"context_free_segments"`
}

type PackedTransactionPrunableFullLegacy struct {
	Signatures            []ecc.Signature `json:"signatures"`
	PackedContextFreeData eos.HexBytes    `json:"packed_context_free_data"`
}

type PrunableData struct {
	eos.BaseVariant
}

func (r *PrunableData) UnmarshalBinary(decoder *eos.Decoder) error {
	return r.BaseVariant.UnmarshalBinaryVariant(decoder, PrunableDataVariant)
}

var SegmentVariant = eos.NewVariantDefinition([]eos.VariantType{
	{Name: "digest", Type: eos.Checksum256{}},
	{Name: "bytes", Type: eos.HexBytes{}},
})

type Segment struct {
	eos.BaseVariant
}

func (r *Segment) UnmarshalBinary(decoder *eos.Decoder) error {
	return r.BaseVariant.UnmarshalBinaryVariant(decoder, SegmentVariant)
}
