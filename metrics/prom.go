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
	"strconv"
	"strings"
	"time"

	pbdashboard "github.com/dfuse-io/dfuse-eosio/dashboard/pb"
	"github.com/golang/protobuf/ptypes"
	tspb "github.com/golang/protobuf/ptypes/timestamp"
	"github.com/tidwall/gjson"
	"go.uber.org/zap"
)

type promMetric struct {
	Timestamp time.Time
	Name      string      `json:"name,omitempty"`
	Help      string      `json:"help,omitempty"`
	Type      string      `json:"type,omitempty"`
	Metrics   interface{} `json:"metrics,omitempty"`
}

type metric struct {
	Labels map[string]string `json:"labels,omitempty"`
	Value  JSONFloat         `json:"value,omitempty"`
}

type JSONFloat float64

func (j *JSONFloat) UnmarshalJSON(data []byte) error {
	var str string
	err := json.Unmarshal(data, &str)
	if err != nil {
		return err
	}
	s, err := strconv.ParseFloat(str, 64)
	if err != nil {
		return err
	}
	*j = JSONFloat(s)
	return nil
}

func (p *promMetric) UnmarshalJSON(data []byte) error {
	p.Name = gjson.GetBytes(data, "name").String()
	p.Help = gjson.GetBytes(data, "help").String()
	p.Type = gjson.GetBytes(data, "type").String()
	p.Metrics = []interface{}{}
	metricsData := gjson.GetBytes(data, "metrics")
	if metricsData.Exists() {
		switch p.Type {
		case "COUNTER", "GAUGE":
			metrics := []*metric{}
			err := json.Unmarshal([]byte(metricsData.Raw), &metrics)
			if err != nil {
				return fmt.Errorf("unable to unmarshal prometheus metric %q with type %s", p.Name, p.Type)
			}
			p.Metrics = metrics
		default:
			zlog.Debug("unsupported metric type", zap.Reflect("metric", p))
		}
	}
	return nil
}

func (m *Manager) promMetricToAppResponse(p *promMetric) (out []*pbdashboard.AppMetricsResponse) {
	mtype := strings.ToUpper(p.Name)
	if metricType, ok := pbdashboard.MetricType_value[mtype]; ok {
		switch metrics := p.Metrics.(type) {
		case []*metric:
			for _, metric := range metrics {
				var metricID string
				if metricID, ok = metric.Labels["app"]; !ok {
					continue
				}

				var appMeta *AppMeta
				if appMeta, ok = m.metridIDToAppMeta[metricID]; !ok {
					zlog.Debug("app metric not link to any registered application", zap.String("app_metric_id", metricID))
					continue
				}

				out = append(out, &pbdashboard.AppMetricsResponse{
					Id:    appMeta.ID,
					Title: appMeta.Title,
					Metrics: []*pbdashboard.Metric{
						{
							Timestamp: timestampProto(p.Timestamp),
							Value:     float32(metric.Value),
							Type:      pbdashboard.MetricType(metricType),
						},
					},
					XXX_NoUnkeyedLiteral: struct{}{},
					XXX_unrecognized:     nil,
					XXX_sizecache:        0,
				})
			}
		default:
		}
	} else {
		//zlog.Error("unknown prometheus metric name", zap.String("metric_type", mtype))
	}
	return out
}

func timestampProto(t time.Time) *tspb.Timestamp {
	out, _ := ptypes.TimestampProto(t)
	return out
}
