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

package bigt

import pbdeos "github.com/dfuse-io/pbgo/dfuse/codecs/deos"

type TransactionRow struct {
	// TODO: Replace Key usage by `ID` and `BlockID`.. Key is an intricacy
	// of Bigtables, shouldn't bubble outside this model.
	// It contains: `trx_id:block_id` as per `Keyer.ReadTransaction`
	Key string

	Transaction      *pbdeos.SignedTransaction
	TransactionTrace *pbdeos.TransactionTrace // really: ExecutionTrace
	BlockHeader      *pbdeos.BlockHeader
	PublicKeys       []string
	CreatedBy        *pbdeos.ExtDTrxOp
	CanceledBy       *pbdeos.ExtDTrxOp
	Irreversible     bool

	// TODO: phase this out, who relies on this anyway?
	Written bool
}
