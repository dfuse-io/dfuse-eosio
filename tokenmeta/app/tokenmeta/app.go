// Copyright 2020 dfuse Platform Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package tokenmeta

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/dfuse-io/bstream"
	"github.com/dfuse-io/dmetrics"

	"github.com/dfuse-io/derr"
	pbabicodec "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/abicodec/v1"
	pbstatedb "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/statedb/v1"
	"github.com/dfuse-io/dfuse-eosio/tokenmeta"
	"github.com/dfuse-io/dfuse-eosio/tokenmeta/cache"
	"github.com/dfuse-io/dgrpc"
	"github.com/dfuse-io/dstore"
	pbblockmeta "github.com/dfuse-io/pbgo/dfuse/blockmeta/v1"
	pbhealth "github.com/dfuse-io/pbgo/grpc/health/v1"
	"github.com/dfuse-io/shutter"
	"go.uber.org/zap"
)

type Config struct {
	GRPCListenAddr       string        // Address to listen for incoming gRPC requests
	StateDBGRPCAddr      string        // StateDB gRPC URL
	BlockStreamAddr      string        // gRPC URL to reach a stream of blocks
	ABICodecAddr         string        // Abi Codec URL
	ABICacheBaseURL      string        // cached ABIS base URL
	ABICacheFileName     string        // cached ABIS filename
	CacheFile            string        // Path to GOB file containing tokenmeta cache. will try to Load and Save to that cache file
	SaveEveryNBlock      uint32        // Save the cache after N blocks processed
	BlocksStoreURL       string        // GS path to read blocks archives
	BootstrapBlockOffset uint64        // Block offset to ensure that we are not bootstrapping from StateDB on a reversible fork
	ReadinessMaxLatency  time.Duration // we advertise as not-ready if the last processed block is older than this
}

type Modules struct {
	BlockFilter func(blk *bstream.Block) error
	BlockMeta   pbblockmeta.BlockIDClient
}

type App struct {
	*shutter.Shutter
	config  *Config
	modules *Modules

	readinessProbe pbhealth.HealthClient
}

func New(config *Config, modules *Modules) *App {
	return &App{
		Shutter: shutter.New(),
		config:  config,
		modules: modules,
	}
}

func (a *App) Run() error {
	zlog.Info("running tokenmeta", zap.Reflect("config", a.config))
	dmetrics.Register(tokenmeta.MetricsSet)
	var err error

	zlog.Info("initialize state db client")
	stateConn, err := dgrpc.NewInternalClient(a.config.StateDBGRPCAddr)
	if err != nil {
		return fmt.Errorf("cannot create statedb connection: %w", err)
	}

	stateClient := pbstatedb.NewStateClient(stateConn)

	var tokenCache *cache.DefaultCache
	zlog.Info("setting up token cache")
	if a.config.CacheFile != "" {
		mkdirCacheFileParents(a.config.CacheFile)

		zlog.Info("trying to load from token cache file", zap.String("filename", a.config.CacheFile))
		tokenCache, err = cache.LoadDefaultCacheFromFile(a.config.CacheFile)
		if err != nil && !isNotExits(err) {
			zlog.Warn("cannot load from cache file", zap.Error(err))
		}
	}

	if tokenCache == nil {
		zlog.Info("tokenmeta cache was not setup generating it from abicodec and statedb")
		tokenCache, err = a.createTokenMetaCacheFromAbi(stateClient)
		if err != nil {
			if err == TokenmetaAppGeneratCacheFromAbiAborted {
				return nil
			}
			return err
		}
	}

	zlog.Info("setting up blockstore")
	blocksStore, err := dstore.NewDBinStore(a.config.BlocksStoreURL)
	derr.Check("failed setting up blocks store", err)

	zlog.Info("setting up start block")
	startBlock := tokenCache.AtBlockRef()
	zlog.Info("resolved start block", zap.Uint64("start_block_num", startBlock.Num()), zap.String("start_block_id", startBlock.ID()))

	zlog.Info("setting up abi client")
	abiCodecConn, err := dgrpc.NewInternalClient(a.config.ABICodecAddr)
	derr.Check("failed getting abi codec grpc client", err)
	abiCodecCli := pbabicodec.NewDecoderClient(abiCodecConn)

	zlog.Info("setting tokenmeta and pipeline")
	tmeta := tokenmeta.NewTokenMeta(tokenCache, abiCodecCli, a.config.SaveEveryNBlock, stateClient, a.modules.BlockMeta)

	tmeta.OnTerminated(a.Shutdown)
	a.OnTerminating(tmeta.Shutdown)

	tmeta.SetupPipeline(startBlock, a.modules.BlockFilter, a.config.BlockStreamAddr, blocksStore)

	server := tokenmeta.NewServer(tokenCache, a.config.ReadinessMaxLatency)

	server.OnTerminated(a.Shutdown)
	a.OnTerminating(server.Shutdown)

	go server.Serve(a.config.GRPCListenAddr)

	gs, err := dgrpc.NewInternalClient(a.config.GRPCListenAddr)
	if err != nil {
		return fmt.Errorf("cannot create readiness probe: %w", err)
	}
	a.readinessProbe = pbhealth.NewHealthClient(gs)

	go tmeta.Launch()
	return nil
}

