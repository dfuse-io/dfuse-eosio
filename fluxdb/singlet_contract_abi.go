package fluxdb

import (
	"errors"
	"fmt"

	pbcodec "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/codec/v1"
	pbfluxdb "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/fluxdb/v1"
	"github.com/eoscanada/eos-go"
	"github.com/eoscanada/eos-go/system"
)

const abiPrefix = "abi"

func init() {
	RegisterSingletFactory(abiPrefix, func(row *pbfluxdb.TabletRow) Singlet {
		return ContractABISinglet(abiPrefix + "/" + row.TabletKey)
	})
}

func NewContractABISinglet(contract string) ContractABISinglet {
	return ContractABISinglet(abiPrefix + "/" + contract)
}

type ContractABISinglet string

func (t ContractABISinglet) Key() string {
	return string(t)
}

func (t ContractABISinglet) KeyAt(blockNum uint32) string {
	return string(t) + "/" + HexRevBlockNum(blockNum)
}

func (t ContractABISinglet) NewEntry(blockNum uint32, packedABI []byte) (*ContractABIEntry, error) {
	_, singletKey, err := ExplodeTabletKey(string(t))
	if err != nil {
		return nil, err
	}

	return &ContractABIEntry{
		BaseSingletEntry: BaseSingletEntry{pbfluxdb.TabletRow{
			Collection:  abiPrefix,
			TabletKey:   singletKey,
			BlockNumKey: HexRevBlockNum(blockNum),
			// A deletion will automatically be recorded when the payload is empty, which represents a deletion
			Payload: packedABI,
		}},
	}, nil
}

func (t ContractABISinglet) NewEntryFromKV(key string, value []byte) (SingletEntry, error) {
	if len(value) != 0 && len(value) < 1 {
		return nil, errors.New("contract abi entry value should have 0 bytes (deletion) at-least 1 byte")
	}

	_, singletKey, blockNumKey, err := ExplodeSingletEntryKey(key)
	if err != nil {
		return nil, fmt.Errorf("unable to explode singlet entry key %q: %s", key, err)
	}

	return &ContractABIEntry{
		BaseSingletEntry: BaseSingletEntry{pbfluxdb.TabletRow{
			Collection:  abiPrefix,
			TabletKey:   singletKey,
			BlockNumKey: blockNumKey,
			Payload:     value,
		}},
	}, nil
}

func (t ContractABISinglet) String() string {
	return string(t)
}

type ContractABIEntry struct {
	BaseSingletEntry
}

func NewContractABIEntry(blockNum uint32, actionTrace *pbcodec.ActionTrace) (*ContractABIEntry, error) {
	var setABI *system.SetABI
	if err := actionTrace.Action.UnmarshalData(&setABI); err != nil {
		return nil, err
	}

	return NewContractABISinglet(string(setABI.Account)).NewEntry(blockNum, []byte(setABI.ABI))
}

func (r *ContractABIEntry) ABI() (*eos.ABI, error) {
	if r == nil {
		return nil, nil
	}

	abi := new(eos.ABI)
	if err := eos.UnmarshalBinary(r.Payload, abi); err != nil {
		return nil, fmt.Errorf("unmarshal binary ABI: %w", err)
	}

	return abi, nil
}

func (r *ContractABIEntry) PackedABI() []byte {
	if r == nil {
		return nil
	}

	return r.Payload
}
