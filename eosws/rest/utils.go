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
	"bytes"
	"context"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"

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

type ReverseProxy struct {
	retries          int
	target           *url.URL
	stripQuerystring bool
	dmeteringKind    string
}

func (p *ReverseProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	for attempt := 1; ; attempt++ {
		if p.tryReq(w, r, attempt > p.retries) {
			return
		}
		time.Sleep(time.Duration(attempt) * 250 * time.Millisecond)
	}
}

func (p *ReverseProxy) tryReq(w http.ResponseWriter, r *http.Request, failDirectly bool) (written bool) {
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	req := r.Clone(ctx)
	if p.stripQuerystring {
		req.URL.RawQuery = ""
	}

	var b bytes.Buffer
	b.ReadFrom(r.Body)
	r.Body = ioutil.NopCloser(&b)
	req.Body = ioutil.NopCloser(bytes.NewReader(b.Bytes()))

	req.RequestURI = ""
	req.URL.Scheme = p.target.Scheme
	req.URL.Host = p.target.Host
	req.Host = p.target.Host
	deleteCORSHeaders(req)
	req.Header.Set("Host", p.target.Host)
	if _, ok := req.Header["User-Agent"]; !ok {
		// explicitly disable User-Agent so it's not set to default value
		req.Header.Set("User-Agent", "")
	}

	client := &http.Client{
		Transport: &ochttp.Transport{
			Base: &http.Transport{
				DisableKeepAlives: true,
			},
			Propagation: &stackdriverPropagation.HTTPFormat{},
		},
	}

	resp, err := client.Do(req)
	if err != nil {
		zlog.Info("REST error",
			zap.String("path", r.URL.Path),
			zap.String("method", r.Method),
			zap.String("host", r.URL.Host),
			zap.Bool("fail_directly", failDirectly),
			zap.Error(err),
		)
		if failDirectly {
			w.WriteHeader(http.StatusBadGateway)
			return true
		}
		return false
	}

	body, bodyErr := ioutil.ReadAll(resp.Body)
	if bodyErr != nil {
		zlog.Info("REST error reading body",
			zap.String("path", r.URL.Path),
			zap.String("method", r.Method),
			zap.String("host", r.URL.Host),
			zap.Bool("fail_directly", failDirectly),
			zap.Error(bodyErr),
		)
		if failDirectly {
			w.WriteHeader(http.StatusBadGateway)
			return true
		}
		return false
	}

	if resp.StatusCode >= 500 {
		zlog.Info("REST error from backend",
			zap.String("path", r.URL.Path),
			zap.String("method", r.Method),
			zap.String("host", r.URL.Host),
			zap.Bool("fail_directly", failDirectly),
			zap.Error(err),
		)
		if failDirectly {
			w.WriteHeader(resp.StatusCode)
			w.Write(body)
			return true
		}
		return false
	}
	_, err = w.Write(body)
	if err != nil {
		zlog.Info("REST error writing to client",
			zap.String("path", r.URL.Path),
			zap.String("method", r.Method),
			zap.String("host", r.URL.Host),
			zap.Bool("fail_directly", failDirectly),
			zap.Error(err),
		)
		return true
	}

	// on success
	zlog.Info("REST response",
		zap.String("path", r.URL.Path),
		zap.String("method", r.Method),
		zap.String("host", r.URL.Host),
		zap.Int("response_code", resp.StatusCode),
		zap.String("response_status", resp.Status),
	)

	resp.Header.Del("X-Trace-ID")

	//////////////////////////////////////////////////////////////////////
	// Billable event on REST API endpoint
	// WARNING: Ingress / Egress bytess is taken care by the middleware
	//////////////////////////////////////////////////////////////////////
	//TODO: WARNING - /v0/state (StateDB) bill one document even though they may be very large
	dmetering.EmitWithContext(dmetering.Event{
		Source:         "eosws",
		Kind:           p.dmeteringKind,
		Method:         r.URL.Path,
		RequestsCount:  1,
		ResponsesCount: 1,
	}, ctx)
	//////////////////////////////////////////////////////////////////////

	return true

}

func NewReverseProxy(target *url.URL, stripQuerystring bool, dmeteringKind string, retries int) http.Handler {
	return &ReverseProxy{
		retries:          retries,
		target:           target,
		stripQuerystring: stripQuerystring,
		dmeteringKind:    dmeteringKind,
	}
}
