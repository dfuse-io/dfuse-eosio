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

const actPrefix = "act"

func init() {
	RegisterTabletFactory(actPrefix, func(row *pbfluxdb.Row) Tablet {
		return AccountsTablet(fmt.Sprintf("%s/%s", actPrefix, row.TabletKey))
	})
}

func NewAccountsTablet() AccountsTablet {
	return AccountsTablet(fmt.Sprintf("%s/%s", actPrefix, "accounts"))
}

type AccountsTablet string

func (t AccountsTablet) Key() string {
	return string(t)
}

func (t AccountsTablet) KeyForRowAt(blockNum uint32, primaryKey string) string {
	return t.KeyAt(blockNum) + "/" + string(primaryKey)
}

func (t AccountsTablet) KeyAt(blockNum uint32) string {
	return string(t) + "/" + HexBlockNum(blockNum)
}

// We actually don't use the payload, but it must be at least 1 byte because a 0 byte value represents a deletion
var actPayload = []byte{0x01}

func (t AccountsTablet) NewRow(blockNum uint32, contract string, isDeletion bool) (*AccountsRow, error) {
	_, tabletKey, err := ExplodeTabletKey(string(t))
	if err != nil {
		return nil, err
	}

	row := &AccountsRow{
		BaseTabletRow: BaseTabletRow{pbfluxdb.Row{
			Collection: actPrefix,
			TabletKey:  tabletKey,
			HeightKey:  HexBlockNum(blockNum),
			PrimKey:    contract,
		}},
	}

	if !isDeletion {
		row.Payload = actPayload
	}

	return row, nil
}

func (t AccountsTablet) NewRowFromKV(key string, value []byte) (TabletRow, error) {
	if len(value) != 0 && len(value) != 1 {
		return nil, errors.New("contract tablet row value should be 0 bytes (deletion), or 1 byte (account exists)")
	}

	collection, tabletKey, heightKey, primaryKey, err := ExplodeTabletRowKey(key)
	if err != nil {
		return nil, fmt.Errorf("unable to explode tablet row key %q: %s", key, err)
	}

	return &AccountsRow{
		BaseTabletRow: BaseTabletRow{pbfluxdb.Row{
			Collection: collection,
			TabletKey:  tabletKey,
			HeightKey:  heightKey,
			PrimKey:    primaryKey,
			Payload:    value,
		}},
	}, nil
}

func (t AccountsTablet) String() string {
	return string(t)
}

func (t AccountsTablet) PrimaryKeyByteCount() int {
	return 8
}

func (t AccountsTablet) EncodePrimaryKey(buffer []byte, primaryKey string) error {
	binary.BigEndian.PutUint64(buffer, N(primaryKey))
	return nil
}

func (t AccountsTablet) DecodePrimaryKey(buffer []byte) (primaryKey string, err error) {
	return eos.NameToString(binary.BigEndian.Uint64(buffer)), nil
}

// Row

type AccountsRow struct {
	BaseTabletRow
}

func NewAccountsRow(blockNum uint32, actionTrace *pbcodec.ActionTrace) (*AccountsRow, error) {
	var newAccount *system.NewAccount
	if err := actionTrace.Action.UnmarshalData(&newAccount); err != nil {
		return nil, err
	}

	return NewAccountsTablet().NewRow(blockNum, string(newAccount.Name), false)
}

func (r *AccountsRow) Account() string {
	return r.PrimKey
}
