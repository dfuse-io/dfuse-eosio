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
	"math/rand"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/streamingfast/bstream"
	"github.com/streamingfast/bstream/forkable"
	"github.com/streamingfast/bstream/hub"
	"github.com/dfuse-io/dfuse-eosio/codec"
	"github.com/dfuse-io/dfuse-eosio/eosws"
	"github.com/dfuse-io/dfuse-eosio/eosws/mdl"
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

	switch r.URL.EscapedPath() {
	case "/v1/chain/push_transaction", "/v1/chain/send_transaction":
		if r.Header.Get("X-Eos-Push-Guarantee") != "" {
			t.pushTransactionHandler.ServeHTTP(w, r)
			return
		}
		//case "/v1/chain/get_info":
		//return
	}
	t.dumbAPIProxy.ServeHTTP(w, r)
}

////// PUSHER

type TxPusher struct {
	API             *eos.API
	extraAPIs       []*eos.API
	subscriptionHub *hub.SubscriptionHub
	headInfoHub     *eosws.HeadInfoHub
	retries         int
}

type PushResponse struct {
	TransactionID string          `json:"transaction_id"`
	BlockID       string          `json:"block_id"`
	BlockNum      uint32          `json:"block_num"`
	Processed     json.RawMessage `json:"processed"`
}

func NewTxPusher(API *eos.API, subscriptionHub *hub.SubscriptionHub, headInfoHub *eosws.HeadInfoHub, retries int, extraAPIs []*eos.API) *TxPusher {
	return &TxPusher{
		API:             API,
		subscriptionHub: subscriptionHub,
		headInfoHub:     headInfoHub,
		retries:         retries,
		extraAPIs:       append(extraAPIs, API), // always include the base API in here
	}
}

func (t *TxPusher) randomAPI() *eos.API {
	return t.extraAPIs[rand.Intn(len(t.extraAPIs))]
}

func (t *TxPusher) tryPush(API *eos.API, ctx context.Context, tx *eos.PackedTransaction, trxID string, useLegacyPush bool) (err error) {
	timedoutContext, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	var pushResp json.RawMessage

	if useLegacyPush {
		pushResp, err = API.PushTransactionRaw(timedoutContext, tx)
	} else {
		pushResp, err = API.SendTransactionRaw(timedoutContext, tx)
	}
	if err != nil {
		return
	}

	idCheck := gjson.GetBytes(pushResp, "transaction_id").String()
	if idCheck != trxID {
		return fmt.Errorf("pushed transaction ID %q mismatch transaction ID received from API %q", trxID, idCheck)
	}
	return nil
}

func isExpiredError(err eos.APIError) bool {
	return err.ErrorStruct.Code == 3040005
}

func isDuplicateError(err eos.APIError) bool {
	return err.ErrorStruct.Code == 3040008 || err.ErrorStruct.Code == 3040009 // duplicate
}

func isRetryable(err eos.APIError) bool {
	if err.ErrorStruct.Code < 3080000 || err.ErrorStruct.Code == 3080001 {
		return false
	}
	// in between those are resource-related errors, like cpu or deadline, we want to retry those
	// see https://docs.google.com/spreadsheets/d/1uHeNDLnCVygqYK-V01CFANuxUwgRkNkrmeLm9MLqu9c/edit#gid=0
	if err.ErrorStruct.Code >= 3090000 {
		return false
	}
	return true
}

type fakePackedTrx struct {
	Signatures        []string `json:"signatures"`
	PackedTransaction string   `json:"packed_trx"`
}

func isValidJSON(payload []byte) bool {
	var fakePacked *fakePackedTrx
	return json.Unmarshal(payload, &fakePacked) == nil
}

