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
	"net/url"
	"strings"
	"time"

	"github.com/dfuse-io/dfuse-eosio/eosdb"
	_ "github.com/go-sql-driver/mysql" // FIXME: Have those be registered by the CALLER
	_ "github.com/mattn/go-sqlite3"    // FIXME: Have this be registered by the CALLER
)

var INSERT_IGNORE = ""
var BEGIN_TRANSACTION = ""

type DB struct {
	db *sql.DB

	enc *eosdb.ProtoEncoder
	dec *eosdb.ProtoDecoder

	// Required only when writing
	writerChainID []byte
}

func init() {
	eosdb.Register("mysql", New)
	eosdb.Register("sqlite3", New)
	eosdb.Register("sqlite", New)
}

func New(dsnString string, opts ...eosdb.Option) (eosdb.Driver, error) {
	u, err := url.Parse(dsnString)
	if err != nil {
		return nil, err
	}

	createTables := u.Query().Get("createTables") == "true"
	vals := u.Query()
	vals.Del("createTables")

	//MySql specific stuff
	var db *sql.DB

	switch u.Scheme {
	case "mysql":
		BEGIN_TRANSACTION = "START TRANSACTION "
		INSERT_IGNORE = "INSERT IGNORE "
		vals.Set("parseTime", "true")

		var pw string
		if u.User != nil {
			pass, _ := u.User.Password()
			pw = fmt.Sprintf("%s:%s@", u.User.Username(), pass)
		}
		dsn := fmt.Sprintf("%s(%s)%s?%s", pw, u.Host, u.Path, vals.Encode())

		db, err = sql.Open("mysql", dsn)

	case "sqlite3", "sqlite":
		BEGIN_TRANSACTION = "BEGIN TRANSACTION "
		INSERT_IGNORE = "INSERT OR IGNORE "
		dsn := u.Host // for sqlite://:memory:
		if dsn == "" {
			dsn = u.Path // for sqlite:///tmp/mama.sqlite
		}
		db, err = sql.Open("sqlite3", dsn)
	}
	if err != nil {
		return nil, err
	}

	if createTables {
		_, err = db.ExecContext(context.Background(), createTrxsTableStmt)
		if err != nil {
			return nil, fmt.Errorf("create trxs: %s", err)
		}

		_, err = db.ExecContext(context.Background(), createImplicitTrxsTableStmt)
		if err != nil {
			return nil, fmt.Errorf("create implicittrxs: %s", err)
		}

		_, err = db.ExecContext(context.Background(), createDtrxsTableStmt)
		if err != nil {
			return nil, fmt.Errorf("create dtrxs: %s", err)
		}

		_, err = db.ExecContext(context.Background(), createTracesTableStmt)
		if err != nil {
			return nil, fmt.Errorf("create traces: %s", err)
		}

		_, err := db.ExecContext(context.Background(), createBlocksTableStmt)
		if err != nil {
			return nil, fmt.Errorf("create blks: %s", err)
		}

		_, err = db.ExecContext(context.Background(), createBlocksBlksTimeIndexesStmt)
		if err != nil {
			if !strings.Contains(err.Error(), "Duplicate key name") && !strings.Contains(err.Error(), "already exists") {
				return nil, fmt.Errorf("create blks blks_time indexes: %s", err)
			}
		}

		_, err = db.ExecContext(context.Background(), createBlocksBlksNumIndexesStmt)
		if err != nil {
			if !strings.Contains(err.Error(), "Duplicate key name") && !strings.Contains(err.Error(), "already exists") {
				return nil, fmt.Errorf("create blks blks_num indexes: %s", err)
			}
		}

		_, err = db.ExecContext(context.Background(), createIrrBlocksTableStmt)
		if err != nil {
			return nil, fmt.Errorf("create irrblks: %s", err)
		}

		_, err = db.ExecContext(context.Background(), createAccountsTableStmt)
		if err != nil {
			return nil, fmt.Errorf("create accts: %s", err)
		}
	}

	return &DB{
		db:  db,
		enc: eosdb.NewProtoEncoder(),
		dec: eosdb.NewProtoDecoder(),
	}, nil
}

type SqliteTime time.Time

func (t *SqliteTime) Scan(v interface{}) error {
	ti := v.(string)
	vt, err := time.Parse("2006-01-02 15:04:05-07:00", ti)
	if err != nil {
		return err
	}
	*t = SqliteTime(vt.UTC())
	return nil
}
