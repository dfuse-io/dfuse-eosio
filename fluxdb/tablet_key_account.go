package fluxdb

import (
	"errors"
	"fmt"
	"strings"

	pbfluxdb "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/fluxdb/v1"
)

// Key Account (keeps a mapping from EOS Public Key to []Account)

const kaPrefix = "ka"

func init() {
	RegisterTabletFactory(kaPrefix, func(row *pbfluxdb.TabletRow) Tablet {
		return KeyAccountTablet(fmt.Sprintf("%s/%s", kaPrefix, row.TabletKey))
	})
}

func NewKeyAccountTablet(publicKey string) KeyAccountTablet {
	return KeyAccountTablet(fmt.Sprintf("%s/%s", kaPrefix, publicKey))
}

type KeyAccountTablet string

func (t KeyAccountTablet) Key() string {
	return string(t)
}

func (t KeyAccountTablet) Explode() (collection, publicKey string) {
	segments := strings.Split(string(t), "/")
	return segments[0], segments[1]
}

func (t KeyAccountTablet) KeyForRowAt(blockNum uint32, primaryKey string) string {
	return t.KeyAt(blockNum) + "/" + string(primaryKey)
}

func (t KeyAccountTablet) KeyAt(blockNum uint32) string {
	return string(t) + "/" + HexBlockNum(blockNum)
}

// We actually don't use the payload, but it must be at least 1 byte because a 0 byte value represents a deletion
var kaPayload = []byte{0x01}

func (t KeyAccountTablet) NewRow(blockNum uint32, account, permission string, isDeletion bool) (*KeyAccountRow, error) {
	_, tabletKey, err := ExplodeTabletKey(string(t))
	if err != nil {
		return nil, err
	}

	row := &KeyAccountRow{
		BaseTabletRow: BaseTabletRow{pbfluxdb.TabletRow{
			Collection:  kaPrefix,
			TabletKey:   tabletKey,
			BlockNumKey: HexBlockNum(blockNum),
			PrimKey:     account + ":" + permission,
		}},
	}

	if !isDeletion {
		row.Payload = kaPayload
	}

	return row, nil
}

func (t KeyAccountTablet) NewRowFromKV(key string, value []byte) (TabletRow, error) {
	if len(value) != 0 && len(value) != 1 {
		return nil, errors.New("auth link tablet row value should have at exactly 0 bytes (deletion) or 1 byte (existence)")
	}

	_, tabletKey, blockNumKey, primaryKey, err := ExplodeTabletRowKey(key)
	if err != nil {
		return nil, fmt.Errorf("unable to explode tablet row key %q: %s", key, err)
	}

	return &KeyAccountRow{
		BaseTabletRow: BaseTabletRow{pbfluxdb.TabletRow{
			Collection:  kaPrefix,
			TabletKey:   tabletKey,
			BlockNumKey: blockNumKey,
			PrimKey:     primaryKey,
			Payload:     value,
		}},
	}, nil
}

func (t KeyAccountTablet) String() string {
	return string(t)
}

func (t KeyAccountTablet) PrimaryKeyByteCount() int {
	return 16
}

var keyAccountPKDecoder = twoUint64PrimaryKeyReaderFactory("key account tablet row")
var keyAccountPKEncoder = twoUint64PrimaryKeyWriterFactory("key account tablet row")

func (t KeyAccountTablet) EncodePrimaryKey(buffer []byte, primaryKey string) error {
	return authLinkPKEncoder(primaryKey, buffer)
}

func (t KeyAccountTablet) DecodePrimaryKey(buffer []byte) (primaryKey string, err error) {
	return authLinkPKDecoder(buffer)
}

// Row

type KeyAccountRow struct {
	BaseTabletRow
}

func NewKeyAccountRow(blockNum uint32, publicKey, account, permission string, isDeletion bool) (*KeyAccountRow, error) {
	tablet := NewKeyAccountTablet(publicKey)
	return tablet.NewRow(blockNum, account, permission, isDeletion)
}

func (r *KeyAccountRow) Account() string {
	account, _ := r.Explode()
	return account
}

func (r *KeyAccountRow) Explode() (account, permission string) {
	parts := strings.Split(r.PrimKey, ":")
	if len(parts) > 0 {
		account = parts[0]
	}

	if len(parts) > 1 {
		permission = parts[1]
	}

	return
}
