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

package blockmeta

import (
	"context"
	"time"

	"github.com/dfuse-io/blockmeta"
	"github.com/dfuse-io/bstream"
	"github.com/dfuse-io/dfuse-eosio/trxdb"
	"go.uber.org/zap"
)

type EOSBlockmetaDB struct {
	Driver trxdb.DB
}

func (db *EOSBlockmetaDB) BlockIDAt(ctx context.Context, start time.Time) (id string, err error) {
	return db.Driver.BlockIDAt(ctx, start)
}

func (db *EOSBlockmetaDB) BlockIDAfter(ctx context.Context, start time.Time, inclusive bool) (id string, foundtime time.Time, err error) {
	return db.Driver.BlockIDAfter(ctx, start, inclusive)
}

func (db *EOSBlockmetaDB) BlockIDBefore(ctx context.Context, start time.Time, inclusive bool) (id string, foundtime time.Time, err error) {
	return db.Driver.BlockIDBefore(ctx, start, inclusive)
}

func (db *EOSBlockmetaDB) GetLastWrittenBlockID(ctx context.Context) (blockID string, err error) {
	return db.Driver.GetLastWrittenBlockID(ctx)
}

func (db *EOSBlockmetaDB) GetIrreversibleIDAtBlockNum(ctx context.Context, num uint64) (ref bstream.BlockRef, err error) {
	return db.Driver.GetClosestIrreversibleIDAtBlockNum(ctx, uint32(num))
}

func (db *EOSBlockmetaDB) GetIrreversibleIDAtBlockID(ctx context.Context, id string) (ref bstream.BlockRef, err error) {
	return db.Driver.GetIrreversibleIDAtBlockID(ctx, id)
}

func (db *EOSBlockmetaDB) GetForkPreviousBlocks(ctx context.Context, forkTop bstream.BlockRef) ([]bstream.BlockRef, error) {
	blocks := []bstream.BlockRef{forkTop} // forkTop always included
	next := forkTop
	window := 10

	counter := 0
	for {
		if counter >= 1000 {
			zlog.Error("stopping after too many iterations",
				zap.String("next_id", next.ID()),
				zap.Uint64("next_num", next.Num()),
			)
			return nil, blockmeta.ErrNotFound
		}
		rows, err := db.Driver.ListBlocks(ctx, uint32(next.Num()), window)
		if err != nil {
			return nil, err
		}
		for _, row := range rows {
			zlog.Debug("looking for next block",
				zap.String("next_id", next.ID()),
				zap.Uint64("next_num", next.Num()),
				zap.String("row_id", row.Block.ID()),
				zap.Uint64("row_num", row.Block.Num()),
			)
			if row.Block.Num() < next.Num() {
				return nil, blockmeta.ErrNotFound
			}
			if row.Block.ID() == next.ID() {
				if row.Irreversible {
					return blocks, nil
				}

				bstream.NewBlockRef(row.Block.Id, uint64(row.Block.Number))
				zlog.Debug("found block", zap.Stringer("row", row.Block))

				blocks = append(blocks)
				next = bstream.NewBlockRefFromID(row.Block.PreviousID())
			}
		}
		if window <= 100 {
			window += 5 // expands in case we are on a very large fork or there are multitude of blocks with same number
		}
		counter++
	}
}
