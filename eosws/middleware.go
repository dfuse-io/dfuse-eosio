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
	"compress/gzip"
	"context"
	"encoding/json"
	"net/http"

	stackdriverPropagation "contrib.go.opencensus.io/exporter/stackdriver/propagation"
	"github.com/streamingfast/derr"
	"github.com/dfuse-io/logging"
	"github.com/eoscanada/eos-go"
	"github.com/eoscanada/eos-go/eoserr"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/streamingfast/dauth/authenticator"
	"go.opencensus.io/plugin/ochttp"
	"go.opencensus.io/trace"
	"go.uber.org/zap"
)

type AuthFeatureChecker = func(ctx context.Context, credentials authenticator.Credentials) error

type AuthFeatureMiddleware struct {
	checker AuthFeatureChecker
}

func CompressionMiddleware(next http.Handler) http.Handler {
	return handlers.CompressHandlerLevel(next, gzip.BestSpeed)
}

func OpenCensusMiddleware(next http.Handler) http.Handler {
	return &ochttp.Handler{
		Handler:     next,
		Propagation: &stackdriverPropagation.HTTPFormat{},
	}
}

func LoggingMiddleware(next http.Handler) http.Handler {
	return &logging.Handler{
		Next:        next,
		Propagation: &stackdriverPropagation.HTTPFormat{},
		RootLogger:  zlog,
	}
}

func RESTTrackingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		TrackUserEvent(ctx, "rest_request",
			"method", r.Method,
			"host", r.Host,
			"path", r.URL.Path,
			"encoded_query", r.URL.Query().Encode(),
		)

		next.ServeHTTP(w, r)
	})
}

func PreTrackingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		zlogger := logging.Logger(r.Context(), zlog)
		zlogger.Debug("handling HTTP request",
			zap.String("method", r.Method),
			zap.Any("host", r.Host),
			zap.Any("url", r.URL),
			zap.Any("headers", r.Header),
		)

		ctx := r.Context()
		span := trace.FromContext(ctx)
		if span == nil {
			zlogger.Error("trace is not present in request but should have been")
		}

		spanContext := span.SpanContext()
		traceID := spanContext.TraceID.String()

		w.Header().Set("X-Trace-ID", traceID)

		next.ServeHTTP(w, r)
	})
}

func NewCORSMiddleware() mux.MiddlewareFunc {

	allowedHeaders := handlers.AllowedHeaders([]string{"X-Requested-With", "Content-Type", "Authorization", "X-Eos-Push-Guarantee"})
	allowedOrigins := handlers.AllowedOrigins([]string{"*"})
	allowedMethods := handlers.AllowedMethods([]string{"GET", "HEAD", "POST", "OPTIONS"})
	maxAge := handlers.MaxAge(86400) // 24 hours - hard capped by Firefox / Chrome is max 10 minutes

	return handlers.CORS(allowedHeaders, allowedOrigins, allowedMethods, maxAge)
}

func NewAuthFeatureMiddleware(checker AuthFeatureChecker) *AuthFeatureMiddleware {
	return &AuthFeatureMiddleware{
		checker: checker,
	}
}

func (middleware *AuthFeatureMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		credentials := authenticator.GetCredentials(ctx)
		if credentials == nil {
			derr.WriteError(ctx, w, "credentials unavailable from context but should have been", derr.UnexpectedError(ctx, nil))
			return
		}

		err := middleware.checker(ctx, credentials)
		if err != nil {
			derr.WriteError(ctx, w, "request not authorized to perform this action", err)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func DfuseErrorHandler(w http.ResponseWriter, ctx context.Context, err error) {
	derr.WriteError(ctx, w, "unable to authorize request", AuthInvalidTokenError(ctx, err, ""))
}

func EOSChainErrorHandler(w http.ResponseWriter, ctx context.Context, err error) {
	apiError := eos.NewAPIError(401, "this feature requires a dfuse API key (https://dfuse.io)", eoserr.ErrUnhandledException)
	zlog.Warn("chain Error", zap.Error(apiError))

	w.WriteHeader(http.StatusUnauthorized)
	json.NewEncoder(w).Encode(apiError)
}
