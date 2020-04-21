package launcher

import (
	"github.com/dfuse-io/dfuse-eosio/metrics"
	dmeshClient "github.com/dfuse-io/dmesh/client"
)

type RuntimeModules struct {
	SearchDmeshClient dmeshClient.SearchClient
	MetricManager     *metrics.Manager
	Launcher          *Launcher
}
