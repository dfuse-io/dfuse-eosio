package main

import (
	"github.com/dfuse-io/derr"
	"github.com/dfuse-io/dfuse-eosio/sqlsync"
	"github.com/dfuse-io/dstore"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

func init() {
	serveCmd.Flags().String("listen-grpc-addr", "localhost:9000", "Address to listen for incoming gRPC requests")
	serveCmd.Flags().String("fluxdb-addr", "http://localhost:9002", "fluxdb URL")

	serveCmd.Flags().String("block-stream-addr", "localhost:9001", "gRPC URL to reach a stream of blocks")
	serveCmd.Flags().String("blocks-store", "gs://dfuseio-global-blocks-us/eos-mainnet/v3", "GS path to read blocks archives")
}

func serveRunE(cmd *cobra.Command, args []string) (err error) {
	setup()

	grpcListenAddr := viper.GetString("serve-cmd-listen-grpc-addr")
	fluxdbAddr := viper.GetString("serve-cmd-fluxdb-addr")
	abisList := viper.GetString("serve-cmd-abis-list")
	blocksStoreURL := viper.GetString("serve-cmd-blocks-store")
	blockStreamAddr := viper.GetString("serve-cmd-block-stream-addr")
	blockmetaAddr := viper.GetString("serve-cmd-blockmeta-addr")

	zlog.Info("Starting tokenta",
		zap.String("listen_grpc_addr", grpcListenAddr),
		zap.String("fluxdb_addr", fluxdbAddr),
		zap.String("block_stream_addr", blockStreamAddr),
		zap.String("blocks_store", blocksStoreURL),
		zap.String("blockmeta_addr", blockmetaAddr))

	// TODO: fetch the ABI for `simpleassets`

	// Init the `db`
	// Create the tables if needed. Make sure the "marker" table exists too.
	// Check for _marker_ existence in database
	// Launch the process to fetch initial snapshot, if table doesn't include a marker
	// Start the pipeline to update the DB.

	zlog.Info("setting up blockstore")
	blocksStore, err := dstore.NewDBinStore(blocksStoreURL)
	derr.Check("failed setting up blocks store", err)

	zlog.Info("setting tokenmeta and pipeline")
	ss := sqlsync.NewSQLSync()
	ss.SetupPipeline(startBlock, blockStreamAddr, blocksStore)

	sigs := derr.SetupSignalHandler(viper.GetDuration("shutdown-drain-delay"))

	go ss.Launch()

	sig := <-sigs
	zlog.Info("terminating through system signal", zap.Reflect("sig", sig))
	tmeta.Shutdown(nil)

	return nil
}

// zlog.Info("initialize flux db client")
// fluxClient := fluxdb.NewClient(fluxdbAddr, &ochttp.Transport{
// 	Propagation: &stackdriverPropagation.HTTPFormat{},
// })
