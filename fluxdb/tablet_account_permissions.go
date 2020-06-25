package fluxdb

// import (
// 	"encoding/binary"
// 	"errors"
// 	"fmt"
// 	"strings"

// 	pbcodec "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/codec/v1"
// 	pbfluxdb "github.com/dfuse-io/pbgo/dfuse/fluxdb/v1"
// 	"github.com/eoscanada/eos-go"
// )

// // Contract State
// const permPrefix = "perm"

// func init() {
// 	RegisterTabletFactory(permPrefix, func(row *pbfluxdb.Row) Tablet {
// 		return AccountPermissionsTablet(fmt.Sprintf("%s/%s", permPrefix, row.TabletKey))
// 	})
// }

// func NewAccountPermissionsTablet(contract, scope, table string) AccountPermissionsTablet {
// 	return AccountPermissionsTablet(fmt.Sprintf("%s/%s:%s:%s", permPrefix, contract, scope, table))
// }

// type AccountPermissionsTablet string

// func (t AccountPermissionsTablet) Key() string {
// 	return string(t)
// }

// func (t AccountPermissionsTablet) Explode() (collection, contract, scope, table string) {
// 	segments := strings.Split(string(t), "/")
// 	tabletParts := strings.Split(segments[1], ":")

// 	return segments[0], tabletParts[0], tabletParts[1], tabletParts[2]
// }

// func (t AccountPermissionsTablet) KeyForRowAt(blockNum uint32, primaryKey string) string {
// 	return t.KeyAt(blockNum) + "/" + string(primaryKey)
// }

// func (t AccountPermissionsTablet) KeyAt(blockNum uint32) string {
// 	return string(t) + "/" + HexBlockNum(blockNum)
// }

// func (t AccountPermissionsTablet) NewRow(blockNum uint32, permissionName string, permissionObject *pbcodec.PermissionObject) (*AccountPermissionsRow, error) {
// 	_, tabletKey, err := ExplodeTabletKey(string(t))
// 	if err != nil {
// 		return nil, err
// 	}

// 	row := &AccountPermissionsRow{
// 		BaseTabletRow: BaseTabletRow{pbfluxdb.Row{
// 			Collection: permPrefix,
// 			TabletKey:  tabletKey,
// 			HeightKey:  HexBlockNum(blockNum),
// 			PrimKey:    primaryKey,
// 		}},
// 	}

// 	if !isDeletion {
// 		row.Payload = make([]byte, len(data)+8)
// 		binary.BigEndian.PutUint64(row.Payload, N(payer))
// 		copy(row.Payload[8:], data)
// 	}

// 	return row, nil
// }

// func (t AccountPermissionsTablet) NewRowFromKV(key string, value []byte) (TabletRow, error) {
// 	if len(value) != 0 && len(value) < 8 {
// 		return nil, errors.New("contract state tablet row value should have 0 bytes (deletion) or at least 8 bytes (payer)")
// 	}

// 	_, tabletKey, blockNumKey, primaryKey, err := ExplodeTabletRowKey(key)
// 	if err != nil {
// 		return nil, fmt.Errorf("unable to explode tablet row key %q: %s", key, err)
// 	}

// 	return &AccountPermissionsRow{
// 		BaseTabletRow: BaseTabletRow{pbfluxdb.Row{
// 			Collection: permPrefix,
// 			TabletKey:  tabletKey,
// 			HeightKey:  blockNumKey,
// 			PrimKey:    primaryKey,
// 			Payload:    value,
// 		}},
// 	}, nil
// }

// func (t AccountPermissionsTablet) String() string {
// 	return string(t)
// }

// // IndexableTablet

// func (t AccountPermissionsTablet) PrimaryKeyByteCount() int {
// 	return 8
// }

// func (t AccountPermissionsTablet) EncodePrimaryKey(buffer []byte, primaryKey string) error {
// 	binary.BigEndian.PutUint64(buffer, N(primaryKey))
// 	return nil
// }

// func (t AccountPermissionsTablet) DecodePrimaryKey(buffer []byte) (primaryKey string, err error) {
// 	return eos.NameToString(binary.BigEndian.Uint64(buffer)), nil
// }

// // Row

// type AccountPermissionsRow struct {
// 	BaseTabletRow
// }

// func NewAccountPermissionsRow(blockNum uint32, op *pbcodec.DBOp) (*AccountPermissionsRow, error) {
// 	tablet := NewAccountPermissionsTablet(op.Code, op.Scope, op.TableName)
// 	isDeletion := op.Operation == pbcodec.DBOp_OPERATION_REMOVE

// 	var payer string
// 	var data []byte
// 	if !isDeletion {
// 		payer = op.NewPayer
// 		data = op.NewData
// 	}

// 	return tablet.NewRow(blockNum, op.PrimaryKey, payer, data, isDeletion)
// }

// func (r *AccountPermissionsRow) Payer() string {
// 	return eos.NameToString(binary.BigEndian.Uint64(r.Payload))
// }

// func (r *AccountPermissionsRow) Data() []byte {
// 	return r.Payload[8:]
// }
