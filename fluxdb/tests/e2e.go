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

package tests

import (
	"context"
	"net/http"
	"testing"

	"github.com/dfuse-io/bstream"
	"github.com/dfuse-io/bstream/forkable"
	"github.com/dfuse-io/dfuse-eosio/fluxdb"
	"github.com/dfuse-io/dfuse-eosio/fluxdb/server"
	pbeos "github.com/dfuse-io/dfuse-eosio/pb/dfuse/codecs/eos"
	"github.com/gavv/httpexpect/v2"
	"github.com/stretchr/testify/require"
)

type e2eTester func(ctx context.Context, t *testing.T, feedBlocks blocksFeeder, e *httpexpect.Expect)
type blocksFeeder func(blocks ...*pbeos.Block)

func e2eTest(t *testing.T, storeFactory StoreFactory, tester e2eTester) {
	ctx := context.Background()
	kvStore, cleanup := storeFactory()
	defer cleanup()

	db := fluxdb.New(kvStore)
	defer db.Close()

	handler := fluxdb.NewHandler(db)
	handler.EnableWrites()
	handler.EnableWriteOnEachIrreversibleStep()
	handler.InitializeStartBlockID()

	db.HeadBlock = handler.HeadBlock
	db.SpeculativeWritesFetcher = handler.FetchSpeculativeWrites

	server := server.New(":25678", db)

	runSource := func(blocks ...*pbeos.Block) {
		source := bstream.NewMockSource(bstreamBlocks(t, blocks...), bstream.NewPreprocessor(fluxdb.PreprocessBlock, forkable.New(handler)))
		source.Run()

		require.NoError(t, source.Err())
	}

	tester(ctx, t, runSource, httpexpect.WithConfig(httpexpect.Config{
		Client: &http.Client{
			Transport: httpexpect.NewBinder(server.Handler()),
			Jar:       httpexpect.NewJar(),
		},
		Reporter: httpexpect.NewAssertReporter(t),
		Printers: []httpexpect.Printer{
			httpexpect.NewDebugPrinter((*exceptLogger)(zlog.Sugar()), true),
		},
	}))
}
