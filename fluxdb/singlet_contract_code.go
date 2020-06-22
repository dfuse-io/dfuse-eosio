package fluxdb

import (
	"errors"
	"fmt"

	pbcodec "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/codec/v1"
	pbfluxdb "github.com/dfuse-io/pbgo/dfuse/fluxdb/v1"
	"github.com/eoscanada/eos-go/system"
)

const codePrefix = "code"

func init() {
	RegisterSingletFactory(codePrefix, func(row *pbfluxdb.Row) Singlet {
		return ContractCodeSinglet(codePrefix + "/" + row.TabletKey)
	})
}

func NewContractCodeSinglet(contract string) ContractCodeSinglet {
	return ContractCodeSinglet(codePrefix + "/" + contract)
}

type ContractCodeSinglet string

func (t ContractCodeSinglet) Key() string {
	return string(t)
}

func (t ContractCodeSinglet) KeyAt(blockNum uint32) string {
	return string(t) + "/" + HexRevBlockNum(blockNum)
}

func (t ContractCodeSinglet) NewEntry(blockNum uint32, packedCode []byte) (*ContractCodeEntry, error) {
	_, singletKey, err := ExplodeTabletKey(string(t))
	if err != nil {
		return nil, err
	}

	return &ContractCodeEntry{
		BaseSingletEntry: BaseSingletEntry{pbfluxdb.Row{
			Collection:  codePrefix,
			TabletKey:   singletKey,
			BlockNumKey: HexRevBlockNum(blockNum),
			Payload:     packedCode,
		}},
	}, nil
}

func (t ContractCodeSinglet) NewEntryFromKV(key string, value []byte) (SingletEntry, error) {
	if len(value) != 0 && len(value) < 1 {
		return nil, errors.New("contract code entry value should have 0 bytes (deletion) or at least 1 byte (code)")
	}

	_, singletKey, blockNumKey, err := ExplodeSingletEntryKey(key)
	if err != nil {
		return nil, fmt.Errorf("unable to explode singlet entry key %q: %s", key, err)
	}

	return &ContractCodeEntry{
		BaseSingletEntry: BaseSingletEntry{pbfluxdb.Row{
			Collection:  codePrefix,
			TabletKey:   singletKey,
			BlockNumKey: blockNumKey,
			// A deletion will automatically be recorded when the payload is empty, which represents a deletion
			Payload: value,
		}},
	}, nil
}

func (t ContractCodeSinglet) String() string {
	return string(t)
}

type ContractCodeEntry struct {
	BaseSingletEntry
}

func NewContractCodeEntry(blockNum uint32, actionTrace *pbcodec.ActionTrace) (*ContractCodeEntry, error) {
	var setCode *system.SetCode
	if err := actionTrace.Action.UnmarshalData(&setCode); err != nil {
		return nil, err
	}

	return NewContractCodeSinglet(string(setCode.Account)).NewEntry(blockNum, []byte(setCode.Code))
}

func (r *ContractCodeEntry) Code() []byte {
	if r == nil {
		return nil
	}

	return r.Payload
}
