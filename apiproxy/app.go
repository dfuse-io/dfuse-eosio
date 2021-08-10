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

package apiproxy

import (
	"fmt"

	"github.com/streamingfast/shutter"
	"github.com/streamingfast/dlauncher/launcher"
)

type Config struct {
	HTTPListenAddr   string
	HTTPSListenAddr  string
	AutocertDomains  []string
	DgraphqlHTTPAddr string
	EoswsHTTPAddr    string
	NodeosHTTPAddr   string
	RootHTTPAddr     string
	AutocertCacheDir string
}

type App struct {
	*shutter.Shutter
	config   *Config
	launcher *launcher.Launcher
}

func New(config *Config) *App {
	return &App{
		Shutter: shutter.New(),
		config:  config,
	}
}

func (a *App) Run() error {
	if a.config.HTTPSListenAddr != "" && len(a.config.AutocertDomains) == 0 {
		return fmt.Errorf("https listen address is set, but you did not specify autocert domains for SSL")
	}

	p := newProxy(a.config)

	a.OnTerminating(p.Shutdown)

	go func() {
		a.Shutdown(p.Launch())
	}()

	return nil
}
