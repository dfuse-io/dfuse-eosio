package grpc

import (
	"fmt"

	"github.com/dfuse-io/dfuse-eosio/fluxdb"
	"github.com/eoscanada/eos-go"
	"google.golang.org/grpc/metadata"
)

type readScopeTableResponse struct {
	Contract string `json:"account"`
	Scope    string `json:"scope"`
	*readTableResponse
}

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

func getMetadata(upToBlockID, lastWrittenBlockID string) metadata.MD {
	md := metadata.New(map[string]string{})
	md.Set("flux-up-to-block-id", upToBlockID)
	md.Set("flux-up-to-block-num", fmt.Sprintf("%d", fluxdb.BlockNum(upToBlockID)))
	md.Set("flux-last-irreversible-block-id", lastWrittenBlockID)
	md.Set("flux-last-irreversible-block-num", fmt.Sprintf("%d", fluxdb.BlockNum(lastWrittenBlockID)))
	return md
}

type tableRow struct {
	Key      string
	Data     interface{}
	Payer    string
	BlockNum uint32
}

type onTheFlyABISerializer struct {
	abi             *eos.ABI
	abiAtBlockNum   uint32
	tableTypeName   string
	rowDataToDecode []byte
}
