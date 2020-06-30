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

package eosws

import (
	"context"
	"time"

	"github.com/dfuse-io/bstream"
	"github.com/dfuse-io/bstream/forkable"
	"github.com/dfuse-io/derr"
	"github.com/dfuse-io/dfuse-eosio/eosws/mdl"
	"github.com/dfuse-io/dfuse-eosio/eosws/metrics"
	"github.com/dfuse-io/dfuse-eosio/eosws/wsmsg"
	pbcodec "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/codec/v1"
	eos "github.com/eoscanada/eos-go"
)

func (ws *WSConn) onGetTransaction(ctx context.Context, msg *wsmsg.GetTransaction) {
	var srcTx *pbcodec.TransactionLifecycle
	var err error

	startBlockID, err := ws.db.GetLastWrittenBlockID(ctx)
	if err != nil {
		ws.EmitErrorReply(ctx, msg, derr.Wrap(err, "unable to get last written block"))
		return
	}

	srcTx, err = ws.db.GetTransaction(ctx, msg.Data.ID)
	if err != nil {
		if !msg.Listen {
			ws.EmitErrorReply(ctx, msg, derr.Wrap(err, "unable to get transaction"))
		}
	} else {
		lc, err := mdl.ToV1TransactionLifecycle(srcTx)
		if err != nil {
			ws.EmitErrorReply(ctx, msg, derr.Wrap(err, "unable to convert transaction"))
			return
		}

		metrics.DocumentResponseCounter.Inc()
		ws.EmitReply(ctx, msg, wsmsg.NewTransactionLifecycle(lc))
	}

	if msg.Listen {
		libID, err := ws.db.GetIrreversibleIDAtBlockID(ctx, startBlockID)
		if err != nil {
			ws.EmitErrorReply(ctx, msg, derr.Wrap(err, "unable to get lib"))
			return
		}

		wantedTrxID := msg.Data.ID
		resendNextNewBlock := false
		handler := bstream.HandlerFunc(func(block *bstream.Block, obj interface{}) error {
			// un an undo or redo notice, we wait for the next "normal" block
			// then we resend the transaction lifecycle
			fObj := obj.(*forkable.ForkableObject)
			if fObj.Step == forkable.StepUndo || fObj.Step == forkable.StepRedo {
				resendNextNewBlock = true
				return nil
			}

			// gate to see if that change is already in bigtable!
			waitForIrreversible := false
			if fObj.Step == forkable.StepIrreversible {
				waitForIrreversible = true
			}

			blk := block.ToNative().(*pbcodec.Block)

			transactionIds := make([]string, len(blk.TransactionTraces()))
			for i, transaction := range blk.TransactionTraces() {
				transactionIds[i] = transaction.Id
			}

			for _, id := range append(transactionIds, append(blk.CreatedDTrxIDs(), blk.CanceledDTrxIDs()...)...) {
				if wantedTrxID == id || resendNextNewBlock {
					resendNextNewBlock = false
					go func() {
						timeout := time.After(300 * time.Second) //this timeout is only for that particular attempt to notify the user about this block
						for {
							select {
							case <-timeout:
								ws.EmitErrorReply(ctx, msg, DBTrxAppearanceTimeoutError(ctx, blk.ID(), wantedTrxID))
								return
							default:
								b, err := ws.db.GetBlock(ctx, blk.ID())
								if err != nil {
									// FIXME: don't we want to distinguish system failures, and NotFound here?
									time.Sleep(time.Second)
									continue
								}

								if waitForIrreversible && !b.Irreversible {
									time.Sleep(time.Second)
									continue
								}

								srcTx, err = ws.db.GetTransaction(ctx, msg.Data.ID)
								if err != nil {
									ws.EmitErrorReply(ctx, msg, derr.Wrap(err, "unable to get transaction, internal error"))
								} else {
									tx, err := mdl.ToV1TransactionLifecycle(srcTx)
									if err != nil {
										ws.EmitErrorReply(ctx, msg, derr.Wrap(err, "unable to convert transaction"))
										return
									}
									metrics.DocumentResponseCounter.Inc()
									ws.EmitReply(ctx, msg, wsmsg.NewTransactionLifecycle(tx))
								}
								return
							}
						}
					}()
					return nil
				}
			}

			return nil
		})

		if freq := msg.WithProgress; freq != 0 {
			handler = NewProgressHandler(handler, ws, msg, ctx).ProcessBlock
		}

		gateHandler := bstream.NewBlockIDGate(startBlockID, bstream.GateExclusive, handler)
		forkableHandler := forkable.New(gateHandler, forkable.WithExclusiveLIB(libID))
		firstGate := bstream.NewBlockIDGate(libID.ID(), bstream.GateInclusive, forkableHandler)

		metrics.IncListeners("get_transaction")
		source := ws.subscriptionHub.NewSourceFromBlockRef(libID, firstGate)
		source.OnTerminating(func(_ error) {
			metrics.CurrentListeners.Dec("get_transaction")
		})

		err = ws.RegisterListener(ctx, msg.ReqID, func() error {
			source.Shutdown(nil)
			return nil
		})

		if err != nil {
			source.Shutdown(nil) // important to ensure that OnRunFunc is run
			ws.EmitErrorReply(ctx, msg, derr.Wrap(err, "unable to register listener to ws connection"))
			return
		}

		ws.EmitReply(ctx, msg, wsmsg.NewListening(eos.BlockNum(startBlockID)))
		go source.Run()

	}
}
