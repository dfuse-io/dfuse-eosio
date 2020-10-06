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
	"net/http"
	"net/http/httputil"
	"net/url"

	stackdriverPropagation "contrib.go.opencensus.io/exporter/stackdriver/propagation"
	"github.com/dfuse-io/dmetering"
	"go.opencensus.io/plugin/ochttp"
	"go.uber.org/zap"
)

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

func NewReverseProxy(target *url.URL, stripQuerystring bool) *httputil.ReverseProxy {
	director := func(req *http.Request) {
		if stripQuerystring {
			req.URL.RawQuery = ""
		}
		req.URL.Scheme = target.Scheme
		req.URL.Host = target.Host
		req.Host = target.Host
		deleteCORSHeaders(req)
		req.Header.Set("Host", target.Host)
		if _, ok := req.Header["User-Agent"]; !ok {
			// explicitly disable User-Agent so it's not set to default value
			req.Header.Set("User-Agent", "")
		}
	}

	return &httputil.ReverseProxy{
		Director: director,
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			zlog.Info("REST error",
				zap.String("path", r.URL.Path),
				zap.String("method", r.Method),
				zap.String("host", r.URL.Host),
				zap.Error(err),
			)
			w.WriteHeader(http.StatusBadGateway)
		},

		ModifyResponse: func(response *http.Response) error {
			ctx := response.Request.Context()

			zlog.Info("REST response",
				zap.String("path", response.Request.URL.Path),
				zap.String("method", response.Request.Method),
				zap.String("host", response.Request.URL.Host),
				zap.Int("response_code", response.StatusCode),
				zap.String("response_status", response.Status),
			)
			response.Header.Del("X-Trace-ID")

			//////////////////////////////////////////////////////////////////////
			// Billable event on REST API endpoint
			// WARNING: Ingress / Egress bytess is taken care by the middleware
			//////////////////////////////////////////////////////////////////////
			//TODO: WARNING - /v0/state (StateDB) bill one document even though they may be very large
			dmetering.EmitWithContext(dmetering.Event{
				Source:         "eosws",
				Kind:           "REST API - Chain State",
				Method:         response.Request.URL.Path,
				RequestsCount:  1,
				ResponsesCount: 1,
			}, ctx)
			//////////////////////////////////////////////////////////////////////

			return nil
		},
		Transport: &ochttp.Transport{
			Base: &http.Transport{
				DisableKeepAlives: true,
			},
			Propagation: &stackdriverPropagation.HTTPFormat{},
		},
	}
}
