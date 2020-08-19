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

package abicodec

import (
	"time"

	"go.uber.org/zap"

	"github.com/dfuse-io/shutter"
)

type Backuper struct {
	*shutter.Shutter

	cache              Cache
	IsLive             bool
	exportABIsEnabled  bool
	exportABIsBaseURL  string
	exportABIsFilename string
}

func NewBackuper(cache Cache, exportABIsEnabled bool, exportABIsBaseURL, exportABIsFilename string) *Backuper {
	handler := &Backuper{
		Shutter:            shutter.New(),
		cache:              cache,
		exportABIsEnabled:  exportABIsEnabled,
		exportABIsBaseURL:  exportABIsBaseURL,
		exportABIsFilename: exportABIsFilename,
	}

	return handler
}

func (b *Backuper) BackupPeriodically(every time.Duration) {
	ticker := time.NewTicker(every)

	for {
		select {
		case <-b.Terminating():
			zlog.Info("terminating backup via shutter")
			return

		case <-ticker.C:
			err := b.cache.SaveState()
			if err != nil {
				zlog.Error("unable to backup abicodec", zap.Error(err))
			}

			if b.exportABIsEnabled && b.IsLive {
				err := b.cache.Export(b.exportABIsBaseURL, b.exportABIsFilename)
				if err != nil {
					zlog.Error("unable to export abis", zap.Error(err))
				}
			}
		}
	}
}
