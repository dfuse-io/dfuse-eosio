package fluxdb

import (
	"encoding/binary"
	"fmt"

	pbcodec "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/codec/v1"
	pbfluxdb "github.com/dfuse-io/pbgo/dfuse/fluxdb/v1"
	"github.com/eoscanada/eos-go"
	"github.com/golang/protobuf/proto"
)

// Contract State
const permPrefix = "perm"

func init() {
	RegisterTabletFactory(permPrefix, func(row *pbfluxdb.Row) Tablet {
		return NewAccountPermissionsTablet(row.TabletKey)
	})
}

func NewAccountPermissionsTablet(account string) AccountPermissionsTablet {
	return AccountPermissionsTablet(permPrefix + "/" + account)
}

type AccountPermissionsTablet string

func (t AccountPermissionsTablet) Key() string {
	return string(t)
}

func (t AccountPermissionsTablet) KeyForRowAt(blockNum uint32, primaryKey string) string {
	return t.KeyAt(blockNum) + "/" + string(primaryKey)
}

func (t AccountPermissionsTablet) KeyAt(blockNum uint32) string {
	return string(t) + "/" + HexBlockNum(blockNum)
}

func (t AccountPermissionsTablet) NewRow(blockNum uint32, permissionName string, permissionObject *pbcodec.PermissionObject, isDeletion bool) (*AccountPermissionsRow, error) {
	_, tabletKey, err := ExplodeTabletKey(string(t))
	if err != nil {
		return nil, err
	}

	row := &AccountPermissionsRow{
		BaseTabletRow: BaseTabletRow{pbfluxdb.Row{
			Collection:  permPrefix,
			TabletKey:   tabletKey,
			BlockNumKey: HexBlockNum(blockNum),
			PrimKey:     permissionName,
		}},
	}

	if !isDeletion {
		row.Payload, err = proto.Marshal(permissionObject)
		if err != nil {
			return nil, fmt.Errorf("marshal permission object: %w", err)
		}
	}

	return row, nil
}

func (t AccountPermissionsTablet) NewRowFromKV(key string, value []byte) (TabletRow, error) {
	_, tabletKey, blockNumKey, primaryKey, err := ExplodeTabletRowKey(key)
	if err != nil {
		return nil, fmt.Errorf("unable to explode tablet row key %q: %s", key, err)
	}

	return &AccountPermissionsRow{
		BaseTabletRow: BaseTabletRow{pbfluxdb.Row{
			Collection:  permPrefix,
			TabletKey:   tabletKey,
			BlockNumKey: blockNumKey,
			PrimKey:     primaryKey,
			Payload:     value,
		}},
	}, nil
}

func (t AccountPermissionsTablet) String() string {
	return string(t)
}

func (t AccountPermissionsTablet) PrimaryKeyByteCount() int {
	return 8
}

func (t AccountPermissionsTablet) EncodePrimaryKey(buffer []byte, primaryKey string) error {
	binary.BigEndian.PutUint64(buffer, N(primaryKey))
	return nil
}

func (t AccountPermissionsTablet) DecodePrimaryKey(buffer []byte) (primaryKey string, err error) {
	return eos.NameToString(binary.BigEndian.Uint64(buffer)), nil
}

type AccountPermissionsRow struct {
	BaseTabletRow
}

func NewAccountPermissionsRow(blockNum uint32, op *pbcodec.PermOp) (*AccountPermissionsRow, error) {
	isDeletion := op.Operation == pbcodec.PermOp_OPERATION_REMOVE
	activePerm := op.NewPerm
	if isDeletion {
		activePerm = op.OldPerm
	}

	tablet := NewAccountPermissionsTablet(activePerm.Owner)
	return tablet.NewRow(blockNum, activePerm.Name, activePerm, isDeletion)
}

func (r *AccountPermissionsRow) PermissionObject() (*pbcodec.PermissionObject, error) {
	out := new(pbcodec.PermissionObject)
	err := proto.UnmarshalMerge(r.Payload, out)
	if err != nil {
		return nil, fmt.Errorf("unmarshal permission object: %w", err)
	}

	return out, nil
}
