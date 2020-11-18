package statedb

import (
	"encoding/binary"
	"errors"

	"github.com/dfuse-io/fluxdb"
	eos "github.com/eoscanada/eos-go"
)

var bigEndian = binary.BigEndian

var baseEntry = fluxdb.NewBaseSingletEntry
var baseRow = fluxdb.NewBaseTabletRow

var errABIUnmarshal = errors.New("unmarshal abi")

func bytesToName(bytes []byte) string {
	return eos.NameToString(bigEndian.Uint64(bytes))
}

func bytesToName2(bytes []byte) (string, string) {
	return eos.NameToString(bigEndian.Uint64(bytes)), eos.NameToString(bigEndian.Uint64(bytes[8:]))
}

func bytesToName3(bytes []byte) (string, string, string) {
	return eos.NameToString(bigEndian.Uint64(bytes)),
		eos.NameToString(bigEndian.Uint64(bytes[8:])),
		eos.NameToString(bigEndian.Uint64(bytes[16:]))
}

func bytesToJoinedName2(bytes []byte) string {
	return eos.NameToString(bigEndian.Uint64(bytes)) + ":" + eos.NameToString(bigEndian.Uint64(bytes[8:]))
}

func bytesToJoinedName3(bytes []byte) string {
	return eos.NameToString(bigEndian.Uint64(bytes)) +
		":" + eos.NameToString(bigEndian.Uint64(bytes[8:])) +
		":" + eos.NameToString(bigEndian.Uint64(bytes[16:]))
}

var standardNameConverter = eos.MustStringToName
var extendedNameConverter = mustExtendedStringToName

func standardNameToBytes(names ...string) (out []byte) {
	return nameToBytes(standardNameConverter, names)
}

func extendedNameToBytes(names ...string) (out []byte) {
	return nameToBytes(extendedNameConverter, names)
}

func nameToBytes(converter func(name string) uint64, names []string) (out []byte) {
	out = make([]byte, 8*len(names))
	moving := out
	for _, name := range names {
		bigEndian.PutUint64(moving, converter(name))
		moving = moving[8:]
	}

	return
}

func nameaToBytes(name eos.AccountName) (out []byte) {
	out = make([]byte, 8)
	bigEndian.PutUint64(out, eos.MustStringToName(string(name)))
	return
}

func namenToBytes(name eos.Name) (out []byte) {
	out = make([]byte, 8)
	bigEndian.PutUint64(out, eos.MustStringToName(string(name)))
	return
}

func mustExtendedStringToName(name string) uint64 {
	val, err := eos.ExtendedStringToName(name)
	if err != nil {
		panic(err)
	}

	return val
}
