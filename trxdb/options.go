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

import "go.uber.org/zap"

type Option func(db DB) error

func WithLogger(logger *zap.Logger) Option {
	return func(db DB) error {
		if d, ok := db.(interface {
			SetLogger(*zap.Logger) error
		}); ok {
			return d.SetLogger(logger)
		}
		return nil
	}
}

func WithPurgeableStoreOption(ttl, purgeInterval uint64) Option {
	return func(db DB) error {
		if d, ok := db.(interface {
			SetPurgeableStore(ttl, purgeInterval uint64) error
		}); ok {
			return d.SetPurgeableStore(ttl, purgeInterval)
		}
		return nil
	}
}
