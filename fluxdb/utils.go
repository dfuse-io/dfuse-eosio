// Copyright 2020 dfuse Platform Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package fluxdb

import (
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/eoscanada/eos-go"
)

func N(name string) uint64 {
	out, _ := StringToName(name)
	return out
}

func NA(name eos.Name) uint64 {
	return eos.MustStringToName(string(name))
}

func EN(name string) uint64 {
	out, _ := eos.ExtendedStringToName(name)
	return out
}

func K(key string) []byte {
	return []byte(key)
}

func BlockNum(blockID string) uint32 {
	if len(blockID) < 8 {
		return 0
	}

	bin, err := hex.DecodeString(blockID[:8])
	if err != nil {
		panic(fmt.Errorf("value %q is not a valid block num uint32 value: %w", blockID, err))
	}

	return big.Uint32(bin)
}

var little = binary.LittleEndian
var big = binary.BigEndian

func HexBlockNum(blockNum uint32) string {
	return fmt.Sprintf("%08x", blockNum)
}

func HexRevBlockNum(blockNum uint32) string {
	return HexBlockNum(math.MaxUint32 - blockNum)
}

func BlockNumHexRev(revBlockNumKey string) uint32 {
	return math.MaxUint32 - hexToBlockNum(revBlockNumKey)
}

func HexName(name uint64) string {
	return fmt.Sprintf("%016x", name)
}

func hexToBlockNum(blockNumHex string) uint32 {
	value, err := strconv.ParseInt(blockNumHex, 16, 32)
	if err != nil {
		panic(fmt.Errorf("value %q is not a valid block num uint32 value", blockNumHex))
	}

	return uint32(value)
}

func explodeWritableRowKey(rowKey string) (tableKey string, blockNum uint32, primKey string, err error) {
	parts := strings.Split(rowKey, ":")
	partCount := len(parts)

	switch {
	// AuthLink al:<account>:<blockNum>:<contract>:<action>
	case parts[0] == "al":
		if partCount != 5 {
			err = fmt.Errorf("auth link row key should have 5 parts, got %d", partCount)
			return
		}

		tableKey = strings.Join(parts[0:2], ":")
		blockNum, err = KeyChunkToBlockNum(parts[2])
		primKey = strings.Join(parts[3:5], ":")

	// AccountResourceLimit arl:<account>:<blockNum>:<primaryKey>
	case parts[0] == "arl":
		if partCount != 4 {
			err = fmt.Errorf("account resource limit row key should have 4 parts, got %d", partCount)
			return
		}

		tableKey = strings.Join(parts[0:2], ":")
		blockNum, err = KeyChunkToBlockNum(parts[2])
		primKey = parts[3]

	// BlockResourceLimit brl:<blockNum>:<primaryKey>
	case parts[0] == "brl":
		if partCount != 3 {
			err = fmt.Errorf("block resource limit row key should have 3 parts, got %d", partCount)
			return
		}

		tableKey = strings.Join(parts[0:1], ":")
		blockNum, err = KeyChunkToBlockNum(parts[1])
		primKey = parts[2]

	// KeyAccount ka2:<publicKey>:<blockNum>:<account>:<permission>
	case parts[0] == "ka2":
		if partCount != 5 {
			err = fmt.Errorf("key account row key should have 5 parts, got %d", partCount)
			return
		}

		tableKey = strings.Join(parts[0:2], ":")
		blockNum, err = KeyChunkToBlockNum(parts[2])
		primKey = strings.Join(parts[3:5], ":")

	// TableData td:<account>:<table>:<scope>:<blockNum>:<rowPrimaryKey>
	case parts[0] == "td":
		if partCount != 6 {
			err = fmt.Errorf("table data row key should have 6 parts, got %d", partCount)
			return
		}

		tableKey = strings.Join(parts[0:4], ":")
		blockNum, err = KeyChunkToBlockNum(parts[4])
		primKey = parts[5]

	// TableScope ts:<account>:<table>:<blockNum>:<scope>
	case parts[0] == "ts":
		if partCount != 5 {
			err = fmt.Errorf("table scope row key should have 5 parts, got %d", partCount)
			return
		}

		tableKey = strings.Join(parts[0:3], ":")
		blockNum, err = KeyChunkToBlockNum(parts[3])
		primKey = parts[4]

	default:
		err = fmt.Errorf("don't know how to explode row key %q", rowKey)
	}

	return
}

// chunkKeyRevBlockNum returns the actual block num out of a
// reverse-encoded block num
func chunkKeyRevBlockNum(key string, prefixKey string) (blockNum uint32, err error) {
	blockNum, err = chunkKeyBlockNum(key, prefixKey)
	if err != nil {
		return 0, err
	}

	return math.MaxUint32 - uint32(blockNum), nil
}

func chunkKeyBlockNum(key string, prefixKey string) (blockNum uint32, err error) {
	if !strings.HasPrefix(key, prefixKey) {
		return 0, fmt.Errorf("key %s should start with prefix key %s", key, prefixKey)
	}

	if len(key) < len(prefixKey)+8 {
		return 0, fmt.Errorf("key %s is too small too contains block num, should have at least 8 characters more than prefix", key)
	}

	revBlockNum := key[len(prefixKey) : len(prefixKey)+8]
	if len(revBlockNum) != 8 {
		return 0, fmt.Errorf("revBlockNum %s should have a length of 8", revBlockNum)
	}

	val, err := strconv.ParseUint(revBlockNum, 16, 32)
	if err != nil {
		return 0, err
	}

	return uint32(val), nil
}

func KeyChunkToBlockNum(chunk string) (blockNum uint32, err error) {
	if len(chunk) != 8 {
		return 0, errors.New("block chunk should have length of 8")
	}

	val, err := strconv.ParseUint(chunk, 16, 32)
	if err != nil {
		return 0, err
	}

	return uint32(val), nil
}

func chunkKeyUint64(key string, chunkIndex int) (value uint64, valid bool) {
	chunks := strings.Split(key, ":")
	if chunkIndex >= len(chunks) {
		return 0, false
	}

	val := chunks[chunkIndex]
	if len(val) != 16 {
		return 0, false
	}

	value, err := strconv.ParseUint(val, 16, 64)
	if err != nil {
		return 0, false
	}

	return value, true
}
