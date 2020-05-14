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
	"time"

	_ "github.com/go-sql-driver/mysql" // FIXME: Have those be registered by the CALLER
	_ "github.com/mattn/go-sqlite3"    // FIXME: Have this be registered by the CALLER
)

var INSERT_IGNORE = ""
var BEGIN_TRANSACTION = ""

type DB struct {
	db *sql.DB
}

func NewDB(dsnString string) (*DB, error) {
	u, err := url.Parse(dsnString)
	if err != nil {
		return nil, err
	}

	var db *sql.DB

	switch u.Scheme {
	case "mysql":
		BEGIN_TRANSACTION = "START TRANSACTION "
		INSERT_IGNORE = "INSERT IGNORE "

		var pw string
		if u.User != nil {
			pass, _ := u.User.Password()
			pw = fmt.Sprintf("%s:%s@", u.User.Username(), pass)
		}
		vals := u.Query()
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

	return &DB{
		db: db,
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
