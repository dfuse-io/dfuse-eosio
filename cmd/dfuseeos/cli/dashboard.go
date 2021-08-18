package cli

import (
	_ "github.com/dfuse-io/dfuse-eosio/dashboard"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/streamingfast/dlauncher/dashboard"
	"github.com/streamingfast/dlauncher/launcher"
)

func init() {
	launcher.RegisterApp(&launcher.AppDef{
		ID:          "dashboard",
		Title:       "Dashboard",
		Description: "dfuse for EOSIO - dashboard",
		MetricsID:   "dashboard",
		Logger:      launcher.NewLoggingDef("github.com/streamingfast/dlauncher/dashboard.*", nil),
		RegisterFlags: func(cmd *cobra.Command) error {
			cmd.Flags().String("dashboard-grpc-listen-addr", DashboardGRPCServingAddr, "TCP Listener addr for http")
			cmd.Flags().String("dashboard-http-listen-addr", DashboardHTTPListenAddr, "TCP Listener addr for gRPC")
			cmd.Flags().String("dashboard-metrics-api-addr", "http://127.0.0.1"+MetricsListenAddr, "HTTP address where to reach the metrics API endpoint")
			cmd.Flags().String("dashboard-eos-node-manager-api-addr", NodeManagerHTTPServingAddr, "Address of the superviser manager api")
			return nil
		},
		FactoryFunc: func(modules *launcher.Runtime) (launcher.App, error) {
			return dashboard.New(&dashboard.Config{
				HTTPListenAddr:      viper.GetString("dashboard-http-listen-addr"),
				GRPCListenAddr:      viper.GetString("dashboard-grpc-listen-addr"),
				NodeManagerAPIAddr:  viper.GetString("dashboard-eos-node-manager-api-addr"),
				MetricsHTTPAddr:     viper.GetString("dashboard-metrics-api-addr"),
				DmeshServiceVersion: viper.GetString("search-common-mesh-service-version"),
				Title:               "dfuse for EOSIO - dashboard",
				BlockExplorerName:   "eosq",
				HeadBlockNumberApp:  "mindreader",
			}, &dashboard.Modules{
				Launcher:    modules.Launcher,
				DmeshClient: modules.SearchDmeshClient,
			}), nil
		},
	})
}
