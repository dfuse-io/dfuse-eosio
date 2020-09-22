package injector

import (
	"github.com/dfuse-io/dfuse-eosio/accounthist"
)

func (i *Injector) UpdateSeqData(key accounthist.Facet, seqData accounthist.SequenceData) {
	i.cacheSeqData[key.String()] = seqData
}
