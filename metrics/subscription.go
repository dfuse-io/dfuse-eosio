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

package metrics

import (
	"sync"

	pbdashboard "github.com/dfuse-io/dfuse-eosio/dashboard/pb"
	"go.uber.org/zap"
)

type subscription struct {
	AppFilter           string
	IncommingAppMetrics chan *pbdashboard.AppMetricsResponse
	Closed              bool
	QuitOnce            sync.Once
}

func newSubscription(chanSize int, appName string) (out *subscription) {
	return &subscription{
		IncommingAppMetrics: make(chan *pbdashboard.AppMetricsResponse, chanSize),
		AppFilter:           appName,
	}
}

func (s *subscription) Push(app *pbdashboard.AppMetricsResponse) {
	if s.Closed {
		return
	}

	zlog.Debug("pushing app metric response state to subscriber",
		zap.String("app_filter", s.AppFilter),
		zap.Reflect("response", app),
	)
	if len(s.IncommingAppMetrics) == cap(s.IncommingAppMetrics) {
		s.QuitOnce.Do(func() {
			zlog.Info("reach max buffer size for metric stream, closing channel",
				zap.String("app_filter", s.AppFilter),
			)
			close(s.IncommingAppMetrics)
			s.Closed = true
		})
		return
	}

	// Clean up
	s.IncommingAppMetrics <- app
}
