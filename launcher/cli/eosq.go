package cli

import (
	eosqApp "github.com/dfuse-io/dfuse-eosio/eosq/app/eosq"
	"github.com/dfuse-io/dlauncher/launcher"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	// EOSQ
	launcher.RegisterApp(&launcher.AppDef{
		ID:          "eosq",
		Title:       "Eosq",
		Description: "EOSIO Block Explorer",
		MetricsID:   "eosq",
		Logger:      launcher.NewLoggingDef("github.com/dfuse-io/dfuse-eosio/eosq.*", nil),
		InitFunc:    nil,
		RegisterFlags: func(cmd *cobra.Command) error {
			cmd.Flags().String("eosq-http-listen-addr", EosqHTTPServingAddr, "Auth URL used to configure the dfuse js client")
			cmd.Flags().String("eosq-api-endpoint-url", APIProxyHTTPListenAddr, "API key used in eosq")
			cmd.Flags().String("eosq-auth-url", JWTIssuerURL, "Auth URL used to configure the dfuse js client")
			cmd.Flags().String("eosq-api-key", EosqAPIKey, "API key used in eosq")
			cmd.Flags().String("eosq-environment", "dev", "Environment where eosq will run (dev, dev, production)")
			cmd.Flags().String("eosq-available-networks", "", "json string to configure the networks section of eosq.")
			cmd.Flags().String("eosq-default-network", "local", "Default network that is displayed. It should correspond to an `id` in the available networks")
			cmd.Flags().Bool("eosq-disable-analytics", true, "Disables sentry and segment")
			cmd.Flags().Bool("eosq-display-price", false, "Should display prices via our price API")
			cmd.Flags().String("eosq-price-ticker-name", "EOS", "The price ticker")
			cmd.Flags().Bool("eosq-on-demand", false, "Is eosq deployed for an on-demand network")
			cmd.Flags().Bool("eosq-disable-tokenmeta", true, "Disables tokenmeta calls from eosq")
			return nil
		},

		FactoryFunc: func(modules *launcher.RuntimeModules) (launcher.App, error) {
			return eosqApp.New(&eosqApp.Config{
				HTTPListenAddr:    viper.GetString("eosq-http-listen-addr"),
				Environement:      viper.GetString("eosq-environment"),
				APIEndpointURL:    viper.GetString("eosq-api-endpoint-url"),
				ApiKey:            viper.GetString("eosq-api-key"),
				AuthEndpointURL:   viper.GetString("eosq-auth-url"),
				AvailableNetworks: viper.GetString("eosq-available-networks"),
				DisableAnalytics:  viper.GetBool("eosq-disable-analytics"),
				DefaultNetwork:    viper.GetString("eosq-default-network"),
				DisplayPrice:      viper.GetBool("eosq-display-price"),
				PriceTickerName:   viper.GetString("eosq-price-ticker-name"),
				OnDemand:          viper.GetBool("eosq-on-demand"),
				DisableTokenmeta:  viper.GetBool("eosq-disable-tokenmeta"),
			}), nil
		},
	})
}
