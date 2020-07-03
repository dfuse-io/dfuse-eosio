package cli

import (
	"path/filepath"

	abicodecApp "github.com/dfuse-io/dfuse-eosio/abicodec/app/abicodec"
	"github.com/dfuse-io/dlauncher/launcher"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	// Abicodec
	launcher.RegisterApp(&launcher.AppDef{
		ID:          "abicodec",
		Title:       "ABI codec",
		Description: "Decodes binary data against ABIs for different contracts",
		MetricsID:   "abicodec",
		Logger:      launcher.NewLoggingDef("github.com/dfuse-io/dfuse-eosio/abicodec.*", nil),
		RegisterFlags: func(cmd *cobra.Command) error {
			cmd.Flags().String("abicodec-grpc-listen-addr", AbiServingAddr, "Address to listen for incoming gRPC requests")
			cmd.Flags().String("abicodec-cache-base-url", "{dfuse-data-dir}/storage/abicache", "path where the cache store is state")
			cmd.Flags().String("abicodec-cache-file-name", "abicodec_cache.bin", "path where the cache store is state")
			cmd.Flags().Bool("abicodec-export-cache", false, "Export cache and exit")
			cmd.Flags().String("abicodec-export-cache-url", "{dfuse-data-dir}/storage/abicache", "path where the exported cache will reside")
			return nil
		},
		FactoryFunc: func(modules *launcher.RuntimeModules) (launcher.App, error) {
			dfuseDataDir, err := dfuseAbsoluteDataDir()
			if err != nil {
				return nil, err
			}
			absDataDir, err := filepath.Abs(dfuseDataDir)
			if err != nil {
				return nil, err
			}

			return abicodecApp.New(&abicodecApp.Config{
				GRPCListenAddr: viper.GetString("abicodec-grpc-listen-addr"),
				SearchAddr:     viper.GetString("common-search-addr"),
				KvdbDSN:        mustReplaceDataDir(absDataDir, viper.GetString("common-trxdb-dsn")),
				CacheBaseURL:   mustReplaceDataDir(dfuseDataDir, viper.GetString("abicodec-cache-base-url")),
				CacheStateName: viper.GetString("abicodec-cache-file-name"),
				ExportCache:    viper.GetBool("abicodec-export-cache"),
				ExportCacheURL: viper.GetString("abicodec-export-cache-url"),
			}), nil
		},
	})
}
