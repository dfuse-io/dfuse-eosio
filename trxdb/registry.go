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

package trxdb

import (
	"fmt"
	"net/url"
	"reflect"
	"strings"

	"github.com/dfuse-io/kvdb/store"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

var stores = make(map[string]DriverFactory)

type DriverFactory func(dsn string, logger *zap.Logger) (Driver, error)

// Register registers a storage backend Driver
func Register(schemeName string, factory DriverFactory) {
	schemeName = strings.ToLower(schemeName)

	if _, ok := stores[schemeName]; ok {
		panic(errors.Errorf("%s is already registered", schemeName))
	}

	stores[schemeName] = factory
}

func IsRegistered(schemeName string) bool {
	_, isRegistered := stores[schemeName]
	return isRegistered
}

// New initializes a new Driver
func New(dsn string, opts ...Option) (Driver, error) {
	if strings.Contains(dsn, " ") {
		return NewSwitchDB(dsn, opts...)
	}

	return newFromDSN(dsn, opts)
}

func newFromDSN(dsn string, opts []Option) (Driver, error) {
	logger := getLogger(opts)

	logger.Debug("creating new trxdb instance", zap.String("dsn", dsn))
	parts := strings.Split(dsn, "://")
	if len(parts) < 2 {
		return nil, fmt.Errorf("missing :// in DSN")
	}

	factory := stores[parts[0]]
	if factory == nil {
		return nil, fmt.Errorf("dsn: unregistered driver for scheme %q, have you '_ import'ed the package?", parts[0])
	}

	storeDSN, dsnOptions, err := optionsFromDSN(dsn, logger)
	if err != nil {
		return nil, fmt.Errorf("invalid dsn options: %w", err)
	}

	logger.Debug("trxdb instance factory", zap.String("store_dsn", storeDSN))
	driver, err := factory(storeDSN, getLogger(opts))
	if err != nil {
		return nil, err
	}

	allOptions := append(dsnOptions, opts...)
	logger.Debug("configuring trxdb instance with options", zap.Stringer("type", reflect.TypeOf(driver)), zap.Int("option_count", len(allOptions)))
	for _, option := range allOptions {
		err := option.setOption(driver)
		if err != nil {
			return nil, fmt.Errorf("unable to set option %T: %w", option, err)
		}
	}

	return driver, err
}

func optionsFromDSN(dsn string, logger *zap.Logger) (storeDSN string, extraOpts []Option, err error) {
	logger.Debug("extracting options from dsn")
	writeOnly, err := getWriteOnlyOption(dsn, logger)
	if err != nil {
		return "", nil, fmt.Errorf("unable to get write only dsn option: %w", err)
	}

	logger.Debug("extracted write only option from dsn", zap.Strings("categories", writeOnly.Categories.AsHumanKeys()))
	extraOpts = []Option{writeOnly}
	storeDSN, err = store.RemoveDSNOptions(dsn, "write")
	if err != nil {
		return "", nil, fmt.Errorf("unable to remove write dsn option: %w", err)
	}

	return
}

func getWriteOnlyOption(dsn string, logger *zap.Logger) (out WriteOnlyOption, err error) {
	dsnURL, err := url.Parse(dsn)
	if err != nil {
		return out, err
	}

	query := dsnURL.Query()
	if len(query) <= 0 {
		logger.Debug(`dsn did not contain "write" option, assuming full indexing`)
		return WriteOnlyOption{FullIndexing}, nil
	}

	categories, err := NewIndexableCategories(query.Get("write"))
	if err != nil {
		return out, err
	}

	return WriteOnlyOption{categories}, nil
}

func getLogger(options []Option) (out *zap.Logger) {
	out = zlog
	for _, option := range options {
		if v, ok := option.(LoggerOption); ok {
			out = v.Logger
		}
	}
	return
}