func (t *TxPusher) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	guarantee := r.Header.Get("X-Eos-Push-Guarantee")
	pushOutput := r.Header.Get("X-Eos-Push-Guarantee-Output-Inline-Traces")

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
	if err != nil {
		if isValidJSON(incomingContent) {
			writeDetailedAPIError(err, "Internal Service Error", 3010010, "packed_transaction_type_exception", "Invalid packed transaction", "Invalid packed transaction", w)
			return
		}
		writeDetailedAPIError(err, "Internal Service Error", 4, "parse_error_exception", "Parse Error", err.Error(), w)
		return
	}

	trxIDCheckSum, err := tx.ID()
	if err != nil {
		writeDetailedAPIError(err, "Internal Service Error", 3010010, "packed_transaction_type_exception", "Invalid packed transaction", "Invalid packed transaction", w)
		return
	}

	trxID := trxIDCheckSum.String()

	liveSourceFactory := bstream.SourceFactory(func(handler bstream.Handler) bstream.Source {
		return t.subscriptionHub.NewSource(handler, 10) // does not need joining
	})

	var trxTraceFoundChan <-chan *pbcodec.TransactionTrace
	var shutdownFunc func(error)
	expirationDelay := time.Minute * 2 //baseline for inblock inclusion
	normalizedGuarantee := guarantee
	switch guarantee {
	case "in-block":
		trxTraceFoundChan, shutdownFunc = awaitTransactionInBlock(ctx, trxID, liveSourceFactory)
	case "handoff:1", "handoffs:1":
		normalizedGuarantee = "handoffs:1"
		expirationDelay += 1 * time.Minute
		trxTraceFoundChan, shutdownFunc = awaitTransactionPassedHandoffs(ctx, t.headInfoHub.LibID(), trxID, 1, t.subscriptionHub)
	case "handoff:2", "handoffs:2":
		normalizedGuarantee = "handoffs:2"
		expirationDelay += 1 * time.Minute
		trxTraceFoundChan, shutdownFunc = awaitTransactionPassedHandoffs(ctx, t.headInfoHub.LibID(), trxID, 2, t.subscriptionHub)
	case "handoff:3", "handoffs:3":
		normalizedGuarantee = "handoffs:3"
		expirationDelay += 1 * time.Minute
		trxTraceFoundChan, shutdownFunc = awaitTransactionPassedHandoffs(ctx, t.headInfoHub.LibID(), trxID, 3, t.subscriptionHub)
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

	maxAttempts := t.retries + 1
	for attempt := 1; ; attempt++ {
		err = t.tryPush(t.API, ctx, tx, trxID, r.URL.EscapedPath() == "/v1/chain/push_transaction")
		if err == nil {
			break
		}

		if apiErr, ok := err.(eos.APIError); ok { // decoded nodeos API error
			retryable := isRetryable(apiErr)
			if apiErrCnt, err := json.Marshal(apiErr); err == nil {
				zapFields := append(
					logFieldsFromAPIErr(apiErr),
					zap.Bool("retryable", retryable),
					zap.Int("attempt", attempt),
					zap.Int("max_attempts", maxAttempts),
					zap.String("trx_id", trxID),
				)
				zlog.Info("push transaction API error",
					zapFields...,
				)
				if attempt < maxAttempts && retryable {
					time.Sleep(time.Duration(attempt) * 500 * time.Millisecond)
					continue
				}
				w.WriteHeader(apiErr.Code)
				w.Write(apiErrCnt)
				return
			}
		}
		// other error, we couldn't reach nodeos...
		zlog.Info("push transaction unknown error",
			zap.Error(err),
			zap.Int("attempt", attempt),
			zap.Int("max_attempts", maxAttempts),
		)
		if attempt < maxAttempts {
			time.Sleep(time.Duration(attempt) * 250 * time.Millisecond)
			continue
		}
		checkHTTPError(err, fmt.Sprintf("cannot push transaction %q to Nodeos API.", trxID), eoserr.ErrUnhandledException, w)
		return
	}

	zlog.Debug("waiting for trx to appear in a block", zap.String("hexTrxID", trxID), zap.Float64("minutes", expirationDelay.Minutes()), zap.String("guarantee", guarantee))

	resend := 0
	trxExpired := false
	expiration := time.After(expirationDelay)
	for {
		select {
		case <-time.After(time.Second * 8): // retries every 8 second if we haven't seen the trx yet, this means at 8, 16, 24 -- considering that most trxs have 30s deadline
			if trxExpired {
				continue // keep waiting for the transaction to appear in a block but we stop trying to push it
			}
			a := t.randomAPI()
			err = t.tryPush(a, ctx, tx, trxID, r.URL.EscapedPath() == "/v1/chain/push_transaction")
			zlog.Debug("retrying send transaction to push API", zap.String("random_api", a.BaseURL), zap.Error(err), zap.Int("resend", resend))
			resend++
			if err != nil {
				if apiErr, ok := err.(eos.APIError); ok { // decoded nodeos API error
					if isExpiredError(apiErr) {
						trxExpired = true
						zlog.Debug("trx expired error.", zap.String("trx_id", trxID))
						continue
					}
					if isDuplicateError(apiErr) {
						zlog.Debug("duplicate error.", zap.String("trx_id", trxID))
						continue
					}
					if isRetryable(apiErr) {
						zlog.Debug("retryable error", zap.String("trx_id", trxID))
						continue
					}

					zapFields := append(
						logFieldsFromAPIErr(apiErr),
						zap.Int("resend", resend),
						zap.String("trx_id", trxID),
					)
					zlog.Info("push transaction API error after earlier success",
						zapFields...,
					)

					// if previously passing transaction now fails, we return it to the client.
					if apiErrCnt, err := json.Marshal(apiErr); err == nil {
						w.WriteHeader(apiErr.Code)
						w.Write(apiErrCnt)
					} else {
						checkHTTPError(errors.New("unknown error"), "unknown error", eoserr.ErrUnhandledException, w)
					}
					return
				}
			}

		case <-expiration:
			metrics.TimedOutPushTrxCount.Inc(normalizedGuarantee)
			msg := fmt.Sprintf("too long waiting for inclusion of %q into a block (after %d retries)", trxID, resend)
			checkHTTPError(errors.New(msg), msg, eoserr.ErrTimeoutException, w)
			return

		case trxTrace := <-trxTraceFoundChan:
			blockID := trxTrace.ProducerBlockId

			var processed json.RawMessage
			if pushOutput == "true" {
				v1tr, err := mdl.ToV1TransactionTrace(trxTrace)
				if checkHTTPError(err, "cannot marshal response", eoserr.ErrUnhandledException, w) {
					return
				}
				out, err := json.Marshal(v1tr)
				if checkHTTPError(err, "cannot marshal response", eoserr.ErrUnhandledException, w) {
					return
				}
				processed = out
			} else {
				eosTrace := codec.TransactionTraceToEOS(trxTrace)
				out, err := json.Marshal(eosTrace)
				if checkHTTPError(err, "cannot marshal response", eoserr.ErrUnhandledException, w) {
					return
				}
				processed = out
			}

			resp := &PushResponse{
				BlockID:       blockID,
				BlockNum:      eos.BlockNum(blockID),
				Processed:     processed,
				TransactionID: trxID,
			}

			out, err := json.Marshal(resp)
			if checkHTTPError(err, "cannot marshal response", eoserr.ErrUnhandledException, w) {
				return
			}

			metrics.SucceededPushTrxCount.Inc(normalizedGuarantee)
			w.Header().Set("content-length", fmt.Sprintf("%d", len(out)))
			w.Write([]byte(out))
			return
		}
	}
}

