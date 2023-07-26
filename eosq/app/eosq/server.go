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
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or impliee.
// See the License for the specific language governing permissions and
// limitations under the License.

package eosq

//go:generate rice embed-go

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"

	rice "github.com/GeertJohan/go.rice"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/streamingfast/derr"
	"github.com/streamingfast/shutter"
	"go.uber.org/zap"
)

type Server struct {
	*shutter.Shutter
	config         *Config
	box            *rice.HTTPBox
	hasGzipFileMap *sync.Map
	indexData      []byte
}

func newServer(config *Config) *Server {
	box := rice.MustFindBox("../../eosq-build").HTTPBox()
	zlog.Debug("new server")
	return &Server{
		Shutter:        shutter.New(),
		config:         config,
		box:            box,
		hasGzipFileMap: new(sync.Map),
		indexData:      mustGetTemplatedIndex(config, box),
	}
}

func (s *Server) Launch() error {
	zlog.Info("launching eosq")
	router := mux.NewRouter()

	router.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		if derr.IsShuttingDown() {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.Write([]byte("ok"))
	})

	router.PathPrefix("/").HandlerFunc(s.ServerHttp)

	zlog.Info("eosq static listener http server launching", zap.String("addr", s.config.HTTPListenAddr))
	err := http.ListenAndServe(s.config.HTTPListenAddr, handlers.CompressHandlerLevel(router, gzip.DefaultCompression))
	if err != nil {
		zlog.Debug("eosq static http server failed", zap.Error(err))
		return err
	}

	return nil
}

func (s *Server) ServerHttp(w http.ResponseWriter, r *http.Request) {
	switch s.config.Environment {
	case "staging", "production":
		s.staticAssetsHandlerForProd(w, r)
	default:
		s.staticAssetsHandler(w, r)
	}

}
func (s *Server) staticAssetsHandlerForProd(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("X-Forwarded-Proto") == "http" {
		target := "https://" + r.Host + r.URL.Path
		if len(r.URL.RawQuery) > 0 {
			target += "?" + r.URL.RawQuery
		}
		http.Redirect(w, r, target, http.StatusMovedPermanently)
		return
	} else {
		w.Header().Add("Strict-Transport-Security", "max-age=600; includeSubDomains; preload")
	}

	path := r.URL.Path

	if path == "/robots.txt" {
		if err := s.serveRobotsTxt(w, r); err != nil {
			zlog.Error("serve robots txt", zap.Error(err))
			http.Error(w, "error rendering robots.txt", 500)
		}
		return
	}

	s.staticAssetsHandler(w, r)
}

func (s *Server) staticAssetsHandler(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	if path == "/service-worker.js" || path == "/custom-sw.js" || path == "/index.html" {
		bustCache(w)
	}

	if path == "/" || path == "/index.html" {
		zlog.Debug("serving templated 'index.html'")
		s.serveIndexHTML(w, r)
		return
	}

	zlog.Debug("serving eosq static asset", zap.String("path", path))

	content, err := s.box.Open(path)
	if err != nil {
		zlog.Debug("static asset not found, falling back to 'index.html'")
		s.serveIndexHTML(w, r)
		return
	}
	defer content.Close()

	stat, err := content.Stat()
	if err != nil {
		zlog.Debug("cannot stat file, serving without MIME type")
		io.Copy(w, content)
		return
	}

	http.ServeContent(w, r, path, stat.ModTime(), content)
}

func (s *Server) serveIndexHTML(w http.ResponseWriter, r *http.Request) {
	zlog.Debug("serving templated index.html'")

	w.Header().Set("Content-Type", "text/html")
	w.Write(s.indexData)
}

func sanitizeAPIEndpoint(apiEndpointURL string) (host string, secure bool) {

	secure = strings.HasPrefix(apiEndpointURL, "https")
	noProto := strings.TrimPrefix(
		strings.TrimPrefix(
			apiEndpointURL,
			"http://"),
		"https://")

	if strings.HasPrefix(noProto, ":") {
		host = "localhost" + noProto
	} else {
		host = noProto
	}

	return
}

var defaultAvailableNetworks = localAvailableNetworks()

