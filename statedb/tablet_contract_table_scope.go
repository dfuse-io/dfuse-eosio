package statedb

import (
	"fmt"

	pbcodec "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/codec/v1"
	pbstatedb "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/statedb/v1"
	"github.com/eoscanada/eos-go"
	"github.com/golang/protobuf/proto"
	"github.com/streamingfast/fluxdb"
)

const ctscpCollection = 0xB200
const ctscpPrefix = "ctscp"

func init() {
	fluxdb.RegisterTabletFactory(ctscpCollection, ctscpPrefix, func(identifier []byte) (fluxdb.Tablet, error) {
		if len(identifier) < 16 {
			return nil, fluxdb.ErrInvalidKeyLengthAtLeast("contract table scope tablet identifier", 16, len(identifier))
		}

		return ContractTableScopeTablet(identifier[0:16]), nil
	})
}

func NewContractTableScopeTablet(contract, table string) ContractTableScopeTablet {
	return ContractTableScopeTablet(standardNameToBytes(contract, table))
}

type ContractTableScopeTablet []byte

func (t ContractTableScopeTablet) Collection() uint16 {
	return ctscpCollection
}

func (t ContractTableScopeTablet) Identifier() []byte {
	return t
}

func (t ContractTableScopeTablet) Row(height uint64, primaryKey []byte, data []byte) (fluxdb.TabletRow, error) {
	if len(primaryKey) != 8 {
		return nil, fluxdb.ErrInvalidKeyLength("contract table scope primary key", 8, len(primaryKey))
	}

	return &ContractTableScopeRow{baseRow(t, height, primaryKey, data)}, nil
}

func (t ContractTableScopeTablet) String() string {
	return ctscpPrefix + ":" + bytesToJoinedName2(t)
}

type ContractTableScopeRow struct {
	fluxdb.BaseTabletRow
}

func NewContractTableScopeRow(blockNum uint64, op *pbcodec.TableOp) (row *ContractTableScopeRow, err error) {
	var value []byte
	if op.Operation != pbcodec.TableOp_OPERATION_REMOVE {
		pb := pbstatedb.ContractTableScopeValue{Payer: eos.MustStringToName(op.Payer)}

		if value, err = proto.Marshal(&pb); err != nil {
			return nil, fmt.Errorf("marshal proto: %w", err)
		}
	}

	tablet := NewContractTableScopeTablet(op.Code, op.TableName)
	return &ContractTableScopeRow{baseRow(tablet, blockNum, standardNameToBytes(op.Scope), value)}, nil
}

func (r *ContractTableScopeRow) Scope() string {
	return bytesToName(r.PrimaryKey())
}

func (r *ContractTableScopeRow) Payer() (string, error) {
	pb := pbstatedb.ContractTableScopeValue{}
	if err := proto.Unmarshal(r.Value(), &pb); err != nil {
		return "", err
	}

	return eos.NameToString(pb.Payer), nil
}

func (r *ContractTableScopeRow) String() string {
	return r.Stringify(bytesToName(r.PrimaryKey()))
}
