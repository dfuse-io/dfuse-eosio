package fluxdb

import (
	"encoding/binary"
	"errors"
	"fmt"

	pbcodec "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/codec/v1"
	pbfluxdb "github.com/dfuse-io/pbgo/dfuse/fluxdb/v1"
	"github.com/eoscanada/eos-go"
	"github.com/eoscanada/eos-go/system"
)

const ctaPrefix = "cta"

func init() {
	RegisterTabletFactory(ctaPrefix, func(row *pbfluxdb.Row) Tablet {
		return ContractTablet(fmt.Sprintf("%s/%s", ctaPrefix, row.TabletKey))
	})
}

func NewContractTablet() ContractTablet {
	return ContractTablet(fmt.Sprintf("%s/%s", ctaPrefix, "contracts"))
}

type ContractTablet string

func (t ContractTablet) Key() string {
	return string(t)
}

func (t ContractTablet) KeyForRowAt(blockNum uint32, primaryKey string) string {
	return t.KeyAt(blockNum) + "/" + string(primaryKey)
}

func (t ContractTablet) KeyAt(blockNum uint32) string {
	return string(t) + "/" + HexBlockNum(blockNum)
}

// We actually don't use the payload, but it must be at least 1 byte because a 0 byte value represents a deletion
var ctaPayload = []byte{0x01}

func (t ContractTablet) NewRow(blockNum uint32, contract string, isDeletion bool) (*ContractRow, error) {
	_, tabletKey, err := ExplodeTabletKey(string(t))
	if err != nil {
		return nil, err
	}

	row := &ContractRow{
		BaseTabletRow: BaseTabletRow{pbfluxdb.Row{
			Collection: ctaPrefix,
			TabletKey:  tabletKey,
			HeightKey:  HexBlockNum(blockNum),
			PrimKey:    contract,
		}},
	}

	if !isDeletion {
		row.Payload = ctaPayload
	}

	return row, nil
}

func (t ContractTablet) NewRowFromKV(key string, value []byte) (TabletRow, error) {
	if len(value) != 0 && len(value) != 1 {
		return nil, errors.New("contract tablet row value should be 0 bytes (deletion), or 1 byte (contract exists)")
	}

	collection, tabletKey, heightKey, primaryKey, err := ExplodeTabletRowKey(key)
	if err != nil {
		return nil, fmt.Errorf("unable to explode tablet row key %q: %s", key, err)
	}

	return &ContractRow{
		BaseTabletRow: BaseTabletRow{pbfluxdb.Row{
			Collection: collection,
			TabletKey:  tabletKey,
			HeightKey:  heightKey,
			PrimKey:    primaryKey,
			Payload:    value,
		}},
	}, nil
}

func (t ContractTablet) String() string {
	return string(t)
}

func (t ContractTablet) PrimaryKeyByteCount() int {
	return 8
}

func (t ContractTablet) EncodePrimaryKey(buffer []byte, primaryKey string) error {
	binary.BigEndian.PutUint64(buffer, N(primaryKey))
	return nil
}

func (t ContractTablet) DecodePrimaryKey(buffer []byte) (primaryKey string, err error) {
	return eos.NameToString(binary.BigEndian.Uint64(buffer)), nil
}

// Row

type ContractRow struct {
	BaseTabletRow
}

func NewContractRow(blockNum uint32, actionTrace *pbcodec.ActionTrace) (*ContractRow, error) {
	var setCode *system.SetCode
	if err := actionTrace.Action.UnmarshalData(&setCode); err != nil {
		return nil, err
	}

	isDeletion := len(setCode.Code) <= 0
	tablet := NewContractTablet()

	return tablet.NewRow(blockNum, string(setCode.Account), isDeletion)
}

func (r *ContractRow) Contract() string {
	return r.PrimKey
}
