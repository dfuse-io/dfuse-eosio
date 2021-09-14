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

package tokenmeta

import (
	"github.com/streamingfast/dmetrics"
)

var MetricsSet = dmetrics.NewSet()

var tokenContractCount = MetricsSet.NewGauge("token_contract_count")
var HeadBlockNum = MetricsSet.NewHeadBlockNumber("tokenmeta")
var HeadTimeDrift = MetricsSet.NewHeadTimeDrift("tokenmeta")
