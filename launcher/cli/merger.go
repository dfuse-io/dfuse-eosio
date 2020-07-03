package cli

import (
	"time"

	"github.com/dfuse-io/dlauncher/launcher"
	mergerApp "github.com/dfuse-io/merger/app/merger"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	launcher.RegisterApp(&launcher.AppDef{
		ID:          "merger",
		Title:       "Merger",
		Description: "Produces merged block files from single-block files",
		MetricsID:   "merger",
		Logger:      launcher.NewLoggingDef("github.com/dfuse-io/merger.*", nil),
		RegisterFlags: func(cmd *cobra.Command) error {
			cmd.Flags().Duration("merger-time-between-store-lookups", 10*time.Second, "delay between polling source store (higher for remote storage)")
			cmd.Flags().String("merger-grpc-listen-addr", MergerServingAddr, "Address to listen for incoming gRPC requests")
			cmd.Flags().Bool("merger-process-live-blocks", true, "Ignore --start-.. and --stop-.. blocks, and process only live blocks")
			cmd.Flags().Uint64("merger-start-block-num", 0, "FOR REPROCESSING: if >= 0, Set the block number where we should start processing")
			cmd.Flags().Uint64("merger-stop-block-num", 0, "FOR REPROCESSING: if > 0, Set the block number where we should stop processing (and stop the process)")
			cmd.Flags().String("merger-progress-filename", "", "FOR REPROCESSING: If non-empty, will update progress in this file and start right there on restart")
			cmd.Flags().Uint64("merger-minimal-block-num", 0, "FOR LIVE: Set the minimal block number where we should start looking at the destination storage to figure out where to start")
			cmd.Flags().Duration("merger-writers-leeway", 10*time.Second, "how long we wait after seeing the upper boundary, to ensure that we get as many blocks as possible in a bundle")
			cmd.Flags().String("merger-seen-blocks-file", "{dfuse-data-dir}/merger/merger.seen.gob", "file to save to / load from the map of 'seen blocks'")
			cmd.Flags().Uint64("merger-max-fixable-fork", 10000, "after that number of blocks, a block belonging to another fork will be discarded (DELETED depending on flagDeleteBlocksBefore) instead of being inserted in last bundle")
			cmd.Flags().Bool("merger-delete-blocks-before", true, "Enable deletion of oneblock files when prior to the currently processed bundle (to avoid long file listings)")
			return nil
		},
		// FIXME: Lots of config value construction is duplicated across InitFunc and FactoryFunc, how to streamline that
		//        and avoid the duplication? Note that this duplicate happens in many other apps, we might need to re-think our
		//        init flow and call init after the factory and giving it the instantiated app...
		InitFunc: func(modules *launcher.RuntimeModules) error {
			dfuseDataDir, err := dfuseAbsoluteDataDir()
			if err != nil {
				return err
			}
			err = mkdirStorePathIfLocal(mustReplaceDataDir(dfuseDataDir, viper.GetString("common-blocks-store-url")))
			if err != nil {
				return err
			}

			err = mkdirStorePathIfLocal(mustReplaceDataDir(dfuseDataDir, viper.GetString("common-oneblock-store-url")))
			if err != nil {
				return err
			}

			err = mkdirStorePathIfLocal(mustReplaceDataDir(dfuseDataDir, viper.GetString("merger-seen-blocks-file")))
			if err != nil {
				return err
			}

			return nil
		},
		FactoryFunc: func(modules *launcher.RuntimeModules) (launcher.App, error) {
			dfuseDataDir, err := dfuseAbsoluteDataDir()
			if err != nil {
				return nil, err
			}
			return mergerApp.New(&mergerApp.Config{
				StorageMergedBlocksFilesPath: mustReplaceDataDir(dfuseDataDir, viper.GetString("common-blocks-store-url")),
				StorageOneBlockFilesPath:     mustReplaceDataDir(dfuseDataDir, viper.GetString("common-oneblock-store-url")),
				TimeBetweenStoreLookups:      viper.GetDuration("merger-time-between-store-lookups"),
				GRPCListenAddr:               viper.GetString("merger-grpc-listen-addr"),
				Live:                         viper.GetBool("merger-process-live-blocks"),
				StartBlockNum:                viper.GetUint64("merger-start-block-num"),
				StopBlockNum:                 viper.GetUint64("merger-stop-block-num"),
				ProgressFilename:             viper.GetString("merger-progress-filename"),
				MinimalBlockNum:              viper.GetUint64("merger-minimal-block-num"),
				WritersLeewayDuration:        viper.GetDuration("merger-writers-leeway"),
				SeenBlocksFile:               mustReplaceDataDir(dfuseDataDir, viper.GetString("merger-seen-blocks-file")),
				MaxFixableFork:               viper.GetUint64("merger-max-fixable-fork"),
				DeleteBlocksBefore:           viper.GetBool("merger-delete-blocks-before"),
			}), nil
		},
	})
}
