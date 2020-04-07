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

package fluxdb

import (
	"context"
	"fmt"
	"net/url"

	"github.com/dfuse-io/dfuse-eosio/fluxdb/store"
	"github.com/dfuse-io/dfuse-eosio/fluxdb/store/bigt"
	"github.com/dfuse-io/dfuse-eosio/fluxdb/store/kv"
	"go.uber.org/zap"
)

// NewKVStore creates the underlying KV store engine base on the DSN string
// received.
//
// This exists in `fluxdb` package since it's shared between `app` and `cmd`
// packages.
func NewKVStore(dsnString string) (store.KVStore, error) {
	ctx := context.Background()

	dsn, err := url.Parse(dsnString)
	if err != nil {
		return nil, fmt.Errorf("parsing fluxdb dsn: %s", err)
	}

	zlog.Info("creating underlying kv store engine", zap.String("scheme", dsn.Scheme), zap.String("dsn", dsnString))

	switch dsn.Scheme {
	case "bigtable":
		return bigt.NewKVStore(ctx, dsnString)
	case "badger", "tikv", "bigkv":
		return kv.NewStore(ctx, dsnString)
	default:
		return nil, fmt.Errorf("unknown scheme %q from dsn %q", dsn.Scheme, dsnString)
	}
}
