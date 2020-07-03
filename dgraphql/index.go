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

package dgraphql

import (
	"fmt"

	"go.uber.org/zap"

	drateLimiter "github.com/dfuse-io/dauth/ratelimiter"
	"github.com/dfuse-io/derr"
	eosResolver "github.com/dfuse-io/dfuse-eosio/dgraphql/resolvers"
	pbabicodec "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/abicodec/v1"
	"github.com/dfuse-io/dfuse-eosio/trxdb"
	"github.com/dfuse-io/dgraphql"
	dgraphqlApp "github.com/dfuse-io/dgraphql/app/dgraphql"
	"github.com/dfuse-io/dgrpc"
	pbblockmeta "github.com/dfuse-io/pbgo/dfuse/blockmeta/v1"
	pbsearch "github.com/dfuse-io/pbgo/dfuse/search/v1"
)

type Config struct {
	dgraphqlApp.Config
	RatelimiterPlugin string
	SearchAddr        string
	ABICodecAddr      string
	BlockMetaAddr     string
	KVDBDSN           string
}

func NewApp(config *Config) (*dgraphqlApp.App, error) {
	zlog.Info("new dgraphql eosio app", zap.Reflect("config", config))

	dgraphqlBaseConfig := config.Config

	return dgraphqlApp.New(&dgraphqlBaseConfig, &SchemaFactory{config: config}), nil
}

type SchemaFactory struct {
	config *Config
}

func (f *SchemaFactory) Schemas() (*dgraphql.Schemas, error) {
	zlog.Info("creating db reader")
	dbReader, err := trxdb.New(f.config.KVDBDSN)
	if err != nil {
		return nil, fmt.Errorf("invalid trxdb connection info provided: %w", err)
	}

	zlog.Info("creating abicodec grpc client")
	abiConn, err := dgrpc.NewInternalClient(f.config.ABICodecAddr)
	if err != nil {
		return nil, fmt.Errorf("failed getting abi grpc client: %w", err)
	}
	abiClient := pbabicodec.NewDecoderClient(abiConn)

	zlog.Info("creating blockmeta grpc client")
	blockMetaClient, err := pbblockmeta.NewClient(f.config.BlockMetaAddr)
	if err != nil {
		return nil, fmt.Errorf("failed creating blockmeta client: %w", err)
	}

	zlog.Info("creating search grpc client")

	searchConn, err := dgrpc.NewInternalClient(f.config.SearchAddr)
	if err != nil {
		return nil, fmt.Errorf("failed getting search grpc client: %w", err)
	}
	searchRouterClient := pbsearch.NewRouterClient(searchConn)

	rateLimiter, err := drateLimiter.New(f.config.RatelimiterPlugin)
	derr.Check("unable to initialize rate limiter", err)

	zlog.Info("configuring resolver and parsing schemas")
	resolver, err := eosResolver.NewRoot(searchRouterClient, dbReader, blockMetaClient, abiClient, rateLimiter)
	if err != nil {
		return nil, fmt.Errorf("unable to create root resolver: %w", err)
	}

	schemas, err := dgraphql.NewSchemas(resolver)
	if err != nil {
		return nil, fmt.Errorf("unable to parse schema: %w", err)
	}

	return schemas, nil
}
