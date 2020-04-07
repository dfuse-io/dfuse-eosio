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
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	pbdashboard "github.com/dfuse-io/dfuse-eosio/dashboard/pb"
	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/prom2json"
	"go.uber.org/zap"
)

const DEFAULT_METRICS_POLLING time.Duration = 5 * time.Second
const DEFAULT_MF_CHAN_SIZE int = 1024

type AppMeta struct {
	Title string
	Id    string
}
type Manager struct {
	metricUrl         string
	polling           time.Duration
	metricNameFilter  []string
	metridIDToAppMeta map[string]*AppMeta

	metricSubscription     map[string][]*subscription
	metricSubscriptionLock sync.RWMutex
}

func NewManager(metricUrl string, metricTypeFilter []string, polling time.Duration, metricIDMap map[string]*AppMeta) *Manager {
	return &Manager{
		metricUrl:          metricUrl,
		metricNameFilter:   metricTypeFilter,
		polling:            polling,
		metricSubscription: make(map[string][]*subscription),
		metridIDToAppMeta:  metricIDMap,
	}
}

func (m *Manager) Subscribe(appID string) *subscription {
	if appID == "" {
		appID = "*"
	}

	chanSize := 500
	sub := newSubscription(chanSize, appID)

	m.metricSubscriptionLock.Lock()
	defer m.metricSubscriptionLock.Unlock()
	if _, ok := m.metricSubscription[appID]; ok {
		m.metricSubscription[appID] = append(m.metricSubscription[appID], sub)
	} else {
		m.metricSubscription[appID] = []*subscription{sub}
	}

	zlog.Debug("metric streaming subscribed",
		zap.Int("new_length", len(m.metricSubscription[appID])),
		zap.String("app_id", appID),
	)
	return sub
}

func (m *Manager) Unsubscribe(appID string, sub *subscription) {
	if sub == nil {
		return
	}

	if appID == "" {
		appID = "*"
	}

	m.metricSubscriptionLock.Lock()
	defer m.metricSubscriptionLock.Unlock()

	subscriptions, found := m.metricSubscription[appID]
	if !found {
		return
	}

	var filtered []*subscription
	for _, candidate := range subscriptions {
		// Pointer address comparison
		if candidate != sub {
			filtered = append(filtered, candidate)
		}
	}

	if len(filtered) <= 0 {
		delete(m.metricSubscription, appID)
	} else {
		m.metricSubscription[appID] = filtered
	}
}

func (m *Manager) getPolling() time.Duration {
	if m.polling == 0 {
		return DEFAULT_METRICS_POLLING
	}
	return m.polling
}

func (m *Manager) Launch() *Manager {
	for {
		timestamp, data, err := m.consumeMetricStream()
		if err != nil {
			zlog.Error("unable to consumer metric stream", zap.Error(err))
			continue
		}

		if len(data) == 0 {
			zlog.Debug("no metrics to parse")
			continue
		}

		metrics, err := m.filterMetrics(timestamp, data)
		if err != nil {
			zlog.Error("unable to consumer metric stream", zap.Error(err))
			continue
		}

		apps := m.promMetricToAppMetricResponses(metrics)
		if err != nil {
			zlog.Error("unable to generate app metrics stream", zap.Error(err))
			continue
		}

		m.streamMetrics(apps)
		time.Sleep(m.getPolling())
	}
}

func (m *Manager) consumeMetricStream() (time.Time, []byte, error) {
	zlog.Debug("consuming metric stream")
	mfChan := make(chan *dto.MetricFamily, DEFAULT_MF_CHAN_SIZE)

	response, err := http.Get(m.metricUrl)
	if err != nil {
		return time.Now(), nil, fmt.Errorf("unable to get http response: %w", err)
	}
	defer response.Body.Close()
	timestamp := time.Now()

	go func() {
		zlog.Debug("launching prom to json parser")
		if err := prom2json.ParseResponse(response, mfChan); err != nil {
			log.Fatal("error reading metrics:", err)
		}
	}()

	result := []*prom2json.Family{}
	for mf := range mfChan {
		result = append(result, prom2json.NewFamily(mf))
	}

	jsonText, err := json.Marshal(result)
	if err != nil {
		return time.Now(), nil, fmt.Errorf("unable to marshal metrics to JSON: %w", err)
	}

	return timestamp, jsonText, nil
}

func (m *Manager) filterMetrics(timestamp time.Time, rawMetric []byte) ([]*promMetric, error) {
	metrics := []*promMetric{}
	err := json.Unmarshal(rawMetric, &metrics)
	if err != nil {
		return nil, fmt.Errorf("unable to unmarshal raw metrics into array of structured metrics")
	}

	zlog.Debug("unmarshal raw metrics into structured metrics")
	out := []*promMetric{}
	for _, metric := range metrics {
		if m.isMetricWhitelisted(metric.Name) {
			metric.Timestamp = timestamp
			out = append(out, metric)
		}
	}

	zlog.Debug("filtered metrics", zap.Reflect("metrics", out))
	return out, nil
}

func (m *Manager) promMetricToAppMetricResponses(metrics []*promMetric) (out []*pbdashboard.AppMetricsResponse) {
	appMetrics := map[string]*pbdashboard.AppMetricsResponse{}
	for _, metric := range metrics {
		apps := m.promMetricToAppResponse(metric)
		for _, app := range apps {
			if _, ok := appMetrics[app.Id]; ok {
				appMetrics[app.Id].Metrics = append(appMetrics[app.Id].Metrics, app.Metrics...)
			} else {
				appMetrics[app.Id] = app
			}
		}
	}

	for _, v := range appMetrics {
		out = append(out, v)
	}
	return out
}

func (m *Manager) streamMetrics(appsMetrics []*pbdashboard.AppMetricsResponse) {
	m.metricSubscriptionLock.RLock()
	defer m.metricSubscriptionLock.RUnlock()

	for _, appMetrics := range appsMetrics {
		zlog.Debug("streaming app",
			zap.String("app_id", appMetrics.Id),
			zap.Reflect("metrics", appMetrics.Metrics),
		)

		if subs, found := m.metricSubscription[appMetrics.Id]; found {
			for _, sub := range subs {
				sub.Push(appMetrics)
			}
		}

		if subs, found := m.metricSubscription["*"]; found {
			for _, sub := range subs {
				sub.Push(appMetrics)
			}
		}
	}
}

func (m *Manager) isMetricWhitelisted(metricName string) bool {
	for _, mname := range m.metricNameFilter {
		if mname == metricName {
			return true
		}
	}
	return false
}
