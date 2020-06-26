package blockmeta

import (
	"github.com/dfuse-io/blockmeta"
	"github.com/eoscanada/eos-go"
)

func init() {
	blockmeta.GetBlockNumFromID = blockNumFromID
}

func blockNumFromID(id string) uint64 {
	return uint64(eos.BlockNum(id))
}
