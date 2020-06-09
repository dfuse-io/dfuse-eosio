package grpc

import (
	"github.com/dfuse-io/dfuse-eosio/fluxdb"
	"github.com/eoscanada/eos-go"
)

type readTableResponse struct {
	ABI  *eos.ABI    `json:"abi"`
	Rows []*tableRow `json:"rows"`
}

type readTableRowResponse struct {
	ABI *eos.ABI  `json:"abi"`
	Row *tableRow `json:"row"`
}

type readReference struct {
	upToBlockId              string
	upToBlockNum             uint64
	lastIrreversibleBlockId  string
	lastIrreversibleBlockNum uint64
}

func newReadReference(upToBlockID, lastWrittenBlockID string) *readReference {
	return &readReference{
		upToBlockId:              upToBlockID,
		upToBlockNum:             uint64(fluxdb.BlockNum(upToBlockID)),
		lastIrreversibleBlockId:  lastWrittenBlockID,
		lastIrreversibleBlockNum: uint64(fluxdb.BlockNum(lastWrittenBlockID)),
	}
}

type tableRow struct {
	Key      string
	Data     interface{}
	Payer    string
	BlockNum uint32
}

type onTheFlyABISerializer struct {
	abi        *eos.ABI
	abiRow     *fluxdb.ABIRow
	structType string
	data       []byte
}
