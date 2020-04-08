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

package dashboard

import (
	"github.com/dfuse-io/dfuse-eosio/launcher"
	"github.com/dfuse-io/dfuse-eosio/metrics"
	dmeshCli "github.com/dfuse-io/dmesh/client"
	"github.com/dfuse-io/shutter"
)

type Config struct {
	ManagerCommandURL        string
	GRPCListenAddr           string
	HTTPListenAddr           string
	DgraphqlHTTPServingAddr  string
	EoswsHTTPServingAddr     string
	NodeosAPIHTTPServingAddr string
	Launcher                 *launcher.Launcher
	MetricManager            *metrics.Manager
	DmeshClient              dmeshCli.SearchClient
}

type App struct {
	*shutter.Shutter
	config   *Config
	launcher *launcher.Launcher
	Ready    chan interface{}
	ready    bool
}

func New(config *Config) *App {
	return &App{
		Shutter: shutter.New(),
		config:  config,
		Ready:   make(chan interface{}),
	}
}

func (a *App) Run() error {
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
