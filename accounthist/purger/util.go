package purger

import "github.com/eoscanada/eos-go"

type EOSName uint64

func (n EOSName) String() string {
	return eos.NameToString(uint64(n))
}
