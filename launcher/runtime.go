package launcher

import (
	dmeshClient "github.com/dfuse-io/dmesh/client"
)

type RuntimeModules struct {
	SearchDmeshClient dmeshClient.SearchClient
	Launcher          *Launcher
}
