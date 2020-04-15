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

package eosdb

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
)

var stores = make(map[string]DriverFactory)

type DriverFactory func(dsn string, opts ...Option) (Driver, error)

// Register registers a storage backend Driver
func Register(schemeName string, factory DriverFactory) {
	schemeName = strings.ToLower(schemeName)

	if _, ok := stores[schemeName]; ok {
		panic(errors.Errorf("%s is already registered", schemeName))
	}

	stores[schemeName] = factory
}

// New initializes a new Driver
func New(dsn string, opts ...Option) (Driver, error) {
	parts := strings.Split(dsn, "://")
	if len(parts) < 2 {
		return nil, fmt.Errorf("missing :// in DSN")
	}

	factory := stores[parts[0]]
	if factory == nil {
		return nil, fmt.Errorf("dsn: unregistered driver for scheme %q, have you '_ import'ed the package?", parts[0])
	}

	return factory(dsn, opts...)
}
