// Copyright 2019 dfuse Platform Inc.
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

package resolvers

import (
	"encoding/hex"
	"fmt"
	"time"

	"github.com/dfuse-io/opaque"
	eos "github.com/eoscanada/eos-go"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/timestamp"
	graphql "github.com/graph-gophers/graphql-go"
)

func S(in string) *string {
	return &in
}

func optS(in string) *string {
	if in == "" {
		return nil
	}
	return &in
}

func decodeCursor(cursor *string) string {
	if cursor == nil {
		return ""
	}
	out, _ := opaque.FromOpaque(*cursor)
	return out
}

func toTime(timestamp *timestamp.Timestamp) graphql.Time {
	t, err := ptypes.Timestamp(timestamp)
	if err != nil {
		panic(fmt.Errorf("toTime: %s", err))
	}

	return graphql.Time{Time: t}
}

func toOptTime(stringInput string) *graphql.Time {
	if stringInput == "" {
		return nil
	}

	t, err := time.Parse("2006-01-02T15:04:05.999", stringInput)
	if err != nil {
		return nil
	}

	return &graphql.Time{Time: t}
}

func toBool(b *bool) bool {
	if b == nil {
		return false
	}
	return *b
}

func nameToEncoding(name string, encoding string) string {
	switch encoding {
	case "NAME":
		return name
	case "DECIMAL":
		val, _ := eos.StringToName(name)
		return fmt.Sprintf(`%d`, val)
	case "HEXADECIMAL":
		val, _ := eos.StringToName(name)
		return fmt.Sprintf(`%x`, val)
	case "SYMBOL":
		val, _ := eos.NameToSymbol(eos.Name(name))
		return val.String()
	case "SYMBOL_CODE":
		val, _ := eos.NameToSymbolCode(eos.Name(name))
		return val.String()
	default:
		return "[invalid encoding]"
	}
}

func validateBlockId(id string) bool {
	_, err := hex.DecodeString(id)
	if err != nil {
		return false
	}
	return true
}

func countMinOne(count int) int64 {
	if count < 1 {
		return 1
	}
	return int64(count)
}
