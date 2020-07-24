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
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/eoscanada/eos-go"
)

var bigEndian = binary.BigEndian

var UN = eos.NameToString
var N = eos.MustStringToName

func EN(name string) uint64 {
	out, _ := eos.ExtendedStringToName(name)
	return out
}

func HexBlockNum(blockNum uint32) string {
	return fmt.Sprintf("%08x", blockNum)
}

func HexRevBlockNum(blockNum uint32) string {
	return HexBlockNum(math.MaxUint32 - blockNum)
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
