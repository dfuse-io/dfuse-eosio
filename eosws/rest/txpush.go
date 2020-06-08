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

package rest

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/dfuse-io/bstream"
	"github.com/dfuse-io/bstream/forkable"
	"github.com/dfuse-io/bstream/hub"
	"github.com/dfuse-io/dfuse-eosio/codec"
	"github.com/dfuse-io/dfuse-eosio/eosws"
	"github.com/dfuse-io/dfuse-eosio/eosws/metrics"
	pbcodec "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/codec/v1"
	"github.com/eoscanada/eos-go"
	"github.com/eoscanada/eos-go/eoserr"
	"github.com/tidwall/gjson"
	"go.uber.org/zap"
)

/////// PUSHER ROUTER

type TxPushRouter struct {
	dumbAPIProxy           http.Handler
	pushTransactionHandler http.Handler
}

func NewTxPushRouter(dumbAPIProxy http.Handler, pushTransactionHandler http.Handler) *TxPushRouter {
	return &TxPushRouter{
		dumbAPIProxy:           dumbAPIProxy,
		pushTransactionHandler: pushTransactionHandler,
	}
}

func (t *TxPushRouter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Delete all CORS related headers from the request so once we reach the
	// proxied endpoint, CORS response headers are not added. This way, we
	// fully control within `eosws` the actual CORS for the requests.
	deleteCORSHeaders(r)

	pushTransactionGuaranteeOption := r.Header.Get("X-Eos-Push-Guarantee")
	if isNotPushTransaction(r.URL.EscapedPath(), pushTransactionGuaranteeOption) {
		t.dumbAPIProxy.ServeHTTP(w, r)
		return
	}

	t.pushTransactionHandler.ServeHTTP(w, r)
}

////// PUSHER

type TxPusher struct {
	API             *eos.API
	subscriptionHub *hub.SubscriptionHub
}

type PushResponse struct {
	TransactionID string                `json:"transaction_id"`
	BlockID       string                `json:"block_id"`
	BlockNum      uint32                `json:"block_num"`
	Processed     *eos.TransactionTrace `json:"processed"`
}

func NewTxPusher(API *eos.API, subscriptionHub *hub.SubscriptionHub) *TxPusher {
	return &TxPusher{
		API:             API,
		subscriptionHub: subscriptionHub,
	}
}

