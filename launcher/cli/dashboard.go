package cli

import (
	"github.com/dfuse-io/dfuse-box/dashboard"
	"github.com/dfuse-io/dfuse-box/launcher"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	launcher.RegisterApp(&launcher.AppDef{
		ID:          "dashboard",
		Description: "dfuse for ethereum - dashboard",
		MetricsID:   "dashboard",
		Logger:      launcher.NewLoggingDef("github.com/dfuse-io/dfuse-box/dashboard.*", nil),
		RegisterFlags: func(cmd *cobra.Command) error {
			cmd.Flags().String("dashboard-grpc-listen-addr", DashboardGrpcServingAddr, "TCP Listener addr for http")
			cmd.Flags().String("dashboard-http-listen-addr", DashboardHTTPListenAddr, "TCP Listener addr for gRPC")
			cmd.Flags().String("dashboard-eos-node-manager-api-addr", EosManagerAPIAddr, "Address of the superviser manager api")
			// FIXME: we can re-add when the app actually makes use of it.
			//cmd.Flags().String("dashboard-mindreader-manager-api-addr", MindreaderNodeosAPIAddr, "Address of the mindreader superviser manager api")
			return nil
		},
		FactoryFunc: func(modules *launcher.RuntimeModules) (launcher.App, error) {
			return dashboard.New(&dashboard.Config{
				GRPCListenAddr: viper.GetString("dashboard-grpc-listen-addr"),
				HTTPListenAddr: viper.GetString("dashboard-http-listen-addr"),
				//EosNodeManagerAPIAddr: viper.GetString("dashboard-eos-node-manager-api-addr"),
				//NodeosAPIHTTPServingAddr: viper.GetString("dashboard-mindreader-manager-api-addr"),
			}, &dashboard.Modules{
				Launcher:    modules.Launcher,
				DmeshClient: modules.SearchDmeshClient,
			}), nil
		},
	})
}
