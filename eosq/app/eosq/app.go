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

package eosq

import (
	"github.com/dfuse-io/shutter"
)

type Config struct {
	HTTPListenAddr string
	Environment    string // i.e: production
	// EOSQ configruation payload
	APIEndpointURL    string // i.e: https://dev1.api.dfuse.dev
	ApiKey            string // i.e: web_XXXXXXXXXXXXXXXXXXXXXX
	AuthEndpointURL   string // i.e: https://auth.dfuse.io
	AvailableNetworks string // this is a JSON string ie: '[{"id": "eos-mainnet", "is_test": false, "logo": "/images/eos-mainnet.png", "name": "EOS Mainnet", "url": "https://eosq.app"}]'
	DisableAnalytics  bool   // Disables sentry and segment
	DefaultNetwork    string // The default network that is displayed, should correspond to an id in the avaiable networks
	DisplayPrice      bool   // Should eosq display prices
	PriceTickerName   string // the Price ticker name i.e. EOS
}

type App struct {
	*shutter.Shutter
	config *Config
	Ready  chan interface{}
	ready  bool
}

type Network struct {
	Id     string
	Name   string
	IsTest bool
	Logo   string
	URL    string
}

func New(config *Config) *App {
	return &App{
		Shutter: shutter.New(),
		config:  config,
		Ready:   make(chan interface{}),
	}
}

func (a *App) Run() error {

	zlog.Info("running eosq")
	s := newServer(a.config)
	a.OnTerminating(s.Shutdown)

	go func() {
		a.Shutdown(s.Launch())
	}()

	close(a.Ready)
	a.ready = true

	return nil
}

func (a *App) OnReady(f func()) {
	<-a.Ready
	f()
}

func (a *App) IsReady() bool {
	return a.ready
}
