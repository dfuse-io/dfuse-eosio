package injector

import (
	"github.com/dfuse-io/dfuse-eosio/accounthist"
)

func (i *Injector) UpdateSeqData(key accounthist.ActionKey, seqData accounthist.SequenceData) {
	i.cacheSeqData[key.String()] = seqData
}
