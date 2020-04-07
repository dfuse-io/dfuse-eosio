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

package kvdb_init

import (
	"encoding/hex"
	"fmt"

	"github.com/dfuse-io/kvdb/eosdb"
	"go.uber.org/zap"
)

type Config struct {
	ChainId  string // Chain ID
	KvdbDsn  string // Storage connection string
	Protocol string // Protocol to load, EOS or ETH
}

type App struct {
	config *Config
}

func New(config *Config) *App {
	return &App{
		config: config,
	}
}

func (a *App) Run() error {
	zlog.Info("launching kvdb init", zap.Reflect("config", a.config))

	switch a.config.Protocol {
	case "EOS":
		chainID, err := hex.DecodeString(a.config.ChainId)
		if err != nil {
			return fmt.Errorf("decoding chain_id from command line argument: %w", err)
		}

		db, err := eosdb.New(a.config.KvdbDsn)
		if err != nil {
			return fmt.Errorf("unable to create eosdb: %w", err)
		}
		// FIXME: make sure we call CLOSE() at the end!
		//defer db.Close()

		db.SetWriterChainID(chainID)
		zlog.Info("exiting after table creation")

	case "ETH":
		return fmt.Errorf("support for ETH temporarily removed")

	default:
		return fmt.Errorf("unsupported --protocol, use EOS or ETH: %q", a.config.Protocol)
	}

	return nil
}
