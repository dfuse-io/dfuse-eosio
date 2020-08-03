package statedb

import (
	"fmt"

	pbcodec "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/codec/v1"
	pbstatedb "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/statedb/v1"
	"github.com/dfuse-io/fluxdb"
	"github.com/eoscanada/eos-go"
	"github.com/eoscanada/eos-go/system"
	"github.com/golang/protobuf/proto"
)

const abiCollection = 0xA000
const abiName = "abi"

func init() {
	fluxdb.RegisterSingletFactory(abiCollection, abiName, func(identifier []byte) (fluxdb.Singlet, error) {
		if len(identifier) < 8 {
			return nil, fluxdb.ErrInvalidKeyLengthAtLeast("abi singlet identifier", 8, len(identifier))
		}

		return ContractABISinglet(identifier[0:8]), nil
	})
}

type ContractABISinglet []byte

func NewContractABISinglet(contract string) ContractABISinglet {
	return ContractABISinglet(nameToBytes(contract))
}

func (s ContractABISinglet) Collection() uint16 {
	return abiCollection
}

func (s ContractABISinglet) Identifier() []byte {
	return []byte(s)
}

func (s ContractABISinglet) Entry(height uint64, data []byte) (fluxdb.SingletEntry, error) {
	return &ContractABIEntry{baseEntry(s, height, data)}, nil
}

func (s ContractABISinglet) Contract() string {
	return bytesToName(s)
}

func (s ContractABISinglet) String() string {
	return abiName + ":" + bytesToName(s)
}

type ContractABIEntry struct {
	fluxdb.BaseSingletEntry
}

func NewContractABIEntry(blockNum uint64, actionTrace *pbcodec.ActionTrace) (entry *ContractABIEntry, err error) {
	var setABI *system.SetABI
	if err := actionTrace.Action.UnmarshalData(&setABI); err != nil {
		return nil, err
	}

	var value []byte
	if len(setABI.ABI) > 0 {
		pb := pbstatedb.ContractABIValue{RawAbi: setABI.ABI}

		if value, err = proto.Marshal(&pb); err != nil {
			return nil, fmt.Errorf("marshal proto: %w", err)
		}
	}

	singlet := ContractABISinglet(nameaToBytes(setABI.Account))
	return &ContractABIEntry{baseEntry(singlet, blockNum, value)}, nil
}

func (r *ContractABIEntry) Contract() string {
	return r.Singlet().(ContractABISinglet).Contract()
}

type ContractABIOption bool

var ContractABIPackedOnly = ContractABIOption(true)

func (r *ContractABIEntry) ABI(options ...ContractABIOption) (abi *eos.ABI, rawBytes []byte, err error) {
	if r == nil {
		return nil, nil, nil
	}

	pb := pbstatedb.ContractABIValue{}
	if err := proto.Unmarshal(r.Value(), &pb); err != nil {
		return nil, nil, err
	}

	rawABI := pb.RawAbi
	if len(options) > 0 && options[0] == ContractABIPackedOnly {
		return nil, rawABI, nil
	}

	abi = new(eos.ABI)
	if err := eos.UnmarshalBinary(rawABI, abi); err != nil {
		return nil, rawABI, errABIUnmarshal
	}

	return abi, rawABI, nil
}
