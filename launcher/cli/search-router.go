package cli

import (
	"github.com/dfuse-io/dlauncher/launcher"
	routerApp "github.com/dfuse-io/search/app/router"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	// Search Router
	launcher.RegisterApp(&launcher.AppDef{
		ID:          "search-router",
		Title:       "Search router",
		Description: "Routes search queries to archiver, live",
		MetricsID:   "router",
		Logger:      launcher.NewLoggingDef("github.com/dfuse-io/search/(router|app/router).*", nil),
		RegisterFlags: func(cmd *cobra.Command) error {
			// Router-specific flags
			cmd.Flags().String("search-router-grpc-listen-addr", RouterServingAddr, "Address to listen for incoming gRPC requests")
			cmd.Flags().Bool("search-router-enable-retry", false, "Enables the router's attempt to retry a backend search if there is an error. This could have adverse consequences when search through the live")
			cmd.Flags().Uint64("search-router-head-delay-tolerance", 0, "Number of blocks above a backend's head we allow a request query to be served (Live & Router)")
			cmd.Flags().Uint64("search-router-lib-delay-tolerance", 0, "Number of blocks above a backend's lib we allow a request query to be served (Live & Router)")
			return nil
		},
		FactoryFunc: func(modules *launcher.RuntimeModules) (launcher.App, error) {
			return routerApp.New(&routerApp.Config{
				ServiceVersion:     viper.GetString("search-common-mesh-service-version"),
				BlockmetaAddr:      viper.GetString("common-blockmeta-addr"),
				GRPCListenAddr:     viper.GetString("search-router-grpc-listen-addr"),
				HeadDelayTolerance: viper.GetUint64("search-router-head-delay-tolerance"),
				LibDelayTolerance:  viper.GetUint64("search-router-lib-delay-tolerance"),
				EnableRetry:        viper.GetBool("search-router-enable-retry"),
			}, &routerApp.Modules{
				Dmesh: modules.SearchDmeshClient,
			}), nil
		},
	})
}
