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

package sqlsync

import (
	"database/sql"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/dfuse-io/bstream"

	// FIXME: Have those be registered by the CALLER
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	// _ "github.com/mattn/go-sqlite3" // this requires CGO_ENABLED so we disable it by default
)

var INSERT_IGNORE = ""
var BEGIN_TRANSACTION = ""
var SQL_UINT64 = "int unsigned NOT NULL"
var SQL_UINT32 = "int unsigned NOT NULL"

type DB struct {
	db                    *sql.DB
	paramsPlaceholderFunc func(int) string
}

func (d *DB) Empty() bool {
	//FIXME
	return true
}
func (d *DB) GetStartBlock() (bstream.BlockRef, error) {
	//FIXME
	return nil, nil
}

func genParamsPlaceholder(params int, questionMark bool) string {
	out := "("
	for i := 1; i < params; i++ {

		if questionMark {
			out = fmt.Sprintf("%s?,", out)
		} else {
			out = fmt.Sprintf("%s$%d,", out, i)
		}
	}
	if questionMark {
		out = fmt.Sprintf("%s?)", out)
	} else {
		out = fmt.Sprintf("%s$%d)", out, params)
	}
	return out
}

func NewDB(dsnString string) (*DB, error) {
	u, err := url.Parse(dsnString)
	if err != nil {
		return nil, err
	}

	var db *sql.DB
	var questionMarks bool

	switch u.Scheme {
	case "mysql":
		BEGIN_TRANSACTION = "START TRANSACTION "
		INSERT_IGNORE = "INSERT IGNORE "
		questionMarks = true

		var pw string
		if u.User != nil {
			pass, _ := u.User.Password()
			pw = fmt.Sprintf("%s:%s@", u.User.Username(), pass)
		}
		vals := u.Query()
		dsn := fmt.Sprintf("%s(%s)%s?%s", pw, u.Host, u.Path, vals.Encode())

		db, err = sql.Open("mysql", dsn)

		//	case "sqlite3", "sqlite":
		//		BEGIN_TRANSACTION = "BEGIN TRANSACTION "
		//		INSERT_IGNORE = "INSERT OR IGNORE "
		//		questionMarks = true
		//		dsn := u.Host
		//		if dsn == "" {
		//			dsn = u.Path // for sqlite:///tmp/mama.sqlite
		//		}
		//		db, err = sql.Open("sqlite3", dsn)
		//		if err != nil {
		//			return nil, err
		//		}
		//		err = db.Ping() // force create empty file at least, to see if it works

	case "postgres":
		BEGIN_TRANSACTION = "START TRANSACTION "
		INSERT_IGNORE = "INSERT IGNORE "
		questionMarks = false

		SQL_UINT64 = "bigint NOT NULL"
		SQL_UINT32 = "int NOT NULL"

		var user, password string
		if u.User != nil {
			user = u.User.Username()
			password, _ = u.User.Password()
		}

		hostOnly := strings.TrimSuffix(u.Host, ":"+u.Port())
		dsn := fmt.Sprintf("host=%s port=%s user=%s "+
			"password=%s dbname=%s sslmode=disable",
			hostOnly, u.Port(), user, password, strings.TrimLeft(u.Path, "/"))

		db, err = sql.Open("postgres", dsn)
		if err != nil {
			return nil, err
		}
		err = db.Ping()

	}
	if err != nil {
		return nil, err
	}

	chainToSQLTypes = map[string]string{
		"name":   "varchar(13) NOT NULL",
		"string": "varchar(1024) NOT NULL",
		"symbol": "varchar(8) NOT NULL",
		"bool":   "boolean",
		"int64":  "int NOT NULL",
		"uint64": SQL_UINT64,
		"int32":  "int NOT NULL", // make smaller
		"uint32": SQL_UINT32,
		"asset":  "varchar(64) NOT NULL",
	}

	return &DB{
		db:                    db,
		paramsPlaceholderFunc: func(p int) string { return genParamsPlaceholder(p, questionMarks) },
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
