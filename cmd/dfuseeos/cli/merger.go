package cli

import (
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/streamingfast/dlauncher/launcher"
	mergerApp "github.com/streamingfast/merger/app/merger"
)

func init() {
	launcher.RegisterApp(&launcher.AppDef{
		ID:          "merger",
		Title:       "Merger",
		Description: "Produces merged block files from single-block files",
		MetricsID:   "merger",
		Logger:      launcher.NewLoggingDef("github.com/streamingfast/merger.*", nil),
		RegisterFlags: func(cmd *cobra.Command) error {
			cmd.Flags().Duration("merger-time-between-store-lookups", 5*time.Second, "delay between source store polling (should be higher for remote storage)")
			cmd.Flags().String("merger-state-file", "{dfuse-data-dir}/merger/merger.seen.gob", "Path to file containing last written block number, as well as a map of all 'seen blocks' in the 'max-fixable-fork' range")
			cmd.Flags().Bool("merger-batch-mode", false, "Ignores the state file, starts and stop based on flags")
			cmd.Flags().Uint64("merger-start-block-num", 0, "[BATCH] Set the block number where we should start processing")
			cmd.Flags().Uint64("merger-stop-block-num", 0, "[BATCH] Set the block number where we should stop processing (and stop the process)")
			cmd.Flags().String("merger-grpc-listen-addr", MergerServingAddr, "Address to listen for incoming gRPC requests")
			cmd.Flags().Uint64("merger-minimal-block-num", 0, "Never process blocks lower than that number")
			cmd.Flags().Duration("merger-writers-leeway", 10*time.Second, "how long we wait after seeing the upper boundary, to ensure that we get as many blocks as possible in a bundle")
			cmd.Flags().Uint64("merger-max-fixable-fork", 10000, "Size of the 'seen blocks' cache. Anything one-block-file older than this will be deleted.")
			cmd.Flags().Int("merger-one-block-deletion-threads", 10, "number of parallel threads used to delete one-block-files (more means more stress on your storage backend)")
			cmd.Flags().Int("merger-max-one-block-operations-batch-size", 2000, "max number of 'good' (mergeable) files to look up from storage in one polling operation")

			return nil
		},
		// FIXME: Lots of config value construction is duplicated across InitFunc and FactoryFunc, how to streamline that
		//        and avoid the duplication? Note that this duplicate happens in many other apps, we might need to re-think our
		//        init flow and call init after the factory and giving it the instantiated app...
		InitFunc: func(runtime *launcher.Runtime) (err error) {
			dfuseDataDir := runtime.AbsDataDir

			if err = mkdirStorePathIfLocal(mustReplaceDataDir(dfuseDataDir, viper.GetString("common-blocks-store-url"))); err != nil {
				return
			}

			if err = mkdirStorePathIfLocal(mustReplaceDataDir(dfuseDataDir, viper.GetString("common-oneblock-store-url"))); err != nil {
				return
			}

			if err = mkdirStorePathIfLocal(mustReplaceDataDir(dfuseDataDir, viper.GetString("merger-state-file"))); err != nil {
				return
			}

			return nil
		},
		FactoryFunc: func(runtime *launcher.Runtime) (launcher.App, error) {
			dfuseDataDir := runtime.AbsDataDir
			return mergerApp.New(&mergerApp.Config{
				StorageMergedBlocksFilesPath: mustReplaceDataDir(dfuseDataDir, viper.GetString("common-blocks-store-url")),
				StorageOneBlockFilesPath:     mustReplaceDataDir(dfuseDataDir, viper.GetString("common-oneblock-store-url")),
				TimeBetweenStoreLookups:      viper.GetDuration("merger-time-between-store-lookups"),
				GRPCListenAddr:               viper.GetString("merger-grpc-listen-addr"),
				BatchMode:                    viper.GetBool("merger-batch-mode"),
				StartBlockNum:                viper.GetUint64("merger-start-block-num"),
				StopBlockNum:                 viper.GetUint64("merger-stop-block-num"),

				MinimalBlockNum:                viper.GetUint64("merger-minimal-block-num"),
				WritersLeewayDuration:          viper.GetDuration("merger-writers-leeway"),
				StateFile:                      mustReplaceDataDir(dfuseDataDir, viper.GetString("merger-state-file")),
				MaxFixableFork:                 viper.GetUint64("merger-max-fixable-fork"),
				MaxOneBlockOperationsBatchSize: viper.GetInt("merger-max-one-block-operations-batch-size"),
				OneBlockDeletionThreads:        viper.GetInt("merger-one-block-deletion-threads"),
			}), nil
		},
	})
}
