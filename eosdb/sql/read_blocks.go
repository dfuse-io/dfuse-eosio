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

	"github.com/dfuse-io/bstream"
	"github.com/dfuse-io/dfuse-eosio/eosdb"
	pbeos "github.com/dfuse-io/dfuse-eosio/pb/dfuse/codecs/eos"
	"github.com/dfuse-io/kvdb"
)

func (db *DB) GetLastWrittenBlockID(ctx context.Context) (blockID string, err error) {
	q := `SELECT id FROM blks ORDER BY number DESC LIMIT 1`
	var id string
	if err = db.db.QueryRowContext(ctx, q).Scan(&id); err != nil {
		return "", err
	}

	return id, nil
}

func (db *DB) GetBlock(ctx context.Context, id string) (*pbeos.BlockWithRefs, error) {
	// TODO: did this function support prefix search on the block ID?  Is it important?
	// if id is full length, then check for equality, otherwise for LIKE = '?%'
	q := getBlockSelectFields + `WHERE blks.id = ? LIMIT 1`
	rows, err := db.db.QueryContext(ctx, q, id)
	if err != nil {
		return nil, err
	}

	out, err := db.scanBlockRows(rows)
	if err != nil {
		return nil, err
	}

	// TODO: is that how we return an empty thing? What's the signature expected by the interface?
	if len(out) == 0 {
		return nil, kvdb.ErrNotFound
	}
	return out[0], nil
}

var getBlockSelectFields = `
SELECT blks.id, blks.block, blks.trxRefs, blks.traceRefs, blks.implicitTrxRefs, irrblks.irreversible
    FROM blks
    LEFT JOIN irrblks ON (blks.id = irrblks.id)
`

func (db *DB) scanBlockRows(rows *sql.Rows) (out []*pbeos.BlockWithRefs, err error) {
	for rows.Next() {
		blk := &pbeos.BlockWithRefs{}

		var blockData, rawTrxRefs, rawTraceRefs, rawImplicitTrxRefs []byte
		var irr *bool
		err := rows.Scan(
			&blk.Id, &blockData, &rawTrxRefs, &rawTraceRefs, &rawImplicitTrxRefs, &irr,
		)
		if err != nil {
			return nil, fmt.Errorf("scanning: %s", err)
		}

		blk.Irreversible = eosdb.BoolPtr(irr)

		blk.Block = &pbeos.Block{}
		if err := db.dec.Into(blockData, blk.Block); err != nil {
			return nil, fmt.Errorf("decode block: %w", err)
		}

		// NOTE: we stitch them together, because historically, we had
		// the dtrx and trx mixed in the refs saved in the DB.  In this Driver
		// implementation, we split them in storage, so we can reconstruct the
		// full block without losing any information.
		blk.TransactionTraceRefs = &pbeos.TransactionRefs{}
		if err := db.dec.Into(rawTraceRefs, blk.TransactionTraceRefs); err != nil {
			return nil, fmt.Errorf("decode trace refs: %w", err)
		}
		blk.TransactionRefs = &pbeos.TransactionRefs{}
		if err := db.dec.Into(rawTrxRefs, blk.TransactionRefs); err != nil {
			return nil, fmt.Errorf("decode trx refs: %w", err)
		}
		blk.ImplicitTransactionRefs = &pbeos.TransactionRefs{}
		if err := db.dec.Into(rawImplicitTrxRefs, blk.ImplicitTransactionRefs); err != nil {
			return nil, fmt.Errorf("decode dtrxrefs: %w", err)
		}
		// TODO: bigtable data merges `implicitTrxRefs` and `trxRefs` .. eventually,
		// we'll want to have them split so we can reconstruct the blocks.  Right now,
		// we can't fully do that.

		out = append(out, blk)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error scanning: %w", err)
	}

	return
}

func (db *DB) GetBlockByNum(ctx context.Context, num uint32) ([]*pbeos.BlockWithRefs, error) {
	q := getBlockSelectFields + `WHERE blks.number = ?`
	rows, err := db.db.QueryContext(ctx, q, num)
	if err != nil {
		return nil, err
	}

	out, err := db.scanBlockRows(rows)
	if err != nil {
		return nil, err
	}

	if out == nil {
		return nil, kvdb.ErrNotFound
	}

	return out, nil
}

func (db *DB) GetClosestIrreversibleIDAtBlockNum(ctx context.Context, num uint32) (ref bstream.BlockRef, err error) {
	q := getBlockSelectFields + `where blks.number <= ? and irrblks.irreversible = true ORDER BY blks.number DESC limit 10`

	rows, err := db.db.QueryContext(ctx, q, num)
	if err != nil {
		return nil, err
	}

	blocks, err := db.scanBlockRows(rows)
	if err != nil {
		return nil, err
	}
	if len(blocks) < 1 {
		return nil, kvdb.ErrNotFound
	}

	ref = bstream.NewBlockRefFromID(bstream.BlockRefFromID(blocks[0].Id))
	return ref, nil
}

func (db *DB) GetIrreversibleIDAtBlockID(ctx context.Context, ID string) (ref bstream.BlockRef, err error) {
	blk, err := db.GetBlock(ctx, ID)
	if err != nil {
		return nil, err
	}
	q := getBlockSelectFields + `where blks.number = ? and irrblks.irreversible = true`

	rows, err := db.db.QueryContext(ctx, q, blk.Block.DposIrreversibleBlocknum)
	if err != nil {
		return nil, err
	}

	blocks, err := db.scanBlockRows(rows)
	if err != nil {
		return nil, err
	}
	if len(blocks) < 1 {
		return nil, kvdb.ErrNotFound
	}

	ref = bstream.NewBlockRefFromID(bstream.BlockRefFromID(blocks[0].Id))
	return ref, nil
}

func (db *DB) ListBlocks(ctx context.Context, startBlockNum uint32, limit int) (out []*pbeos.BlockWithRefs, err error) {
	q := getBlockSelectFields + `
		WHERE blks.number <= ?
		ORDER BY blks.number DESC
		LIMIT ?
`
	rows, err := db.db.QueryContext(ctx, q, startBlockNum, limit)
	if err != nil {
		return nil, err
	}

	out, err = db.scanBlockRows(rows)
	if err != nil {
		return nil, err
	}

	return
}
func (db *DB) ListSiblingBlocks(ctx context.Context, blockNum uint32, spread uint32) (out []*pbeos.BlockWithRefs, err error) {

	startBlockNum := blockNum + spread
	endBlockNum := blockNum - (spread + 1)
	if spread >= blockNum {
		endBlockNum = blockNum - 1
	}

	q := getBlockSelectFields + `
		WHERE blks.number <= ?
		AND blks.number > ?
		ORDER BY blks.number DESC
`
	rows, err := db.db.QueryContext(ctx, q, startBlockNum, endBlockNum)
	if err != nil {
		return nil, err
	}

	out, err = db.scanBlockRows(rows)
	if err != nil {
		return nil, err
	}

	return
}
