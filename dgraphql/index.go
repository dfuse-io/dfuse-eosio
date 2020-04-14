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

	eosResolver "github.com/dfuse-io/dfuse-eosio/dgraphql/resolvers"
	"github.com/dfuse-io/dgraphql"
	dgraphqlApp "github.com/dfuse-io/dgraphql/app/dgraphql"
	"github.com/dfuse-io/dgrpc"
	"github.com/dfuse-io/kvdb/eosdb"
	pbabicodec "github.com/dfuse-io/pbgo/dfuse/abicodec/eosio/v1"
	pbblockmeta "github.com/dfuse-io/pbgo/dfuse/blockmeta/v1"
	pbsearch "github.com/dfuse-io/pbgo/dfuse/search/v1"
)

type Config struct {
	dgraphqlApp.Config
	SearchAddr    string
	ABICodecAddr  string
	BlockMetaAddr string
	KVDBDSN       string
}

func NewApp(config *Config) (*dgraphqlApp.App, error) {
	schemas, err := SetupSchemas(&Config{
		SearchAddr:    config.SearchAddr,
		ABICodecAddr:  config.ABICodecAddr,
		BlockMetaAddr: config.BlockMetaAddr,
		KVDBDSN:       config.KVDBDSN,
	})
	if err != nil {
		return nil, err
	}
	return dgraphqlApp.New(&dgraphqlApp.Config{
		HTTPListenAddr:  config.HTTPListenAddr,
		GRPCListenAddr:  config.GRPCListenAddr,
		NetworkID:       config.NetworkID,
		AuthPlugin:      config.AuthPlugin,
		MeteringPlugin:  config.MeteringPlugin,
		OverrideTraceID: config.OverrideTraceID,
		Protocol:        config.Protocol,
		Schemas:         schemas,
	}), nil
}

func SetupSchemas(config *Config) (*dgraphql.Schemas, error) {

	zlog.Info("creating db reader")
	dbReader, err := eosdb.New(config.KVDBDSN)
	if err != nil {
		return nil, fmt.Errorf("invalid eosdb connection info provided: %w", err)
	}

	zlog.Info("creating abicodec grpc client")
	abiConn, err := dgrpc.NewInternalClient(config.ABICodecAddr)
	if err != nil {
		return nil, fmt.Errorf("failed getting abi grpc client: %w", err)
	}
	abiClient := pbabicodec.NewDecoderClient(abiConn)

	zlog.Info("creating blockmeta grpc client")
	blockMetaClient, err := pbblockmeta.NewClient(config.BlockMetaAddr)
	if err != nil {
		return nil, fmt.Errorf("failed creating blockmeta client: %w", err)
	}

	zlog.Info("creating search grpc client")

	searchConn, err := dgrpc.NewInternalClient(config.SearchAddr)
	if err != nil {
		return nil, fmt.Errorf("failed getting search grpc client: %w", err)
	}
	searchRouterClient := pbsearch.NewRouterClient(searchConn)

	zlog.Info("configuring resolver and parsing schemas")
	resolver := eosResolver.NewRoot(searchRouterClient, dbReader, blockMetaClient, abiClient)
	schemas, err := dgraphql.NewSchemas(resolver)
	if err != nil {
		return nil, fmt.Errorf("unable to parse schema: %w", err)
	}

	return schemas, nil
}
