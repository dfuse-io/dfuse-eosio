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
			cmd.Flags().Bool("abicodec-export-abis-enabled", true, "Enable abis JSON export")
			cmd.Flags().String("abicodec-export-abis-base-url", "{dfuse-data-dir}/storage/abicache", "path where to put json.zstd abis export")
			cmd.Flags().String("abicodec-export-abis-file-name", "abi-cache.json.zst", "abi cache json filename")
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
				GRPCListenAddr:     viper.GetString("abicodec-grpc-listen-addr"),
				SearchAddr:         viper.GetString("common-search-addr"),
				KvdbDSN:            mustReplaceDataDir(absDataDir, viper.GetString("common-trxdb-dsn")),
				CacheBaseURL:       mustReplaceDataDir(dfuseDataDir, viper.GetString("abicodec-cache-base-url")),
				CacheStateName:     viper.GetString("abicodec-cache-file-name"),
				ExportABIsEnabled:  viper.GetBool("abicodec-export-abis-enabled"),
				ExportABIsBaseURL:  mustReplaceDataDir(dfuseDataDir, viper.GetString("abicodec-export-abis-base-url")),
				ExportABIsFilename: viper.GetString("abicodec-export-abis-file-name"),
			}), nil
		},
	})
}
