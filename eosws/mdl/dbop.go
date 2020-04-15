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

package mdl

import (
	"encoding/hex"

	v0 "github.com/dfuse-io/eosws-go/mdl/v0"

	pbeos "github.com/dfuse-io/dfuse-eosio/pb/dfuse/codecs/eos"
	v1 "github.com/dfuse-io/eosws-go/mdl/v1"
)

func ToV1DBOps(in []*pbeos.DBOp) (out []*v1.DBOp) {
	for _, inOp := range in {
		out = append(out, ToV1DBOp(inOp))
	}
	return out
}

func ToV1DBOp(in *pbeos.DBOp) *v1.DBOp {
	out := &v1.DBOp{
		Op:          in.LegacyOperation(),
		ActionIndex: int(in.ActionIndex),
		Account:     in.Code,
		Table:       in.TableName,
		Scope:       in.Scope,
		Key:         in.PrimaryKey,
		New:         ToV1DBRow(in.NewData, in.NewPayer),
		Old:         ToV1DBRow(in.OldData, in.OldPayer),
	}
	return out

}

func ToV0DBOps(in []*pbeos.DBOp) (out []*v0.DBOp) {
	for _, inOp := range in {
		out = append(out, ToV0DBOp(inOp))
	}
	return out
}

func ToV0DBOp(in *pbeos.DBOp) *v0.DBOp {
	out := &v0.DBOp{
		Operation:   in.LegacyOperation(),
		ActionIndex: int(in.ActionIndex),
		OldPayer:    in.OldPayer,
		NewPayer:    in.NewPayer,
		TablePath:   in.Code + "/" + in.Scope + "/" + in.TableName + "/" + in.PrimaryKey,
		OldData:     hex.EncodeToString(in.OldData),
		NewData:     hex.EncodeToString(in.NewData),
	}
	return out
}

func ToV1DBRow(data []byte, payer string) *v1.DBRow {
	row := &v1.DBRow{
		Payer: payer,
		Hex:   hex.EncodeToString(data),
	}
	return row
}
