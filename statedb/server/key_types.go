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

package server

import (
	"encoding/binary"
	"encoding/hex"
	"strconv"

	eos "github.com/eoscanada/eos-go"
)

type KeyConverter interface {
	FromString(key string) (uint64, error)
	ToString(key uint64) (string, error)
}

var keyTypeToKeyConverter = map[string]KeyConverter{
	"uint64":      &Uint64KeyConverter{},
	"name":        &NameKeyConverter{},
	"symbol":      &SymbolKeyConverter{},
	"symbol_code": &SymbolCodeKeyConverter{},
	"hex":         &HexKeyConverter{},
	"hex_be":      &HexBEKeyConverter{},
}

type Uint64KeyConverter struct{}

func (c *Uint64KeyConverter) FromString(key string) (uint64, error) {
	return strconv.ParseUint(key, 10, 64)
}

func (c *Uint64KeyConverter) ToString(key uint64) (string, error) {
	return strconv.FormatUint(key, 10), nil
}

type NameKeyConverter struct{}

func (c *NameKeyConverter) FromString(key string) (uint64, error) {
	return eos.ExtendedStringToName(key)
}

func (c *NameKeyConverter) ToString(key uint64) (string, error) {
	return eos.NameToString(key), nil
}

type SymbolKeyConverter struct{}

func (c *SymbolKeyConverter) FromString(key string) (uint64, error) {
	symbol, err := eos.StringToSymbol(key)
	if err != nil {
		return 0, err
	}

	return symbol.ToUint64()
}

func (c *SymbolKeyConverter) ToString(key uint64) (string, error) {
	return eos.NewSymbolFromUint64(key).String(), nil
}

type SymbolCodeKeyConverter struct{}

func (c *SymbolCodeKeyConverter) FromString(key string) (uint64, error) {
	symbolCode, err := eos.StringToSymbolCode(key)
	if err != nil {
		return 0, err
	}

	return uint64(symbolCode), nil
}

func (c *SymbolCodeKeyConverter) ToString(key uint64) (string, error) {
	return eos.SymbolCode(key).String(), nil
}

type HexKeyConverter struct{}

func (c *HexKeyConverter) FromString(key string) (uint64, error) {
	return strconv.ParseUint(key, 16, 64)
}

func (c *HexKeyConverter) ToString(key uint64) (string, error) {
	keyBuffer := make([]byte, 8)

	binary.LittleEndian.PutUint64(keyBuffer, key)
	return hex.EncodeToString(keyBuffer), nil
}

type HexBEKeyConverter struct{}

func (c *HexBEKeyConverter) FromString(key string) (uint64, error) {
	// FIXME: Provablty need invertiing the chars
	return strconv.ParseUint(key, 16, 64)
}

func (c *HexBEKeyConverter) ToString(key uint64) (string, error) {
	keyBuffer := make([]byte, 8)

	binary.LittleEndian.PutUint64(keyBuffer, key)
	return hex.EncodeToString(keyBuffer), nil
}

func getKeyConverterForType(keyType string) KeyConverter {
	keyConverter, exists := keyTypeToKeyConverter[keyType]
	if !exists {
		// Name is always the default key converter whatever happen
		keyConverter = keyTypeToKeyConverter["name"]
	}

	return keyConverter
}
