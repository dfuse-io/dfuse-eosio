package accounthist

import (
	"encoding/hex"

	"github.com/eoscanada/eos-go"
)

type Key []byte

func (k Key) String() string {
	return hex.EncodeToString(k)
}

type EOSName uint64

func (n EOSName) String() string {
	return eos.NameToString(uint64(n))
}
