package statedb

import (
	"errors"
	"fmt"
	"math"

	pbstatedb "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/statedb/v1"
	"github.com/dfuse-io/fluxdb"
	eos "github.com/eoscanada/eos-go"
	"github.com/golang/protobuf/proto"
)

const kaCollection = 0xB300
const kaPrefix = "ka"

// We actually don't use the payload, so we cache an empty protobuf and use it as the value when the element is "set"
var kaValue []byte

func init() {
	fluxdb.RegisterTabletFactory(kaCollection, kaPrefix, func(identifier []byte) (fluxdb.Tablet, error) {
		if len(identifier) < 2 {
			return nil, fluxdb.ErrInvalidKeyLengthAtLeast("key account tablet identifier", 2, len(identifier))
		}

		publicKeyLen := int(bigEndian.Uint16(identifier))
		if len(identifier) < 2+publicKeyLen {
			return nil, fluxdb.ErrInvalidKeyLengthAtLeast("key account tablet identifier", 2+publicKeyLen, len(identifier))
		}

		return KeyAccountTablet(identifier[0 : 2+publicKeyLen]), nil
	})

	var err error
	if kaValue, err = proto.Marshal(&pbstatedb.KeyAccountValue{Present: true}); err != nil {
		panic(fmt.Errorf("unable to marshal key account payload: %w", err))
	}

	if len(kaValue) == 0 {
		panic(errors.New("marshal key account payload should have at least 1 byte, got 0"))
	}
}

func NewKeyAccountTablet(publicKey string) KeyAccountTablet {
	if len(publicKey) > math.MaxUint16 {
		panic(fmt.Errorf("only accepting public keys smaller than %d characters, got %d", math.MaxUint16, len(publicKey)))
	}

	key := make([]byte, 2+len(publicKey))
	bigEndian.PutUint16(key, uint16(len(publicKey)))
	copy(key[2:], []byte(publicKey))

	return KeyAccountTablet(key)
}

type KeyAccountTablet []byte

func (t KeyAccountTablet) Collection() uint16 {
	return kaCollection
}

func (t KeyAccountTablet) Identifier() []byte {
	return t
}

func (t KeyAccountTablet) Row(height uint64, primaryKey []byte, data []byte) (fluxdb.TabletRow, error) {
	if len(primaryKey) != 16 {
		return nil, fluxdb.ErrInvalidKeyLength("key account primary key", 16, len(primaryKey))
	}

	return &KeyAccountRow{baseRow(t, height, primaryKey, data)}, nil
}

func (t KeyAccountTablet) String() string {
	return kaPrefix + ":" + string(t[2:])
}

type KeyAccountRow struct {
	fluxdb.BaseTabletRow
}

func NewKeyAccountRow(blockNum uint64, publicKey, account, permission string, isDeletion bool) (row *KeyAccountRow, err error) {
	var value []byte
	if !isDeletion {
		value = kaValue
	}

	tablet := NewKeyAccountTablet(publicKey)
	return &KeyAccountRow{baseRow(tablet, blockNum, standardNameToBytes(account, permission), value)}, nil
}

func (r *KeyAccountRow) Scope() string {
	return bytesToName(r.PrimaryKey())
}

func (r *KeyAccountRow) Payer() (string, error) {
	pb := pbstatedb.ContractTableScopeValue{}
	if err := proto.Unmarshal(r.Value(), &pb); err != nil {
		return "", err
	}

	return eos.NameToString(pb.Payer), nil
}

func (r *KeyAccountRow) Explode() (account, permission string) {
	return bytesToName2(r.PrimaryKey())
}

func (r *KeyAccountRow) String() string {
	return r.Stringify(bytesToName(r.PrimaryKey()))
}
