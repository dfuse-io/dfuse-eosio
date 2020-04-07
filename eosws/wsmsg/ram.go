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

package wsmsg

func init() {
	RegisterOutgoingMessage("ram_out", RAMOut{})
}

type RAMOut struct {
	CommonOut
	Data struct {
		Delta    int64  `json:"delta"`
		Account  string `json:"account"`
		BlockNum uint32 `json:"block_num"`
		Action   string `json:"action"`
	} `json:"data"`
}

func NewRAMOut(blockNum uint32, action string, delta int64, account string) *RAMOut {
	out := &RAMOut{}
	out.Data.Delta = delta
	out.Data.BlockNum = blockNum
	out.Data.Account = account
	out.Data.Action = action
	return out
}
