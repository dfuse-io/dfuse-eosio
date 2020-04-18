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

package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/dfuse-io/dfuse-eosio/eosdb"
	pbcodec "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/codec/v1"
	"github.com/dfuse-io/dhammer"
)

var concurrentBatches = flag.Int("concurrent-batches", 2, "concurrent threads")
var batchSize = flag.Int("max-batch-size", 15, "maximum batch size")

func main() {
	flag.Parse()

	cli, err := eosdb.New("bigtable://dev.dev/test")
	if err != nil {
		panic(err)
	}

	batchFunc := func(ctx context.Context, toProcess []interface{}) (processed []interface{}, err error) {
		var trxs []string
		fmt.Println("querying with those lines", toProcess)
		for _, v := range toProcess {
			trxs = append(trxs, v.(string))
		}

		trxEvents, err := cli.GetTransactionEventsBatch(ctx, trxs)

		for idx, evs := range trxEvents {
			if len(evs) == 0 {
				err = fmt.Errorf("cannot get result for id: %s", trxs[idx])
				break
			} else {
				processed = append(processed, evs)

			}
		}
		return
	}
	hammer := dhammer.NewHammer(*batchSize, *concurrentBatches, batchFunc, dhammer.FirstBatchUnitary())

	doneReading := make(chan interface{})
	// display the results
	go func() {
		for {
			select {
			case <-hammer.Terminating():
				fmt.Println("done ?", hammer.Err())
				return
			case v, ok := <-hammer.Out:
				if !ok {
					close(doneReading)
					return
				}
				evs := v.([]*pbcodec.TransactionEvent)
				fmt.Println("got row: ", evs[0].Id, "events:", len(evs))
			}
		}
	}()

	ctx := context.Background()
	hammer.Start(ctx)

	reader := bufio.NewReader(os.Stdin)
	for {
		text, err := reader.ReadString('\n')
		if err != nil || text == "\n" || text == "" {
			close(hammer.In)
			break
		}
		text = strings.Replace(text, "\n", "", -1)
		hammer.In <- text
	}

	<-doneReading
	fmt.Println(hammer.Err())

}