func (t *TxPusher) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	guarantee := r.Header.Get("X-Eos-Push-Guarantee")

	ctx := r.Context()

	eosws.TrackUserEvent(ctx, "rest_request",
		"method", r.Method,
		"host", r.Host,
		"path", r.URL.Path,
		"query", r.URL.Query(),
		"push-trx-guarantee", guarantee,
	)

	var tx *eos.PackedTransaction
	incomingContent, err := ioutil.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		zlog.Warn("pushTrx: ioutil.ReadAll", zap.Error(err))
		return
	}

	err = json.Unmarshal(incomingContent, &tx)
	if checkHTTPError(err, "couldn't decode incoming json", eoserr.ErrParseErrorException, w, zap.String("body", string(incomingContent))) {
		return
	}

	nodeosInfo, err := t.API.GetInfo(r.Context())
	if checkHTTPError(err, "cannot connect to API", eoserr.ErrUnhandledException, w) {
		return
	}

	trxIDCheckSum, err := tx.ID()
	if checkHTTPError(err, "cannot compute transaction ID", eoserr.ErrUnhandledException, w) {
		return
	}
	trxID := trxIDCheckSum.String()

	liveSourceFactory := bstream.SourceFactory(func(handler bstream.Handler) bstream.Source {
		return t.subscriptionHub.NewSource(handler, 10) // does not need joining
	})

	var trxTraceFoundChan <-chan *pbcodec.TransactionTrace
	var shutdownFunc func(error)
	lib := nodeosInfo.LastIrreversibleBlockID.String()
	expirationDelay := time.Minute * 2 //baseline for inblock inclusion
	normalizedGuarantee := guarantee
	switch guarantee {
	case "in-block":
		zlog.Debug("Waiting for trx to appear in a block", zap.String("hexTrxID", trxID), zap.Float64("minutes", expirationDelay.Minutes()))
		trxTraceFoundChan, shutdownFunc = awaitTransactionInBlock(ctx, trxID, liveSourceFactory)
	case "handoff:1", "handoffs:1":
		normalizedGuarantee = "handoffs:1"
		expirationDelay += 1 * time.Minute
		trxTraceFoundChan, shutdownFunc = awaitTransactionPassedHandoffs(ctx, lib, trxID, 1, t.subscriptionHub)
	case "handoff:2", "handoffs:2":
		normalizedGuarantee = "handoffs:2"
		expirationDelay += 1 * time.Minute
		trxTraceFoundChan, shutdownFunc = awaitTransactionPassedHandoffs(ctx, lib, trxID, 2, t.subscriptionHub)
	case "handoff:3", "handoffs:3":
		normalizedGuarantee = "handoffs:3"
		expirationDelay += 1 * time.Minute
		trxTraceFoundChan, shutdownFunc = awaitTransactionPassedHandoffs(ctx, lib, trxID, 3, t.subscriptionHub)
	case "irreversible":
		expirationDelay += 6 * time.Minute
		trxTraceFoundChan, shutdownFunc = awaitTransactionIrreversible(ctx, trxID, liveSourceFactory)
	default:
		msg := "unknown value for X-Eos-Push-Guarantee. Please use 'irreversible', 'in-block', 'handoff:1', 'handoffs:2', 'handoffs:3'"
		checkHTTPError(fmt.Errorf(msg), msg, eoserr.ErrUnhandledException, w)
		return
	}
	metrics.IncListeners("push_transaction")
	metrics.PushTrxCount.Inc(normalizedGuarantee)
	defer metrics.CurrentListeners.Dec("push_transaction")
	defer shutdownFunc(nil) // closing the "awaitTransaction" pipelines...

	timedoutContext, cancel := context.WithTimeout(ctx, 1*time.Minute)

	var pushResp json.RawMessage
	if r.URL.EscapedPath() == "/v1/chain/push_transaction" {
		pushResp, err = t.API.PushTransactionRaw(timedoutContext, tx)
	} else {
		pushResp, err = t.API.SendTransactionRaw(timedoutContext, tx)
	}

	cancel()
	if err != nil {
		if err.Error() == context.Canceled.Error() {
			metrics.TimedOutPushingTrxCount.Inc(normalizedGuarantee)
		} else {
			metrics.FailedPushTrxCount.Inc(normalizedGuarantee)
		}
		if apiErr, ok := err.(eos.APIError); ok {
			apiErrCnt, err := json.Marshal(apiErr)
			if err == nil {
				w.WriteHeader(apiErr.Code)
				w.Write(apiErrCnt)
				return
			}
		}
		zlog.Error("cannot push transaction to Nodeos API", zap.Error(err))
		checkHTTPError(err, fmt.Sprintf("cannot push transaction %q to Nodeos API.", trxID), eoserr.ErrUnhandledException, w)
		return
	}

	idCheck := gjson.GetBytes(pushResp, "transaction_id").String()
	if idCheck != trxID {
		msg := fmt.Sprintf("pushed transaction ID %q mismatch transaction ID received from API %q", trxID, idCheck)
		checkHTTPError(errors.New(msg), msg, eoserr.ErrUnhandledException, w)
	}

	// FIXME:
	// if we return an error but we DID submit the transaction to the chain, ideally
	// we'd return something to that effect.. so the user can start tracking its transaction ID
	// separately.. Maybe we can't do anything about it though...

	select {
	case <-time.After(expirationDelay):
		metrics.TimedOutPushTrxCount.Inc(normalizedGuarantee)
		msg := fmt.Sprintf("too long waiting for inclusion of %q into a block", trxID)
		checkHTTPError(errors.New(msg), msg, eoserr.ErrTimeoutException, w)
		return

	case trxTrace := <-trxTraceFoundChan:
		blockID := trxTrace.ProducerBlockId

		eosTrace := codec.TransactionTraceToEOS(trxTrace)

		resp := &PushResponse{
			BlockID:       blockID,
			BlockNum:      eos.BlockNum(blockID),
			Processed:     eosTrace,
			TransactionID: trxID,
		}

		out, err := json.Marshal(resp)
		if checkHTTPError(err, "cannot marshal response", eoserr.ErrUnhandledException, w) {
			return
		}

		metrics.SucceededPushTrxCount.Inc(normalizedGuarantee)
		w.Header().Set("content-length", fmt.Sprintf("%d", len(out)))
		w.Write([]byte(out))
	}
}

