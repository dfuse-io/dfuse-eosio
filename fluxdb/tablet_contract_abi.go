package fluxdb

import (
	"errors"
	"fmt"

	pbcodec "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/codec/v1"
	pbfluxdb "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/fluxdb/v1"
	"github.com/eoscanada/eos-go/system"
)

// Contract ABI
const abiPrefix = "abi"

func init() {
	RegisterTabletFactory(abiPrefix, func(row *pbfluxdb.TabletRow) Tablet {
		return ContractABITablet(abiPrefix + "/" + row.TabletKey)
	})
}

func NewContractABITablet(account string) ContractABITablet {
	return ContractABITablet(abiPrefix + "/" + account)
}

type ContractABITablet string

func (t ContractABITablet) Key() TabletKey {
	return TabletKey(t)
}

func (t ContractABITablet) RowKey(blockNum uint32, primaryKey PrimaryKey) RowKey {
	return RowKey(t.RowKeyPrefix(blockNum))
}

func (t ContractABITablet) RowKeyPrefix(blockNum uint32) string {
	return string(t) + "/" + HexRevBlockNum(blockNum)
}

func (t ContractABITablet) ReadRow(rowKey string, value []byte) (Row, error) {
	if len(value) < 0 {
		return nil, errors.New("contract abi row value should have at least 1 byte")
	}

	_, tabletKey, blockNumKey, err := ExplodeSingleRowKey(rowKey)
	if err != nil {
		return nil, fmt.Errorf("unable to explode row key %q: %s", rowKey, err)
	}

	return &ContractABIRow{
		TabletRow: NewTabletRow(pbfluxdb.TabletRow{
			Collection:  abiPrefix,
			TabletKey:   tabletKey,
			BlockNumKey: blockNumKey,
			Payload:     value,
		}),
	}, nil
}

func (t ContractABITablet) String() string {
	return string(t)
}

// SingleRowTablet

func (t ContractABITablet) SingleRowOnly() bool {
	return true
}

// Row

type ContractABIRow struct {
	TabletRow
}

func NewContractABIRow(blockNum uint32, actionTrace *pbcodec.ActionTrace) (*ContractABIRow, error) {
	var setABI *system.SetABI
	if err := actionTrace.Action.UnmarshalData(&setABI); err != nil {
		return nil, err
	}

	row := &ContractABIRow{
		TabletRow: NewTabletRow(pbfluxdb.TabletRow{
			Collection:  abiPrefix,
			TabletKey:   string(setABI.Account),
			BlockNumKey: HexRevBlockNum(blockNum),
		}),
	}

	row.Payload = []byte(setABI.ABI)

	return row, nil
}
