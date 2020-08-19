package statedb

import (
	"fmt"

	pbcodec "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/codec/v1"
	pbstatedb "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/statedb/v1"
	"github.com/dfuse-io/fluxdb"
	"github.com/eoscanada/eos-go"
	"github.com/golang/protobuf/proto"
)

const cstCollection = 0xB000
const cstPrefix = "cst"

func init() {
	fluxdb.RegisterTabletFactory(cstCollection, cstPrefix, func(identifier []byte) (fluxdb.Tablet, error) {
		if len(identifier) < 24 {
			return nil, fluxdb.ErrInvalidKeyLengthAtLeast("contract state tablet identifier", 24, len(identifier))
		}

		return ContractStateTablet(identifier[0:24]), nil
	})
}

func NewContractStateTablet(contract, table string, scope string) ContractStateTablet {
	return ContractStateTablet(nameToBytes(contract, table, scope))
}

type ContractStateTablet []byte

func (t ContractStateTablet) Collection() uint16 {
	return cstCollection
}

func (t ContractStateTablet) Identifier() []byte {
	return t
}

func (t ContractStateTablet) Row(height uint64, primaryKey []byte, data []byte) (fluxdb.TabletRow, error) {
	if len(primaryKey) != 8 {
		return nil, fluxdb.ErrInvalidKeyLength("contract state primary key", 8, len(primaryKey))
	}

	return &ContractStateRow{baseRow(t, height, primaryKey, data)}, nil
}

func (t ContractStateTablet) Explode() (contract, table, scope string) {
	return bytesToName3(t)
}

func (t ContractStateTablet) String() string {
	return cstPrefix + ":" + bytesToJoinedName3(t)
}

type ContractStateRow struct {
	fluxdb.BaseTabletRow
}

func NewContractStateRow(blockNum uint64, op *pbcodec.DBOp) (row *ContractStateRow, err error) {
	var value []byte
	if op.Operation != pbcodec.DBOp_OPERATION_REMOVE {
		pb := pbstatedb.ContractStateValue{
			Payer: eos.MustStringToName(op.NewPayer),
			Data:  op.NewData,
		}

		if value, err = proto.Marshal(&pb); err != nil {
			return nil, fmt.Errorf("marshal proto: %w", err)
		}
	}

	tablet := NewContractStateTablet(op.Code, op.TableName, op.Scope)
	return &ContractStateRow{baseRow(tablet, blockNum, nameToBytes(op.PrimaryKey), value)}, nil
}

func (r *ContractStateRow) Info() (payer string, rowData []byte, err error) {
	pb := pbstatedb.ContractStateValue{}
	if err := proto.Unmarshal(r.Value(), &pb); err != nil {
		return "", nil, err
	}

	return eos.NameToString(pb.Payer), pb.Data, nil
}

func (r *ContractStateRow) String() string {
	return r.Stringify(bytesToName(r.PrimaryKey()))
}

type ContractStatePrimaryKey []byte

func (k ContractStatePrimaryKey) Bytes() []byte  { return k }
func (k ContractStatePrimaryKey) String() string { return bytesToName(k) }