func checkHTTPError(err error, msg string, errorCode eoserr.Error, w http.ResponseWriter, logFields ...zap.Field) bool {
	if err != nil {
		fields := append([]zap.Field{
			zap.Error(err),
			zap.Int("errorCode", errorCode.Code),
			zap.String("errorName", errorCode.Name),
		}, logFields...)

		zlog.Info("push transaction error, "+msg, fields...)
		apiError := eos.NewAPIError(500, msg, errorCode)
		// FIXME: the error logging should use something like one of:
		// http://www.gorillatoolkit.org/pkg/handlers#CustomLoggingHandler
		// http://www.gorillatoolkit.org/pkg/handlers#CombinedLoggingHandler
		// http://www.gorillatoolkit.org/pkg/handlers#LoggingHandler

		out, _ := json.Marshal(apiError)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(500)
		w.Write([]byte(out))
		return true
	}
	return false
}

var corsRequestHeaders = []string{
	"Origin",
	"Access-Control-Request-Method",
	"Access-Control-Request-Headers",
}

func deleteCORSHeaders(r *http.Request) {
	for _, corsRequestHeader := range corsRequestHeaders {
		r.Header.Del(corsRequestHeader)
	}
}

func countUniqueElem(elements []string) int {
	encountered := map[string]bool{}
	for v := range elements {
		encountered[elements[v]] = true
	}
	return len(encountered)
}

var runningPushInHandoffs int64

// awaitTransactionPassedHandoffs starts a forkaware pipeline that awaits a number
// of producer handoffs after which it sends back the transaction traces in a channel
func awaitTransactionPassedHandoffs(ctx context.Context, libID string, trxID string, requiredHandoffs int, subscriptionHub *hub.SubscriptionHub) (<-chan *pbcodec.TransactionTrace, func(error)) {
	trxFound := make(chan *pbcodec.TransactionTrace)
	var done bool
	var seenTrxTraces *pbcodec.TransactionTrace
	var producers []string

	atomic.AddInt64(&runningPushInHandoffs, 1)
	zlog.Info("waiting for trx to live for handoffs", zap.Int("handoffs", requiredHandoffs), zap.String("trx_id", trxID), zap.Int64("count", atomic.LoadInt64(&runningPushInHandoffs)))

	handle := bstream.HandlerFunc(func(block *bstream.Block, obj interface{}) error {
		fObj := obj.(*forkable.ForkableObject)

		zlog.Debug("handoff awaiting processing", zap.Stringer("block", block), zap.Stringer("step", fObj.Step))
		if done {
			return nil
		}

		blk := block.ToNative().(*pbcodec.Block)
		producer := blk.Header.Producer

		switch fObj.Step {
		case forkable.StepIrreversible, forkable.StepStalled:
			return nil

		case forkable.StepNew, forkable.StepRedo:
			if trxTraces := traceExecutedInBlock(trxID, blk); trxTraces != nil {
				seenTrxTraces = trxTraces
			}

			if seenTrxTraces == nil {
				break
			}

			producers = append(producers, producer) // push
			if countUniqueElem(producers)-1 >= requiredHandoffs {
				trxFound <- seenTrxTraces
				done = true
			}

		case forkable.StepUndo:
			if seenTrxTraces != nil && len(producers) > 0 {
				producers = producers[:len(producers)-1] // pop
			}
			if trxTraces := traceExecutedInBlock(trxID, blk); trxTraces != nil {
				seenTrxTraces = nil
			}

		default:
			return fmt.Errorf("unhandled forkable step")
		}

		return nil
	})

	forkHandler := forkable.New(handle, forkable.WithExclusiveLIB(bstream.BlockRefFromID(libID)))
	forkablePostGate := bstream.NewBlockIDGate(libID, bstream.GateInclusive, forkHandler)

	source := subscriptionHub.NewSourceFromBlockRef(bstream.BlockRefFromID(libID), forkablePostGate)
	source.OnTerminating(func(e error) {
		atomic.AddInt64(&runningPushInHandoffs, -1)
	})

	go source.Run()

	return trxFound, source.Shutdown
}

