package tokenmeta

import (
	"github.com/dfuse-io/dmetrics"
)

var MetricsSet = dmetrics.NewSet()

var tokenContractCount = MetricsSet.NewGauge("token_contract_count")
