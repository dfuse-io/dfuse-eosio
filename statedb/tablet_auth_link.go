package statedb

import (
	"fmt"

	pbcodec "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/codec/v1"
	pbstatedb "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/statedb/v1"
	"github.com/dfuse-io/fluxdb"
	"github.com/eoscanada/eos-go"
	"github.com/eoscanada/eos-go/system"
	"github.com/golang/protobuf/proto"
)

const alCollection = 0xB100
const alPrefix = "al"

func init() {
	fluxdb.RegisterTabletFactory(alCollection, alPrefix, func(identifier []byte) (fluxdb.Tablet, error) {
		if len(identifier) < 8 {
			return nil, fluxdb.ErrInvalidKeyLengthAtLeast("auth link tablet identifier", 8, len(identifier))
		}

		return AuthLinkTablet(identifier[0:8]), nil
	})
}

func NewAuthLinkTablet(account string) AuthLinkTablet {
	return AuthLinkTablet(standardNameToBytes(account))
}

// AuthLinkTablet tablet is composed
type AuthLinkTablet []byte

func (t AuthLinkTablet) Collection() uint16 {
	return alCollection
}

func (t AuthLinkTablet) Identifier() []byte {
	return t
}

func (t AuthLinkTablet) Row(height uint64, primaryKey []byte, data []byte) (fluxdb.TabletRow, error) {
	if len(primaryKey) != 16 {
		return nil, fluxdb.ErrInvalidKeyLength("auth link primary key", 16, len(primaryKey))
	}

	return &AuthLinkRow{baseRow(t, height, primaryKey, data)}, nil
}

func (t AuthLinkTablet) String() string {
	return alPrefix + ":" + bytesToName(t)
}

type AuthLinkRow struct {
	fluxdb.BaseTabletRow
}

func NewInsertAuthLinkRow(blockNum uint64, actionTrace *pbcodec.ActionTrace) (*AuthLinkRow, error) {
	var linkAuth *system.LinkAuth
	if err := actionTrace.Action.UnmarshalData(&linkAuth); err != nil {
		return nil, err
	}

	return newAuthLinkRow(blockNum, string(linkAuth.Account), string(linkAuth.Code), string(linkAuth.Type), string(linkAuth.Requirement))
}

func NewDeleteAuthLinkRow(blockNum uint64, actionTrace *pbcodec.ActionTrace) (*AuthLinkRow, error) {
	var unlinkAuth *system.UnlinkAuth
	if err := actionTrace.Action.UnmarshalData(&unlinkAuth); err != nil {
		return nil, err
	}

	return newAuthLinkRow(blockNum, string(unlinkAuth.Account), string(unlinkAuth.Code), string(unlinkAuth.Type), "")
}

func newAuthLinkRow(blockNum uint64, account, contract, action string, permission string) (row *AuthLinkRow, err error) {
	tablet := NewAuthLinkTablet(account)
	primaryKey := standardNameToBytes(string(contract), string(action))

	var value []byte
	if permission != "" {
		pb := pbstatedb.AuthLinkValue{Permission: eos.MustStringToName(permission)}

		if value, err = proto.Marshal(&pb); err != nil {
			return nil, fmt.Errorf("marshal proto: %w", err)
		}
	}

	return &AuthLinkRow{baseRow(tablet, blockNum, primaryKey, value)}, nil
}

func (r *AuthLinkRow) Explode() (contract, action string) {
	return bytesToName2(r.PrimaryKey())
}

func (r *AuthLinkRow) Permission() (out eos.PermissionName, err error) {
	pb := pbstatedb.AuthLinkValue{}
	if err := proto.Unmarshal(r.Value(), &pb); err != nil {
		return out, fmt.Errorf("marshal proto: %w", err)
	}

	return eos.PermissionName(eos.NameToString(pb.Permission)), nil
}

func (r *AuthLinkRow) String() string {
	return r.Stringify(bytesToJoinedName2(r.PrimaryKey()))
}