func (a *App) createTokenMetaCacheFromAbi(stateClient pbstatedb.StateClient) (*cache.DefaultCache, error) {
	zlog.Info("tokenmeta cache not present loading cached abis",
		zap.String("abis_base_url", a.config.ABICacheBaseURL),
		zap.String("abis_file_name", a.config.ABICacheFileName),
	)

	store, err := dstore.NewStore(a.config.ABICacheBaseURL, "", "zstd", true)
	if err != nil {
		zlog.Warn("cannot setup abis store", zap.Error(err))
		return nil, err
	}

	zlog.Info("tokenmeta retrieving cached abi files",
		zap.String("abi_cache_file", a.config.ABICacheFileName),
	)

	cnt, err := a.getAbiCacheFile(store, a.config.ABICacheFileName)
	if err != nil {
		return nil, err
	}

	tokens, balances, stakedEntries, startBlock, err := tokenmeta.Bootstrap(cnt, stateClient, a.config.BootstrapBlockOffset)
	if err != nil {
		zlog.Warn("error bootstrap tokenmeta", zap.Error(err))
		return nil, err
	}

	zlog.Info("creating default cache from bootstrap",
		zap.Int("tokens_count", len(tokens)),
		zap.Int("balances_count", len(balances)),
		zap.Int("staked_entries_count", len(stakedEntries)),
	)

	tokenCache := cache.NewDefaultCacheWithData(tokens, balances, stakedEntries, startBlock, a.config.CacheFile)

	err = tokenCache.SaveToFile()
	if err != nil {
		zlog.Error("cannot save token cache file", zap.Error(err), zap.String("filename", a.config.CacheFile))
	}
	return tokenCache, nil
}

var TokenmetaAppGeneratCacheFromAbiAborted = fmt.Errorf("getting abi cache file aborted by tokenmeta application")

func (a *App) getAbiCacheFile(store dstore.Store, abiCacheFilename string) ([]byte, error) {
	sleepTime := time.Duration(0)
	for {
		if a.IsTerminating() {
			zlog.Debug("leaving getAbiCacheFile because app is terminating")
			return nil, TokenmetaAppGeneratCacheFromAbiAborted
		}

		time.Sleep(sleepTime)
		sleepTime = time.Second * 2

		reader, err := store.OpenObject(context.Background(), abiCacheFilename)
		if err != nil {
			zlog.Info("abi cache file is not available, retrying...", zap.Error(err))
			continue
		}
		defer reader.Close()

		cnt, err := ioutil.ReadAll(reader)
		if err != nil {
			return nil, fmt.Errorf("read abi cache file: %w", err)
		}
		return cnt, nil
	}
}
func mkdirCacheFileParents(file string) error {
	dir := filepath.Dir(file)
	err := os.MkdirAll(dir, os.ModePerm)
	if err != nil {
		return fmt.Errorf("unablr to create parents %q for cache file %q: %w", dir, file, err)
	}

	return nil
}

func isNotExits(err error) bool {
	for {
		if os.IsNotExist(err) {
			return true
		}

		err = errors.Unwrap(err)
		if err == nil {
			return false
		}
	}
}
