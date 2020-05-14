package main

import (
	"io/ioutil"
	"net"

	stackdriverPropagation "contrib.go.opencensus.io/exporter/stackdriver/propagation"
	"github.com/dfuse-io/derr"
	"github.com/dfuse-io/dstore"
	"github.com/dfuse-io/fluxdb-client"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.opencensus.io/plugin/ochttp"
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
	abiCodecAddr := viper.GetString("serve-cmd-abi-codec-addr")
	abisBaseUrl := viper.GetString("serve-cmd-abis-base-url")
	abisFileName := viper.GetString("serve-cmd-abis-file-name")
	abisStoreFilepath := viper.GetString("serve-cmd-abis-store-filepath")
	abisList := viper.GetString("serve-cmd-abis-list")
	cacheFile := viper.GetString("serve-cmd-cache-file")
	blocksStoreURL := viper.GetString("serve-cmd-blocks-store")
	blockStreamAddr := viper.GetString("serve-cmd-block-stream-addr")
	blockmetaAddr := viper.GetString("serve-cmd-blockmeta-addr")
	saveEveryNBlock := viper.GetUint32("serve-cmd-save-every-n-block")
	bootstrapblockOffset := viper.GetUint64("serve-cmd-bootstrap-block-offset")

	zlog.Info("Starting tokenta",
		zap.String("listen_grpc_addr", grpcListenAddr),
		zap.String("fluxdb_addr", fluxdbAddr),
		zap.String("abi_codec_addr", abiCodecAddr),
		zap.String("abis_store_filepath", abisStoreFilepath),
		zap.String("abis_base_url", abisBaseUrl),
		zap.String("abis_file_name", abisFileName),
		zap.String("abis_list", abisList),
		zap.String("cache_file", cacheFile),
		zap.String("block_stream_addr", blockStreamAddr),
		zap.String("blocks_store", blocksStoreURL),
		zap.String("blockmeta_addr", blockmetaAddr),
		zap.Uint32("save_every_n_block", saveEveryNBlock),
		zap.Uint64("bootstrap_block_offset", bootstrapblockOffset))

	var tokenCache *cache.DefaultCache
	zlog.Info("setting up token cache")

	if cacheFile != "" {
		zlog.Info("trying to load from token cache file", zap.String("filename", cacheFile))
		tokenCache, err = cache.LoadDefaultCacheFromFile(cacheFile)
		if err != nil {
			zlog.Warn("cannot load from cache file", zap.Error(err))
		}
	}

	if tokenCache == nil {
		tokenCache, err = createTokenMetaCacheFromAbi(abisBaseUrl, abisFileName, cacheFile, fluxdbAddr, bootstrapblockOffset)
		if err != nil {
			return err
		}
	}

	zlog.Info("setting up blockstore")
	blocksStore, err := dstore.NewDBinStore(blocksStoreURL)
	derr.Check("failed setting up blocks store", err)

	zlog.Info("setting up start block")
	startBlock := tokenCache.AtBlockRef()
	zlog.Info("resolved start block", zap.Uint64("start_block_num", startBlock.Num()), zap.String("start_block_id", startBlock.ID()))

	zlog.Info("setting tokenmeta and pipeline")
	tmeta := tokenmeta.NewTokenMeta(tokenCache, abiCodecCli, saveEveryNBlock)
	tmeta.SetupPipeline(startBlock, blockStreamAddr, blocksStore)

	sigs := derr.SetupSignalHandler(viper.GetDuration("shutdown-drain-delay"))
	go func() {
		sig := <-sigs
		zlog.Info("terminating through system signal", zap.Reflect("sig", sig))
		tmeta.Shutdown(nil)
	}()

	zlog.Info("setting grpc server")
	server := tokenmeta.NewServer(tokenCache)
	listener, err := net.Listen("tcp", grpcListenAddr)
	if err != nil {
		zlog.Error("failed listening grpc", zap.Error(err))
		return err
	}

	go func() {
		zlog.Info("serving gRPC", zap.String("grpc_addr", grpcListenAddr))
		err = server.Serve(listener)
		if err != nil {
			zlog.Panic("unable to start gRPC server", zap.String("grpc_addr", grpcListenAddr), zap.Error(err))
		}
	}()

	return tmeta.Launch()
}

func createTokenMetaCacheFromAbi(abisBaseUrl, abisFileName, cacheFile, fluxdbAddr string, bootstrapblockOffset uint64) (*cache.DefaultCache, error) {
	zlog.Info("tokenmeta cache not present loading cached abis", zap.String("abis_base_url", abisBaseUrl), zap.String("abis_file_name", abisFileName))

	store, err := dstore.NewStore(abisBaseUrl, "", "zstd", true)
	if err != nil {
		zlog.Warn("cannot setup abis store", zap.Error(err))
		return nil, err
	}

	reader, err := store.OpenObject(abisFileName)
	if err != nil {
		zlog.Warn("cannot open abis cache file", zap.Error(err))
		return nil, err
	}
	defer reader.Close()

	cnt, err := ioutil.ReadAll(reader)
	if err != nil {
		zlog.Warn("cannot open abis cache file", zap.Error(err))
		return nil, err
	}

	zlog.Info("initialize flux db client")
	fluxClient := fluxdb.NewClient(fluxdbAddr, &ochttp.Transport{
		Propagation: &stackdriverPropagation.HTTPFormat{},
	})
	tokens, balances, stakedEntries, startBlock, err := tokenmeta.Bootstrap(cnt, fluxClient, bootstrapblockOffset)
	if err != nil {
		zlog.Warn("error bootstrap tokenmeta", zap.Error(err))
		return nil, err
	}

	zlog.Info("creating default cache from bootstrap", zap.Int("tokens_count", len(tokens)), zap.Int("balances_count", len(balances)), zap.Int("staked_entries_count", len(stakedEntries)))

	tokenCache := cache.NewDefaultCacheWithData(tokens, balances, stakedEntries, startBlock, cacheFile)

	err = tokenCache.SaveToFile()
	if err != nil {
		zlog.Error("cannot save token cache file", zap.Error(err), zap.String("filename", cacheFile))
	}
	return tokenCache, nil
}
