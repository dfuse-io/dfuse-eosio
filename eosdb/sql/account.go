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

package sql

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	pbeos "github.com/dfuse-io/dfuse-eosio/pb/dfuse/codecs/eos"
	"github.com/dfuse-io/kvdb"
	"github.com/golang/protobuf/ptypes"
)

var accountSelectFields = `
SELECT account, blockId, blockNum, blockTime, creator, trxId
    FROM accts
`

func (db *DB) scanAccountRows(rows *sql.Rows) (out []*pbeos.AccountCreationRef, err error) {
	for rows.Next() {
		account := &pbeos.AccountCreationRef{
			Account:       "",
			Creator:       "",
			BlockNum:      0,
			BlockId:       "",
			BlockTime:     nil,
			TransactionId: "",
		}

		var blockTime *SqliteTime
		err := rows.Scan(
			&account.Account, &account.BlockId, &account.BlockNum, &blockTime, &account.Creator, &account.TransactionId,
		)
		if err != nil {
			return nil, fmt.Errorf("scanning: %s", err)
		}

		account.BlockTime, err = ptypes.TimestampProto(time.Time(*blockTime))
		if err != nil {
			return nil, fmt.Errorf("block time to proto: %s", err)
		}
		out = append(out, account)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error scanning: %w", err)
	}
	return
}

func (db *DB) putNewAccount(blk *pbeos.Block, trace *pbeos.TransactionTrace, act *pbeos.ActionTrace) error {
	// TODO: do we ALWAYS have the decoded data for `newaccount`, even at the beginning of the chain? Do we have an ABI set during the boot sequence?
	_, err := db.db.Exec(INSERT_IGNORE+"INTO accts (account, blockId, blockNum, blockTime, creator, trxId) VALUES (?, ?, ?, ?, ?, ?)",
		act.GetData("name").String(),
		blk.ID(),
		blk.Num(),
		blk.MustTime(),
		act.GetData("creator").String(),
		trace.Id,
	)
	if err != nil {
		return err
	}
	return nil
}

func (db *DB) GetAccount(ctx context.Context, accountName string) (*pbeos.AccountCreationRef, error) {
	q := accountSelectFields + `WHERE accts.account = ?`
	rows, err := db.db.QueryContext(ctx, q, accountName)
	if err != nil {
		return nil, err
	}

	out, err := db.scanAccountRows(rows)
	if err != nil {
		return nil, err
	}
	if len(out) == 0 {
		return nil, kvdb.ErrNotFound
	}
	return out[0], nil
}

func (db *DB) ListAccountNames(ctx context.Context, concurrentReadCount uint32) ([]string, error) {
	if concurrentReadCount < 1 {
		return nil, fmt.Errorf("invalid concurrent read")
	}
	q := accountSelectFields
	rows, err := db.db.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}

	accountRows, err := db.scanAccountRows(rows)
	if err != nil {
		return nil, err
	}
	if len(accountRows) == 0 {
		return nil, nil
	}

	var out []string

	for _, account := range accountRows {
		out = append(out, account.Account)
	}

	return out, nil
}