func writeDetailedAPIError(err error, msg string, errorCode int, errorName, errorWhat, detailMessage string, w http.ResponseWriter) {
	fields := []zap.Field{
		zap.Error(err),
		zap.Int("errorCode", errorCode),
		zap.String("errorName", errorName),
		zap.String("errorWhat", errorWhat),
	}

	zlog.Info("push transaction error, "+msg, fields...)

	apiError := &eos.APIError{
		Code:    500,
		Message: msg,
	}
	apiError.ErrorStruct.Code = errorCode
	apiError.ErrorStruct.Name = errorName
	apiError.ErrorStruct.What = errorWhat
	apiError.ErrorStruct.Details = []eos.APIErrorDetail{
		eos.APIErrorDetail{
			File:       "",
			LineNumber: 0,
			Message:    detailMessage,
			Method:     "proxied_by_dfuse",
		},
	}

	out, _ := json.Marshal(apiError)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(500)
	w.Write([]byte(out))

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

	irrRef := bstream.NewBlockRefFromID(libID)
	forkHandler := forkable.New(handle, forkable.WithLogger(zlog), forkable.WithExclusiveLIB(irrRef))
	forkablePostGate := bstream.NewBlockIDGate(libID, bstream.GateInclusive, forkHandler, bstream.GateOptionWithLogger(zlog))

	source := subscriptionHub.NewSourceFromBlockRef(irrRef, forkablePostGate)
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

	irrForkableHandler := forkable.New(getTransactionCatcher(ctx, trxID, trxTraceFoundChan), forkable.WithLogger(zlog), forkable.WithFilters(forkable.StepIrreversible))
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
	for _, trxTrace := range blk.TransactionTraces() {
		if trxTrace.Id == trxID {
			return trxTrace
		}
	}

	return nil
}

func logFieldsFromAPIErr(apiErr eos.APIError) []zap.Field {
	return []zap.Field{
		zap.String("name", apiErr.ErrorStruct.Name),
		zap.Int("code", apiErr.Code),
		zap.Int("errstruct_code", apiErr.ErrorStruct.Code),
		zap.String("error_message", apiErr.Message),
		zap.String("what", apiErr.ErrorStruct.What),
		zap.Any("details", apiErr.ErrorStruct.Details),
	}
}
