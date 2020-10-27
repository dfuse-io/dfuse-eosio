package cli

import (
	tokenmetaApp "github.com/dfuse-io/dfuse-eosio/tokenmeta/app/tokenmeta"
	"github.com/dfuse-io/dlauncher/launcher"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	launcher.RegisterApp(&launcher.AppDef{
		ID:          "tokenmeta",
		Title:       "Tokenmeta",
		Description: "Serves token contracts information on a given network",
		MetricsID:   "tokenmeta",
		Logger:      launcher.NewLoggingDef("github.com/dfuse-io/dfuse-eosio/tokenmeta.*", nil),
		RegisterFlags: func(cmd *cobra.Command) error {
			cmd.Flags().String("tokenmeta-grpc-listen-addr", ":14001", "Address to listen for incoming gRPC requests")
			cmd.Flags().String("tokenmeta-statedb-grpc-addr", StateDBGRPCServingAddr, "StateDB GRPC address")
			cmd.Flags().String("tokenmeta-abi-codec-addr", AbiServingAddr, "ABI Codec URL")
			cmd.Flags().String("tokenmeta-abis-base-url", "{dfuse-data-dir}/storage/abicache", "cached ABIS base URL")
			cmd.Flags().String("tokenmeta-abis-file-name", "abi-cache.json.zst", "cached ABIS filename")
			cmd.Flags().String("tokenmeta-cache-file", "{dfuse-data-dir}/tokenmeta/token-cache.gob", "Path to GOB file containing tokenmeta cache. will try to Load and Save to that cache file")
			cmd.Flags().Uint32("tokenmeta-save-every-n-block", 900, "Save the cache after N blocks processed")
			cmd.Flags().Uint64("tokenmeta-bootstrap-block-offset", 20, "Block offset to ensure that we are not bootstrapping from statedb on a reversible fork")
			return nil
		},
		FactoryFunc: func(runtime *launcher.Runtime) (app launcher.App, e error) {
			dfuseDataDir := runtime.AbsDataDir

			return tokenmetaApp.New(&tokenmetaApp.Config{
				GRPCListenAddr:       viper.GetString("tokenmeta-grpc-listen-addr"),
				StateDBGRPCAddr:      viper.GetString("tokenmeta-statedb-grpc-addr"),
				BlockStreamAddr:      viper.GetString("common-blockstream-addr"),
				ABICodecAddr:         viper.GetString("tokenmeta-abi-codec-addr"),
				ABICacheBaseURL:      mustReplaceDataDir(dfuseDataDir, viper.GetString("tokenmeta-abis-base-url")),
				ABICacheFileName:     viper.GetString("tokenmeta-abis-file-name"),
				CacheFile:            mustReplaceDataDir(dfuseDataDir, viper.GetString("tokenmeta-cache-file")),
				SaveEveryNBlock:      viper.GetUint32("tokenmeta-save-every-n-block"),
				BlocksStoreURL:       mustReplaceDataDir(dfuseDataDir, viper.GetString("common-blocks-store-url")),
				BootstrapBlockOffset: viper.GetUint64("tokenmeta-bootstrap-block-offset"),
			}, &tokenmetaApp.Modules{
				BlockFilter: runtime.BlockFilter.TransformInPlace,
			}), nil
		},
	})
}
