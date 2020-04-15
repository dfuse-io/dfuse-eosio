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
	"time"

	"github.com/dfuse-io/dfuse-eosio/eosdb"

	"github.com/dfuse-io/kvdb"
)

func (db *DB) BlockIDAt(ctx context.Context, start time.Time) (id string, err error) {
	// This implementation improves on the previous as it takes in precedence the irreversible
	// block ID at that time, and falls back on whatever block exists if not irreversible
	// for that time.

	q := `SELECT blks.id, blks.blockTime, irrblks.irreversible
    FROM blks
    LEFT JOIN irrblks ON (blks.id = irrblks.id)
    WHERE blockTime = ?
    ORDER BY blks.blockTime, blks.id
`

	id, _, err = db.scanBlockIDIrreversibleOrFirst(ctx, q, start)
	if id == "" {
		return "", kvdb.ErrNotFound
	}
	return id, err
}

func (db *DB) scanBlockIDIrreversibleOrFirst(ctx context.Context, q string, start time.Time) (id string, tm time.Time, err error) {
	rows, err := db.db.QueryContext(ctx, q, start)
	if err != nil {
		return
	}

	var firstID string
	var firstTime time.Time
	for rows.Next() {
		var irr *bool
		var id string
		var tm SqliteTime
		if err = rows.Scan(&id, &tm, &irr); err != nil {
			return id, time.Time(tm), err
		}

		if firstID == "" {
			firstID = id
			firstTime = time.Time(tm)
		}

		if isIrreversible := eosdb.BoolPtr(irr); isIrreversible {
			return id, time.Time(tm), nil
		}
	}
	if err := rows.Err(); err != nil {
		return id, tm, err
	}

	return firstID, firstTime, nil
}

func (db *DB) BlockIDAfter(ctx context.Context, start time.Time, inclusive bool) (id string, foundtime time.Time, err error) {
	q := `SELECT blks.id, blks.blockTime, irrblks.irreversible
    FROM blks
    LEFT JOIN irrblks ON (blks.id = irrblks.id)`
	if inclusive {
		q += " WHERE blockTime >= ?"
	} else {
		q += " WHERE blockTime > ?"
	}
	q += " ORDER BY blockTime ASC"
	q += " LIMIT 4" // supporting 4 forks of the same block number, which would be very surprising on EOS

	id, foundtime, err = db.scanBlockIDIrreversibleOrFirst(ctx, q, start)
	if id == "" {
		return "", time.Time{}, kvdb.ErrNotFound
	}
	return
}

func (db *DB) BlockIDBefore(ctx context.Context, start time.Time, inclusive bool) (id string, foundtime time.Time, err error) {
	q := `SELECT blks.id, blks.blockTime, irrblks.irreversible
    FROM blks
    LEFT JOIN irrblks ON (blks.id = irrblks.id)`
	if inclusive {
		q += " WHERE blockTime <= ?"
	} else {
		q += " WHERE blockTime < ?"
	}
	q += " ORDER BY blockTime DESC"
	q += " LIMIT 4" // supporting 4 forks of the same block number, which would be very surprising on EOS

	id, foundtime, err = db.scanBlockIDIrreversibleOrFirst(ctx, q, start)
	if id == "" {
		return "", time.Time{}, kvdb.ErrNotFound
	}
	return
}
