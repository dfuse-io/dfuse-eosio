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

package bigt

import "cloud.google.com/go/bigtable"

const abiFamilyName = "abi"
const abiColumnName = "abi"

const rowFamilyName = "raw"
const rowColumnName = "raw"

const indexFamilyName = "index"
const indexColumnName = "snapshot"

const lastBlockFamilyName = "state"
const lastBlockColumnName = "block_id"

var latestCellOnly = bigtable.LatestNFilter(1)
var latestCellFilter = bigtable.RowFilter(latestCellOnly)

func btRowItem(row bigtable.Row, familyName, columnName string) (out bigtable.ReadItem, ok bool) {
	colName := familyName + ":" + columnName
	for _, el := range row[familyName] {
		if el.Column == colName {
			return el, true
		}
	}
	return
}
