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
	RegisterSigletFactory(abiPrefix, func(row *pbfluxdb.TabletRow) Siglet {
		return ContractABISiglet(abiPrefix + "/" + row.TabletKey)
	})
}

func NewContractABISiglet(contract string) ContractABISiglet {
	return ContractABISiglet(abiPrefix + "/" + contract)
}

type ContractABISiglet string

func (t ContractABISiglet) Key() string {
	return string(t)
}

func (t ContractABISiglet) KeyAt(blockNum uint32) string {
	return string(t) + "/" + HexRevBlockNum(blockNum)
}

func (t ContractABISiglet) NewEntry(blockNum uint32, packedABI []byte) (*ContractABIEntry, error) {
	_, sigletKey, err := ExplodeTabletKey(string(t))
	if err != nil {
		return nil, err
	}

	return &ContractABIEntry{
		BaseSigletEntry: BaseSigletEntry{pbfluxdb.TabletRow{
			Collection:  abiPrefix,
			TabletKey:   sigletKey,
			BlockNumKey: HexRevBlockNum(blockNum),
			Payload:     packedABI,
		}},
	}, nil
}

func (t ContractABISiglet) NewEntryFromKV(key string, value []byte) (SigletEntry, error) {
	if len(value) < 0 {
		return nil, errors.New("contract abi entry value should have at least 1 byte")
	}

	_, sigletKey, blockNumKey, err := ExplodeSigletEntryKey(key)
	if err != nil {
		return nil, fmt.Errorf("unable to explode siglet entry key %q: %s", key, err)
	}

	return &ContractABIEntry{
		BaseSigletEntry: BaseSigletEntry{pbfluxdb.TabletRow{
			Collection:  abiPrefix,
			TabletKey:   sigletKey,
			BlockNumKey: blockNumKey,
			Payload:     value,
		}},
	}, nil
}

func (t ContractABISiglet) String() string {
	return string(t)
}

type ContractABIEntry struct {
	BaseSigletEntry
}

func NewContractABIEntry(blockNum uint32, actionTrace *pbcodec.ActionTrace) (*ContractABIEntry, error) {
	var setABI *system.SetABI
	if err := actionTrace.Action.UnmarshalData(&setABI); err != nil {
		return nil, err
	}

	return NewContractABISiglet(string(setABI.Account)).NewEntry(blockNum, []byte(setABI.ABI))
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
