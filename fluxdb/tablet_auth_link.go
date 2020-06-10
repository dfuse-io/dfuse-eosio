package fluxdb

import (
	"encoding/binary"
	"errors"
	"fmt"
	"strings"

	pbcodec "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/codec/v1"
	pbfluxdb "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/fluxdb/v1"
	"github.com/eoscanada/eos-go"
	"github.com/eoscanada/eos-go/system"
)

// Permission Authority link
const alPrefix = "al"

func init() {
	RegisterTabletFactory(alPrefix, func(row *pbfluxdb.TabletRow) Tablet {
		return AuthLinkTablet(fmt.Sprintf("%s/%s", alPrefix, row.TabletKey))
	})
}

func NewAuthLinkTablet(account string) AuthLinkTablet {
	return AuthLinkTablet(fmt.Sprintf("%s/%s", alPrefix, account))
}

type AuthLinkTablet string

func (t AuthLinkTablet) Key() string {
	return string(t)
}

func (t AuthLinkTablet) Explode() (collection, account string) {
	segments := strings.Split(string(t), "/")
	return segments[0], segments[1]
}

func (t AuthLinkTablet) KeyForRowAt(blockNum uint32, primaryKey string) string {
	return t.KeyAt(blockNum) + "/" + string(primaryKey)
}

func (t AuthLinkTablet) KeyAt(blockNum uint32) string {
	return string(t) + "/" + HexBlockNum(blockNum)
}

func (t AuthLinkTablet) NewRow(blockNum uint32, contract, action, permission string, isDeletion bool) (*AuthLinkRow, error) {
	_, tabletKey, err := ExplodeTabletKey(string(t))
	if err != nil {
		return nil, err
	}

	row := &AuthLinkRow{
		BaseTabletRow: BaseTabletRow{pbfluxdb.TabletRow{
			Collection:  alPrefix,
			TabletKey:   tabletKey,
			BlockNumKey: HexBlockNum(blockNum),
			PrimKey:     contract + ":" + action,
		}},
	}

	if !isDeletion {
		row.Payload = make([]byte, 8)
		binary.BigEndian.PutUint64(row.Payload, NA(eos.Name(permission)))
	}

	return row, nil
}

func (t AuthLinkTablet) NewRowFromKV(key string, value []byte) (TabletRow, error) {
	if len(value) == 0 || len(value) == 8 {
		return nil, errors.New("auth link tablet row value should have at exactly 0 byte (deletion) or 8 bytes (permission)")
	}

	collection, tabletKey, blockNumKey, primaryKey, err := ExplodeTabletRowKey(key)
	if err != nil {
		return nil, fmt.Errorf("unable to explode tablet row key %q: %s", key, err)
	}

	return &AuthLinkRow{
		BaseTabletRow: BaseTabletRow{pbfluxdb.TabletRow{
			Collection:  collection,
			TabletKey:   tabletKey,
			BlockNumKey: blockNumKey,
			PrimKey:     primaryKey,
			Payload:     value,
		}},
	}, nil
}

func (t AuthLinkTablet) String() string {
	return string(t)
}

func (t AuthLinkTablet) PrimaryKeyByteCount() int {
	return 16
}

var authLinkPKDecoder = twoUint64PrimaryKeyReaderFactory("auth link tablet row")
var authLinkPKEncoder = twoUint64PrimaryKeyWriterFactory("auth link tablet row")

func (t AuthLinkTablet) EncodePrimaryKey(buffer []byte, primaryKey string) error {
	return authLinkPKEncoder(primaryKey, buffer)
}

func (t AuthLinkTablet) DecodePrimaryKey(buffer []byte) (primaryKey string, err error) {
	return authLinkPKDecoder(buffer)
}

// Row

type AuthLinkRow struct {
	BaseTabletRow
}

func NewInsertAuthLinkRow(blockNum uint32, actionTrace *pbcodec.ActionTrace) (*AuthLinkRow, error) {
	var linkAuth *system.LinkAuth
	if err := actionTrace.Action.UnmarshalData(&linkAuth); err != nil {
		return nil, err
	}

	tablet := NewAuthLinkTablet(string(linkAuth.Account))
	return tablet.NewRow(blockNum, string(linkAuth.Code), string(linkAuth.Type), string(linkAuth.Requirement), false)
}

func NewDeleteAuthLinkRow(blockNum uint32, actionTrace *pbcodec.ActionTrace) (*AuthLinkRow, error) {
	var unlinkAuth *system.UnlinkAuth
	if err := actionTrace.Action.UnmarshalData(&unlinkAuth); err != nil {
		return nil, err
	}

	tablet := NewAuthLinkTablet(string(unlinkAuth.Account))
	return tablet.NewRow(blockNum, string(unlinkAuth.Code), string(unlinkAuth.Type), string(""), true)
}

func (r *AuthLinkRow) Contract() string {
	contract, _ := r.Explode()
	return contract
}

func (r *AuthLinkRow) Action() string {
	_, action := r.Explode()
	return action
}

func (r *AuthLinkRow) Explode() (contract, action string) {
	parts := strings.Split(r.PrimKey, ":")
	if len(parts) > 0 {
		contract = parts[0]
	}

	if len(parts) > 1 {
		action = parts[1]
	}

	return
}

func (r *AuthLinkRow) Permission() eos.PermissionName {
	return eos.PermissionName(eos.NameToString(binary.BigEndian.Uint64(r.Payload)))
}