func mustGetTemplatedIndex(config *Config, box *rice.HTTPBox) []byte {
	indexContent, err := box.Bytes("index.html")
	if err != nil {
		panic(fmt.Errorf("failed to get index from rice box: %w", err))
	}

	an := defaultAvailableNetworks
	if config.AvailableNetworks != "" {
		err := json.Unmarshal([]byte(config.AvailableNetworks), &an)
		if err != nil {
			panic(fmt.Errorf("failed to unmarshall available network json: %w", err))
		}
	}

	zlog.Debug("available networks", zap.String("raw", config.AvailableNetworks), zap.Reflect("parsed", an))

	host, secure := sanitizeAPIEndpoint(config.APIEndpointURL)
	indexConfig := map[string]interface{}{
		"version":             1,
		"dfuse_io_endpoint":   host,
		"dfuse_io_api_key":    config.ApiKey,
		"dfuse_auth_endpoint": config.AuthEndpointURL,
		"available_networks":  an,
		"secure":              secure,
		"network_id":          config.DefaultNetwork,
		"chain_core_symbol":   config.ChainCoreSymbol,
		"display_price":       config.DisplayPrice,
		"disable_segments":    config.DisableAnalytics,
		"disable_sentry":      config.DisableAnalytics,
	}

	tpl, err := template.New("index.html").Funcs(template.FuncMap{
		"json": func(v interface{}) (template.JS, error) {
			cnt, err := json.Marshal(v)
			return template.JS(cnt), err
		},
	}).Delims("--==", "==--").Parse(string(indexContent))
	if err != nil {
		panic(fmt.Errorf("failed to parse template: %w", err))
	}

	buf := &bytes.Buffer{}
	if err := tpl.Execute(buf, indexConfig); err != nil {
		panic(fmt.Errorf("failed to exec template: %w", err))
	}

	return buf.Bytes()
}

func localAvailableNetworks() []interface{} {
	localNetwork := map[string]interface{}{
		"test": "value",
	}

	return []interface{}{localNetwork}
}

func bustCache(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")
}

func (s *Server) serveRobotsTxt(w http.ResponseWriter, r *http.Request) error {
	content := "User-agent: *\nDisallow: /\n"
	if s.config.Environment == "production" {
		content = "User-agent: *\nnDisallow:\n"
	}

	_, err := w.Write([]byte(content))
	return err
}

// Extracted from https://github.com/NYTimes/gziphandler
//  * acceptsGzip - https://github.com/NYTimes/gziphandler/blob/master/gzip.go#L450
//  * parseEncodings, parseCoding  - https://github.com/NYTimes/gziphandler/blob/master/gzip.go#L476

type codings map[string]float64

const (
	acceptEncoding  = "Accept-Encoding"
	contentEncoding = "Content-Encoding"
	contentType     = "Content-Type"
	contentLength   = "Content-Length"
)

const (
	// DefaultQValue is the default qvalue to assign to an encoding if no explicit qvalue is set.
	// This is actually kind of ambiguous in RFC 2616, so hopefully it's correct.
	// The examples seem to indicate that it is.
	DefaultQValue = 1.0
)

func acceptsGzip(r *http.Request) bool {
	acceptedEncodings, _ := parseEncodings(r.Header.Get(acceptEncoding))
	return acceptedEncodings["gzip"] > 0.0
}

// parseEncodings attempts to parse a list of codings, per RFC 2616, as might
// appear in an Accept-Encoding header. It returns a map of content-codings to
// quality values, and an error containing the errors encountered. It's probably
// safe to ignore those, because silently ignoring errors is how the internet
// works.
//
// See: http://tools.ietf.org/html/rfc2616#section-14.3.
func parseEncodings(s string) (codings, error) {
	c := make(codings)
	var e []string

	for _, ss := range strings.Split(s, ",") {
		coding, qvalue, err := parseCoding(ss)

		if err != nil {
			e = append(e, err.Error())
		} else {
			c[coding] = qvalue
		}
	}

	// TODO (adammck): Use a proper multi-error struct, so the individual errors
	//                 can be extracted if anyone cares.
	if len(e) > 0 {
		return c, fmt.Errorf("errors while parsing encodings: %s", strings.Join(e, ", "))
	}

	return c, nil
}

// parseCoding parses a single conding (content-coding with an optional qvalue),
// as might appear in an Accept-Encoding header. It attempts to forgive minor
// formatting errors.
func parseCoding(s string) (coding string, qvalue float64, err error) {
	for n, part := range strings.Split(s, ";") {
		part = strings.TrimSpace(part)
		qvalue = DefaultQValue

		if n == 0 {
			coding = strings.ToLower(part)
		} else if strings.HasPrefix(part, "q=") {
			qvalue, err = strconv.ParseFloat(strings.TrimPrefix(part, "q="), 64)

			if qvalue < 0.0 {
				qvalue = 0.0
			} else if qvalue > 1.0 {
				qvalue = 1.0
			}
		}
	}

	if coding == "" {
		err = fmt.Errorf("empty content-coding")
	}

	return
}