var runningPushInBlock int64

func awaitTransactionInBlock(ctx context.Context, trxID string, sourceFactory bstream.SourceFactory) (<-chan *pbcodec.TransactionTrace, func(error)) {
	atomic.AddInt64(&runningPushInBlock, 1)
	zlog.Info("waiting for trx to appear in a block", zap.String("trxID", trxID), zap.Int64("count", atomic.LoadInt64(&runningPushInBlock)))

	trxTraceFoundChan := make(chan *pbcodec.TransactionTrace)

	source := sourceFactory(getTransactionCatcher(ctx, trxID, trxTraceFoundChan))
	source.OnTerminating(func(e error) {
		atomic.AddInt64(&runningPushInBlock, -1)
	})
	go source.Run()

	return trxTraceFoundChan, source.Shutdown
}

var runningIrreversible int64

func awaitTransactionIrreversible(ctx context.Context, trxID string, sourceFactory bstream.SourceFactory) (<-chan *pbcodec.TransactionTrace, func(error)) {
	atomic.AddInt64(&runningIrreversible, 1)
	zlog.Info("waiting for trx to appear in an irreversible block", zap.String("trxID", trxID), zap.Int64("count", atomic.LoadInt64(&runningIrreversible)))

	trxTraceFoundChan := make(chan *pbcodec.TransactionTrace)

	irrForkableHandler := forkable.New(getTransactionCatcher(ctx, trxID, trxTraceFoundChan), forkable.WithFilters(forkable.StepIrreversible))
	source := sourceFactory(irrForkableHandler)
	source.OnTerminating(func(e error) {
		atomic.AddInt64(&runningIrreversible, -1)
	})

	go source.Run()
	return trxTraceFoundChan, source.Shutdown
}

func getTransactionCatcher(ctx context.Context, trxID string, trxTraceFoundChan chan *pbcodec.TransactionTrace) bstream.Handler {
	return bstream.HandlerFunc(func(block *bstream.Block, obj interface{}) error {
		blk := block.ToNative().(*pbcodec.Block)
		trxTrace := traceExecutedInBlock(trxID, blk)
		if trxTrace != nil {
			select {
			case <-ctx.Done():
				return nil
			case trxTraceFoundChan <- trxTrace:
				return nil
			}
		}
		return nil
	})
}

func traceExecutedInBlock(trxID string, blk *pbcodec.Block) *pbcodec.TransactionTrace {
	for _, trxTrace := range blk.TransactionTraces {
		if trxTrace.Id == trxID {
			return trxTrace
		}
	}

	return nil
}

func isNotPushTransaction(url, pushGuaranteeHeaderOption string) bool {
	return (url != "/v1/chain/push_transaction" && url != "/v1/chain/send_transaction") || pushGuaranteeHeaderOption == ""
}
