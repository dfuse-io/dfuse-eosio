package cli

import (
	"fmt"
	"strconv"
	"strings"

	statedbApp "github.com/dfuse-io/dfuse-eosio/statedb/app/statedb"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/streamingfast/dlauncher/launcher"
	fluxdbApp "github.com/streamingfast/fluxdb/app/fluxdb"
)

func init() {
	launcher.RegisterApp(&launcher.AppDef{
		ID:          "statedb",
		Title:       "StateDB",
		Description: "Temporal chain state store",
		MetricsID:   "statedb",
		Logger:      launcher.NewLoggingDef("github.com/dfuse-io/(fluxdb.*|dfuse-eosio/statedb.*)", nil),
		RegisterFlags: func(cmd *cobra.Command) error {
			cmd.Flags().Bool("statedb-enable-server-mode", true, "Enables StateDB server mode, launched HTTP & gRPC servers, if set to `false`, the service will not accept HTTP nor gRPC requests")
			cmd.Flags().Bool("statedb-enable-inject-mode", true, "Enables StateDB inject mode, process new blocks writing state information into the database, if set to 'false', new state information will not be recorded!")
			cmd.Flags().Bool("statedb-enable-reproc-sharder-mode", false, "[BATCH] Enables StateDB reprocessing sharder mode, exclusive option, cannot be set if either server, injector or reproc-injector mode is set")
			cmd.Flags().Bool("statedb-enable-reproc-injector-mode", false, "[BATCH] Enables StateDB reprocessing injector mode, exclusive option, cannot be set if either server, injector or reproc-shard mode is set")
			cmd.Flags().String("statedb-store-dsn", StateDBDSN, "KV database connection string for State database")
			cmd.Flags().String("statedb-http-listen-addr", StateDBHTTPServingAddr, "Address to listen for incoming HTTP requests")
			cmd.Flags().String("statedb-grpc-listen-addr", StateDBGRPCServingAddr, "Address to listen for incoming gRPC requests")
			cmd.Flags().String("statedb-reproc-shard-store-url", "file://{dfuse-data-dir}/statedb/reproc-shards", "[BATCH] Storage url where all reproc shard write requests should be written to")
			cmd.Flags().String("statedb-reproc-shard-scratch-directory", "", "[BATCH] Provide a scratch directory where sharder while write each element composing a shard to a temporary file instead of holding everything in RAM, trade-off between I/O bound and RAM bound")
			cmd.Flags().Uint64("statedb-reproc-shard-count", 0, "[BATCH] Number of shards to split in (in 'reproc-sharder' mode), or join (in 'reproc-injector' mode)")
			cmd.Flags().Uint64("statedb-reproc-shard-start-block-num", 0, "[BATCH] Start processing blocks at this height, must be on a 100-blocks boundary")
			cmd.Flags().Uint64("statedb-reproc-shard-stop-block-num", 0, "[BATCH] Stop processing blocks at this height, must be on a 100-blocks boundary, inclusive value")
			cmd.Flags().Uint64("statedb-reproc-injector-shard-index", 0, "[BATCH] Index of the shard to perform injection for, should be lower than shard-count")
			cmd.Flags().Bool("statedb-disable-indexing", false, "[DEV] Do not perform any indexation of tablet when injecting data into storage engine, should never be used in production, present for repair jobs")
			cmd.Flags().Bool("statedb-disable-pipeline", false, "[DEV] Disables the blocks pipeline to keep up with live data (only set to true when testing locally)")
			cmd.Flags().Bool("statedb-disable-shard-reconciliation", false, "[DEV] Do not reconcile all shard last written block as the current active last written block, should never be used in production, present for repair jobs")
			cmd.Flags().String("statedb-ignore-index-range", "", "[DEV] When indexing a tablet, ignore any index found in the given range, should be used by developer only when rebuilding tablet indexes, format is <start>-<stop>")
			cmd.Flags().Bool("statedb-write-on-each-block", false, "[DEV] Forcefully flush block at each irreversible block step received, hinders write performance (only set to 'true' when testing locally)")
			return nil
		},
		FactoryFunc: func(runtime *launcher.Runtime) (launcher.App, error) {
			dfuseDataDir := runtime.AbsDataDir
			scratchDirectory := viper.GetString("statedb-reproc-shard-scratch-directory")
			if scratchDirectory != "" {
				scratchDirectory = mustReplaceDataDir(dfuseDataDir, scratchDirectory)
				scratchDirectory = strings.TrimPrefix(scratchDirectory, "file://")
			}

			fluxConfig := &fluxdbApp.Config{
				EnableServerMode:              viper.GetBool("statedb-enable-server-mode"),
				EnableInjectMode:              viper.GetBool("statedb-enable-inject-mode"),
				EnableReprocSharderMode:       viper.GetBool("statedb-enable-reproc-sharder-mode"),
				EnableReprocInjectorMode:      viper.GetBool("statedb-enable-reproc-injector-mode"),
				StoreDSN:                      mustReplaceDataDir(dfuseDataDir, viper.GetString("statedb-store-dsn")),
				BlockStreamAddr:               viper.GetString("common-blockstream-addr"),
				BlockStoreURL:                 mustReplaceDataDir(dfuseDataDir, viper.GetString("common-blocks-store-url")),
				ReprocShardStoreURL:           mustReplaceDataDir(dfuseDataDir, viper.GetString("statedb-reproc-shard-store-url")),
				ReprocShardCount:              viper.GetUint64("statedb-reproc-shard-count"),
				ReprocSharderScratchDirectory: scratchDirectory,
				ReprocSharderStartBlockNum:    viper.GetUint64("statedb-reproc-shard-start-block-num"),
				ReprocSharderStopBlockNum:     viper.GetUint64("statedb-reproc-shard-stop-block-num"),
				ReprocInjectorShardIndex:      viper.GetUint64("statedb-reproc-injector-shard-index"),
				DisableIndexing:               viper.GetBool("statedb-disable-indexing"),
				DisablePipeline:               viper.GetBool("statedb-disable-pipeline"),
				DisableShardReconciliation:    viper.GetBool("statedb-disable-shard-reconciliation"),
				WriteOnEachBlock:              viper.GetBool("statedb-write-on-each-block"),
			}

			ignoreIndexRange := viper.GetString("statedb-ignore-index-range")
			if ignoreIndexRange != "" {
				start, stop, err := parseRange(ignoreIndexRange)
				if err != nil {
					return nil, fmt.Errorf("invalid statedb-ignore-index-range value: %w", err)
				}

				fluxConfig.IgnoreIndexRangeStart = start
				fluxConfig.IgnoreIndexRangeStop = stop
			}

			return statedbApp.New(&statedbApp.Config{
				Config:         fluxConfig,
				HTTPListenAddr: viper.GetString("statedb-http-listen-addr"),
				GRPCListenAddr: viper.GetString("statedb-grpc-listen-addr"),
			}, &statedbApp.Modules{
				BlockFilter:        runtime.BlockFilter.TransformInPlace,
				BlockMeta:          runtime.BlockMeta,
				StartBlockResolver: runtime.Tracker.ResolveStartBlock,
			}), nil
		},
	})
}

func parseRange(value string) (start, stop uint64, err error) {
	parts := strings.Split(value, "-")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("invalid format, expected <start>-<stop> got %q", value)
	}

	start, err = strconv.ParseUint(parts[0], 10, 64)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid <start> value %q, not a valid number: %w", parts[0], err)
	}

	stop, err = strconv.ParseUint(parts[1], 10, 64)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid <stop> value %q, not a valid number: %w", parts[1], err)
	}

	if start == 0 || stop == 0 {
		return 0, 0, fmt.Errorf("invalid start/stop value, cannot be 0, got start %d and stop %d", start, stop)
	}

	if start >= stop {
		return 0, 0, fmt.Errorf("invalid range value, start %d is after or equal to stop %d", start, stop)
	}

	return start, stop, nil
}
