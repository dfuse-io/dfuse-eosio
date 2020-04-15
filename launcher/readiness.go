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

package launcher

import (
	"sync"

	pbdashboard "github.com/dfuse-io/dfuse-eosio/dashboard/pb"
	"go.uber.org/zap"
)

type subscription struct {
	IncomingAppInfo chan *pbdashboard.AppInfo
	Closed          bool
	QuitOnce        sync.Once
}

func newSubscription(chanSize int) (out *subscription) {
	return &subscription{
		IncomingAppInfo: make(chan *pbdashboard.AppInfo, chanSize),
	}
}

func (s *subscription) Push(app *pbdashboard.AppInfo) {
	if s.Closed {
		return
	}

	userLog.Debug("pushing app readiness state to subscriber",
		zap.Reflect("response", app),
	)
	if len(s.IncomingAppInfo) == cap(s.IncomingAppInfo) {
		s.QuitOnce.Do(func() {
			userLog.Debug("reach max buffer size for readiness stream, closing channel")
			close(s.IncomingAppInfo)
			s.Closed = true
		})
		return
	}

	// Clean up
	s.IncomingAppInfo <- app
}
