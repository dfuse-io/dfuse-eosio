// Copyright 2020 dfuse Platform Inc.
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
	"github.com/dfuse-io/dmetrics"
)

var Metricset = dmetrics.NewSet()

var CurrentConnections = Metricset.NewGauge("eosws_current_connections", "Number of connections open on web socket")
var ConnectionsCounter = Metricset.NewCounter("ws_connections_counter", "Counter of connections on web socket")
var DocumentResponseCounter = Metricset.NewCounter("document_response_counter", "Number of documents sent as response in websocket")
var CurrentListeners = Metricset.NewGaugeVec("current_listeners", []string{"req_type"}, "Number of WS streams active (listening)")
var ListenersCount = Metricset.NewGaugeVec("listeners_count", []string{"req_type"}, "Counter of WS streams requests")
var PushTrxCount = Metricset.NewCounterVec("push_transaction_count", []string{"guarantee"}, "Number of request for push_transaction")
var TimedOutPushTrxCount = Metricset.NewCounterVec("timed_out_pushing_transaction_count", []string{"guarantee"}, "Number of requests for push_transaction timed out waiting for inclusion in block")
var TimedOutPushingTrxCount = Metricset.NewCounterVec("timed_out_push_transaction_count", []string{"guarantee"}, "Number of requests for push_transaction timed out while submitting")
var FailedPushTrxCount = Metricset.NewCounterVec("failed_push_transaction_count", []string{"guarantee"}, "Number of failed requests for push_transaction before being submitted")
var SucceededPushTrxCount = Metricset.NewCounterVec("succeeded_push_transaction_count", []string{"guarantee"}, "Number of succeeded requests for push_transaction")
var HeadBlockNum = Metricset.NewHeadBlockNumber("eosws")
var HeadTimeDrift = Metricset.NewHeadTimeDrift("eosws")

func IncCurrentConnections() {
	CurrentConnections.Inc()
	ConnectionsCounter.Inc()
}

func IncListeners(reqType string) {
	CurrentListeners.Inc(reqType)
	ListenersCount.Inc(reqType)
}
