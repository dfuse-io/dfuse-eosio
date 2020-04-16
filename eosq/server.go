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
	"encoding/json"
	"html/template"
	"io"
	"net/http"
	"time"

	rice "github.com/GeertJohan/go.rice"
	"github.com/dfuse-io/shutter"
	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

type Server struct {
	*shutter.Shutter
	config *Config
	box    *rice.HTTPBox
}

func newServer(config *Config) *Server {
	return &Server{
		Shutter: shutter.New(),
		config:  config,
		box:     rice.MustFindBox("build").HTTPBox(),
	}
}

func (s *Server) Launch() error {
	router := mux.NewRouter()
	router.Path("/v1/auth/issue").Methods("POST").HandlerFunc(authIssueHandler)
	router.PathPrefix("/").HandlerFunc(s.staticAssetsHandler)

	zlog.Debug("eosq static listener http server launching", zap.String("addr", s.config.HttpListenAddr))
	err := http.ListenAndServe(s.config.HttpListenAddr, router)
	if err != nil {
		zlog.Debug("eosq static http server failed", zap.Error(err))
		return err
	}

	return nil
}

func authIssueHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	expiresAtInSeconds := time.Now().Unix() + (180 * 24 * 60 * 60)
	authResponse := map[string]interface{}{
		"token":      "a.b.c",
		"expires_at": expiresAtInSeconds,
	}

	zlog.Debug("serving dummy JWT", zap.Any("jwt", authResponse))
	enc := json.NewEncoder(w)

	if err := enc.Encode(authResponse); err != nil {
		zlog.Error("serving dummy JWT", zap.Error(err))
	}
	return
}

func (s *Server) staticAssetsHandler(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	if path == "/" || path == "/index.html" {
		zlog.Debug("serving templated 'index.html'")
		s.serveIndexHTML(w, r)
		return
	}

	zlog.Debug("serving eosq static asset", zap.String("path", path))
	pathFile, err := s.box.Open(path)

	if err != nil {
		zlog.Debug("static asset not found, falling back to 'index.html'")
		s.serveIndexHTML(w, r)
		return
	}
	defer pathFile.Close()

	stat, err := pathFile.Stat()
	if err != nil {
		zlog.Debug("cannot stat file, serving without MIME type")
		io.Copy(w, pathFile)
		return
	}

	http.ServeContent(w, r, path, stat.ModTime(), pathFile)
}

func (s *Server) serveIndexHTML(w http.ResponseWriter, r *http.Request) {
	zlog.Debug("serving templated index.html'")
	reader, err := s.templatedIndex()

	if err != nil {
		zlog.Error("unable to serve eosq index.html", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("unable to read asset"))
		return
	}

	w.Header().Set("Content-Type", "text/html")
	io.Copy(w, reader)
}

func (s *Server) templatedIndex() (*bytes.Reader, error) {
	indexContent, err := s.box.Bytes("index.html")
	if err != nil {
		return nil, err
	}

	config := map[string]interface{}{
		"version":             1,
		"current_network":     "local",
		"on_demand":           false,
		"dfuse_io_endpoint":   "localhost" + s.config.DashboardHTTPListenAddr,
		"dfuse_io_api_key":    "web_0123456789abcdef",
		"dfuse_auth_endpoint": "http://localhost" + s.config.HttpListenAddr,
		"display_price":       false,
		"price_ticker_name":   "EOS",
		"available_networks":  []interface{}{},
		"secure":              false,
		"disable_segments":    true,
		"disable_sentry":      true,
	}

	tpl, err := template.New("index.html").Funcs(template.FuncMap{
		"json": func(v interface{}) (template.JS, error) {
			cnt, err := json.Marshal(v)
			return template.JS(cnt), err
		},
	}).Delims("--==", "==--").Parse(string(indexContent))
	if err != nil {
		return nil, err
	}

	buf := &bytes.Buffer{}
	if err := tpl.Execute(buf, config); err != nil {
		return nil, err
	}

	return bytes.NewReader(buf.Bytes()), nil
}
