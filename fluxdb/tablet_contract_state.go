package fluxdb

import (
	"encoding/binary"
	"errors"
	"fmt"

	pbcodec "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/codec/v1"
	pbfluxdb "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/fluxdb/v1"
	"github.com/eoscanada/eos-go"
)

// Contract State
const cstPrefix = "cst"

func init() {
	RegisterTabletFactory(cstPrefix, func(row *pbfluxdb.TabletRow) Tablet {
		return ContractStateTablet(fmt.Sprintf("%s/%s", cstPrefix, row.TabletKey))
	})
}

func NewContractStateTablet(account, scope, table string) ContractStateTablet {
	return ContractStateTablet(fmt.Sprintf("%s/%s:%s:%s", cstPrefix, account, scope, table))
}

type ContractStateTablet string

func (t ContractStateTablet) Key() TabletKey {
	return TabletKey(t)
}

func (t ContractStateTablet) RowKey(blockNum uint32, primaryKey PrimaryKey) RowKey {
	return RowKey(t.RowKeyPrefix(blockNum) + "/" + string(primaryKey))
}

func (t ContractStateTablet) RowKeyPrefix(blockNum uint32) string {
	return string(t) + "/" + HexBlockNum(blockNum)
}

func (t ContractStateTablet) ReadRow(rowKey string, value []byte) (Row, error) {
	if len(value) < 8 {
		return nil, errors.New("contract state row value should have at least 8 bytes (payer)")
	}

	_, tabletKey, blockNumKey, primaryKey, err := ExplodeRowKey(rowKey)
	if err != nil {
		return nil, fmt.Errorf("unable to explode row key %q: %s", rowKey, err)
	}

	return &ContractStateRow{
		TabletRow: NewTabletRow(pbfluxdb.TabletRow{
			Collection:  cstPrefix,
			TabletKey:   tabletKey,
			BlockNumKey: blockNumKey,
			PrimKey:     primaryKey,
			Payload:     value,
		}),
	}, nil
}

func (t ContractStateTablet) String() string {
	return string(t)
}

// IndexableTablet

func (t ContractStateTablet) PrimaryKeyByteCount() int {
	return 8
}

func (t ContractStateTablet) EncodePrimaryKey(buffer []byte, primaryKey string) error {
	binary.BigEndian.PutUint64(buffer, NA(eos.Name(primaryKey)))
	return nil
}

func (t ContractStateTablet) DecodePrimaryKey(buffer []byte) (primaryKey string, err error) {
	return eos.NameToString(binary.BigEndian.Uint64(buffer)), nil
}

// Row

type ContractStateRow struct {
	TabletRow
}

func (r *ContractStateRow) Payer() string {
	return eos.NameToString(binary.BigEndian.Uint64(r.Payload))
}

func (r *ContractStateRow) RowData() []byte {
	return r.Payload[8:]
}

func NewContractStateRow(blockNum uint32, op *pbcodec.DBOp) *ContractStateRow {
	row := &ContractStateRow{
		TabletRow: NewTabletRow(pbfluxdb.TabletRow{
			Collection:  cstPrefix,
			TabletKey:   fmt.Sprintf("%s:%s:%s", op.Code, op.Scope, op.TableName),
			BlockNumKey: HexBlockNum(blockNum),
			PrimKey:     op.PrimaryKey,
		}),
	}

	if op.Operation != pbcodec.DBOp_OPERATION_REMOVE {
		row.Payload = make([]byte, len(op.NewData)+8)
		binary.BigEndian.PutUint64(row.Payload, NA(eos.Name(op.NewPayer)))
		copy(row.Payload[8:], op.NewData)
	}

	return row
}
