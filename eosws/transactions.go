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

	"github.com/dfuse-io/dfuse-eosio/eosws/mdl"
	"github.com/dfuse-io/dfuse-eosio/eosws/metrics"
	"github.com/dfuse-io/dfuse-eosio/eosws/wsmsg"
	pbcodec "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/codec/v1"
	"github.com/streamingfast/bstream"
	"github.com/streamingfast/bstream/forkable"
	"github.com/streamingfast/derr"
	"github.com/streamingfast/logging"
	"go.uber.org/zap"
)

func pause() {
	time.Sleep(500 * time.Millisecond)
}

func (ws *WSConn) onGetTransaction(ctx context.Context, msg *wsmsg.GetTransaction) {
	zlogger := logging.Logger(ctx, zlog)

	var srcTx *pbcodec.TransactionLifecycle
	var err error

	srcTx, err = ws.db.GetTransaction(ctx, msg.Data.ID)
	if err != nil {
		if !msg.Listen {
			ws.EmitErrorReply(ctx, msg, derr.Wrap(err, "unable to get transaction"))
		}
	} else {
		zlogger.Debug("found transaction in database, about to send it to client")
		lc, err := mdl.ToV1TransactionLifecycle(srcTx)
		if err != nil {
			ws.EmitErrorReply(ctx, msg, derr.Wrap(err, "unable to convert transaction"))
			return
		}

		metrics.DocumentResponseCounter.Inc()
		ws.EmitReply(ctx, msg, wsmsg.NewTransactionLifecycle(lc))
	}

	// If the user specified to listen but with a prefix transaction, too bad as for now we do not support prefix on listen
	if msg.Listen && len(msg.Data.ID) == 64 {
		var source bstream.Source

		libRef, err := ws.db.GetLastWrittenIrreversibleBlockRef(ctx)
		if err != nil {
			ws.EmitErrorReply(ctx, msg, derr.Wrap(err, "unable to get lib"))
			return
		}

		wantedTrxID := msg.Data.ID
		first := false
		handler := bstream.HandlerFunc(func(block *bstream.Block, obj interface{}) error {
			if !first {
				zlogger.Debug("transaction listen handler first block", zap.Stringer("block", block))
				first = true
			}

			blk := block.ToNative().(*pbcodec.Block)

			transactionIDs := make([]string, len(blk.TransactionTraces()))
			for i, transaction := range blk.TransactionTraces() {
				transactionIDs[i] = transaction.Id
			}
			transactionIDs = append(transactionIDs, append(blk.CreatedDTrxIDs(), blk.CanceledDTrxIDs()...)...)

			var sendUpdate bool
			var expectTrxMatchBlockID bool
			var expectBlockIrreversible bool
			for _, id := range transactionIDs {
				if id == wantedTrxID {
					sendUpdate = true
					fObj := obj.(*forkable.ForkableObject)
					expectTrxMatchBlockID = fObj.Step == forkable.StepNew || fObj.Step == forkable.StepRedo
					expectBlockIrreversible = fObj.Step == forkable.StepIrreversible
					break
				}
			}
			if !sendUpdate {
				return nil
			}

			zlogger.Debug("found block with transaction in it",
				zap.Stringer("block", block),
				zap.String("wanted_trx_id", wantedTrxID),
				zap.Bool("expect_trx_match_block_id", expectTrxMatchBlockID),
				zap.Bool("expect_block_irreversible", expectBlockIrreversible),
			)

			blkID := blk.ID()
			go func() {
				timeout := time.After(300 * time.Second) //this timeout is only for that particular attempt to notify the user about this block
				for {
					var err error
					select {
					case <-timeout:
						ws.EmitErrorReply(ctx, msg, DBTrxAppearanceTimeoutError(ctx, blk.ID(), wantedTrxID))
						return
					default:
						if expectBlockIrreversible {
							var lastDBIrr bstream.BlockRef
							lastDBIrr, err = ws.db.GetLastWrittenIrreversibleBlockRef(ctx)
							if err != nil {
								zlogger.Debug("error getting last irreversible blockref from DB", zap.Stringer("block", block), zap.Error(err))
								pause()
								continue
							}
							if lastDBIrr.Num() < blk.Num() {
								pause()
								continue
							}
							srcTx, err = ws.db.GetTransaction(ctx, wantedTrxID)
						} else {
							if expectTrxMatchBlockID {
								srcTx, err = ws.db.GetTransactionWithExpectedBlockID(ctx, wantedTrxID, blkID)
							} else { // we are in a undo/redo, just sending data again to ensure  we have latest data...
								srcTx, err = ws.db.GetTransaction(ctx, wantedTrxID)
							}
						}
						if err != nil {
							zlogger.Debug("error getting transaction from DB", zap.Stringer("block", block), zap.Error(err))
							pause()
							continue
						} else {
							if srcTx.ExecutionIrreversible || srcTx.CancelationIrreversible {
								zlog.Debug("done watching transaction, now irreversible", zap.String("wanted_trx_id", wantedTrxID))
								defer source.Shutdown(nil)
							}

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
		})

		if freq := msg.WithProgress; freq != 0 {
			handler = NewProgressHandler(handler, ws, msg, ctx).ProcessBlock
		}

		nextBlockRef := libRef
		effectiveHandler := bstream.Handler(handler)

		// If we have seen the transaction in the database, we know at which block we must start, it's the block right
		// after execution trace's block id, since we have now seen this block.
		if srcTx != nil && srcTx.ExecutionTrace != nil && srcTx.ExecutionTrace.BlockNum > libRef.Num() {
			nextBlockRef = bstream.NewBlockRefFromID(srcTx.ExecutionTrace.ProducerBlockId)
			effectiveHandler = bstream.NewBlockIDGate(nextBlockRef.ID(), bstream.GateExclusive, handler, bstream.GateOptionWithLogger(zlog))
		}

		forkableHandler := forkable.New(effectiveHandler, forkable.WithLogger(zlog), forkable.WithInclusiveLIB(libRef))
		firstGate := bstream.NewBlockIDGate(libRef.ID(), bstream.GateInclusive, forkableHandler, bstream.GateOptionWithLogger(zlog))

		zlogger.Debug("starting listen transaction handler", zap.Stringer("lib", libRef), zap.Stringer("next_block", nextBlockRef))
		metrics.IncListeners("get_transaction")
		source = ws.subscriptionHub.NewSourceFromBlockRef(libRef, firstGate)
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

		ws.EmitReply(ctx, msg, wsmsg.NewListening(uint32(nextBlockRef.Num())))
		go source.Run()

	}
}
