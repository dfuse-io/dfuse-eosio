package firehose

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/dfuse-io/bstream"
	"github.com/dfuse-io/bstream/blockstream"
	blockstreamv2 "github.com/dfuse-io/bstream/blockstream/v2"
	"github.com/dfuse-io/bstream/hub"
	dauthAuthenticator "github.com/dfuse-io/dauth/authenticator"
	"github.com/dfuse-io/dfuse-eosio/filtering"
	pbcodec "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/codec/v1"
	"github.com/dfuse-io/dgraphql/metrics"
	"github.com/dfuse-io/dgrpc"
	"github.com/dfuse-io/dmetering"
	"github.com/dfuse-io/dmetrics"
	"github.com/dfuse-io/dstore"
	pbbstream "github.com/dfuse-io/pbgo/dfuse/bstream/v1"
	"github.com/dfuse-io/shutter"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

type Config struct {
	AuthPlugin              string
	BlocksStoreURLs         []string
	GRPCListenAddr          string
	MeteringPlugin          string
	UpstreamBlockStreamAddr string
}

type Modules struct {
	Authenticator dauthAuthenticator.Authenticator
	Meterering    dmetering.Metering
	Tracker       *bstream.Tracker
}

type App struct {
	*shutter.Shutter
	config    *Config
	modules   *Modules
	ReadyFunc func()
	isReady   func() bool
}

func New(config *Config, modules *Modules) *App {
	return &App{
		Shutter:   shutter.New(),
		config:    config,
		modules:   modules,
		ReadyFunc: func() {},
	}
}

func (a *App) Run() error {
	appCtx, cancel := context.WithCancel(context.Background())
	a.Shutter.OnTerminating(func(_ error) {
		cancel()
	})

	dmetrics.Register(metrics.MetricSet)

	zlog.Info("running firehose", zap.Reflect("config", a.config))
	if len(a.config.BlocksStoreURLs) == 0 {
		return fmt.Errorf("invalid config: no block store urls set up")
	}

	blockStores := make([]dstore.Store, len(a.config.BlocksStoreURLs))
	for i, url := range a.config.BlocksStoreURLs {
		store, err := dstore.NewDBinStore(url)
		if err != nil {
			return fmt.Errorf("failed setting up block store from url %q: %w", url, err)
		}

		blockStores[i] = store
	}

	withLive := a.config.UpstreamBlockStreamAddr != ""

	var subscriptionHub *hub.SubscriptionHub
	var serverLiveHeadTracker bstream.BlockRefGetter
	var serverLiveSourceFactory bstream.SourceFactory

	if withLive {
		var err error
		subscriptionHub, err = a.newSubscriptionHub(appCtx, blockStores)
		if err != nil {
			return fmt.Errorf("setting up subscription hub: %w", err)
		}

		serverLiveHeadTracker = subscriptionHub.HeadTracker
		serverLiveSourceFactory = bstream.SourceFactory(func(h bstream.Handler) bstream.Source {
			return subscriptionHub.NewSource(h, 250)
		})
	}

	zlog.Info("setting up blockstream V2 server", zap.Bool("live_support", withLive))
	blockStreamService := blockstreamv2.NewServer(
		zlog,
		blockStores,
		serverLiveSourceFactory,
		serverLiveHeadTracker,
		a.modules.Tracker,
		blockstreamv2.BlockTrimmerFunc(trimBlock),
	)

	blockStreamService.SetPreprocFactory(func(req *pbbstream.BlocksRequestV2) (bstream.PreprocessFunc, error) {
		filter, err := filtering.NewBlockFilter([]string{req.IncludeFilterExpr}, []string{req.ExcludeFilterExpr}, nil)
		if err != nil {
			return nil, fmt.Errorf("parsing filter: %w", err)
		}
		preproc := &filtering.FilteringPreprocessor{Filter: filter}
		return preproc.PreprocessBlock, nil
	})

	blockStreamService.SetPostHook(func(ctx context.Context, response *pbbstream.BlockResponseV2) {
		//////////////////////////////////////////////////////////////////////
		dmetering.EmitWithContext(dmetering.Event{
			Source:      "firehose",
			Kind:        "gRPC Stream",
			Method:      "Blocks",
			EgressBytes: int64(response.XXX_Size()),
		}, ctx)
		//////////////////////////////////////////////////////////////////////
	})

	a.isReady = blockStreamService.IsReady

	srv := a.newGRPCServer(blockStreamService.IsReady)

	zlog.Info("registering grpc services")
	srv.RegisterService(func(gs *grpc.Server) {
		pbbstream.RegisterBlockStreamV2Server(gs, blockStreamService)
	})

	a.OnTerminating(func(_ error) { srv.Shutdown(30 * time.Second) })
	srv.OnTerminated(a.Shutdown)

	listenAddr := strings.ReplaceAll(a.config.GRPCListenAddr, "*", "")
	zlog.Info("launching grpc server", zap.String("listening_addr", listenAddr))
	go srv.Launch(listenAddr)

	if withLive {
		go subscriptionHub.Launch()
	}

	go func() {
		if withLive {
			subscriptionHub.WaitUntilRealTime()
		}

		zlog.Info("firehose is now ready")
		blockStreamService.SetReady()
		a.ReadyFunc()
	}()

	return nil
}

func (a *App) newSubscriptionHub(ctx context.Context, blockStores []dstore.Store) (*hub.SubscriptionHub, error) {
	var start uint64
	zlog.Info("retrieving live start block")
	for retries := 0; ; retries++ {
		lib, err := a.modules.Tracker.Get(ctx, bstream.BlockStreamLIBTarget)
		if err != nil {
			if retries%5 == 4 {
				zlog.Warn("cannot get lib num from blockstream, retrying", zap.Int("retries", retries), zap.Error(err))
			}
			time.Sleep(time.Second)
			continue
		}
		start = lib.Num()
		break
	}

	liveSourceFactory := bstream.SourceFromNumFactory(func(startBlockNum uint64, h bstream.Handler) bstream.Source {
		return blockstream.NewSource(
			context.Background(),
			a.config.UpstreamBlockStreamAddr,
			100,
			bstream.HandlerFunc(func(blk *bstream.Block, obj interface{}) error {
				metrics.HeadTimeDrift.SetBlockTime(blk.Time())
				return h.ProcessBlock(blk, obj)
			}),
			blockstream.WithRequester("firehose"),
		)
	})

	fileSourceFactory := bstream.SourceFromNumFactory(func(startBlockNum uint64, h bstream.Handler) bstream.Source {
		var options []bstream.FileSourceOption
		if len(blockStores) > 1 {
			options = append(options, bstream.FileSourceWithSecondaryBlocksStores(blockStores[1:]))
		}

		zlog.Info("creating file source", zap.String("block_store", blockStores[0].ObjectPath("")), zap.Uint64("start_block_num", startBlockNum))
		src := bstream.NewFileSource(blockStores[0], startBlockNum, 1, nil, h, options...)
		return src
	})

	zlog.Info("setting up subscription hub")
	buffer := bstream.NewBuffer("hub-buffer", zlog.Named("hub"))
	tailManager := bstream.NewSimpleTailManager(buffer, 350)
	go tailManager.Launch()

	return hub.NewSubscriptionHub(
		start,
		buffer,
		tailManager.TailLock,
		fileSourceFactory,
		liveSourceFactory,
		hub.Withlogger(zlog),
		hub.WithRealtimeTolerance(1*time.Minute),
		hub.WithoutMemoization(), // This should be tweakable on the Hub, by the bstreamv2.Server
	)
}

func (a *App) newGRPCServer(checkIsReady func() bool) *dgrpc.Server {
	options := []dgrpc.ServerOption{
		dgrpc.WithLogger(zlog),
		dgrpc.WithHealthCheck(dgrpc.HealthCheckOverGRPC|dgrpc.HealthCheckOverHTTP, func(ctx context.Context) (isReady bool, out interface{}, err error) {
			return checkIsReady(), nil, nil
		}),
	}

	if strings.Contains(a.config.GRPCListenAddr, "*") {
		options = append(options, dgrpc.InsecureServer())
	} else {
		options = append(options, dgrpc.PlainTextServer())
	}

	if a.modules.Authenticator.IsAuthenticationTokenRequired() {
		options = append(options, dgrpc.WithAuthChecker(a.modules.Authenticator.Check))
	}

	return dgrpc.NewServer2(options...)
}

func trimBlock(blk interface{}, details pbbstream.BlockDetails) interface{} {
	if details == pbbstream.BlockDetails_BLOCK_DETAILS_FULL {
		return blk
	}

	// We need to create a new instance because this block could be in the live segment
	// which is shared across all streams that requires live block. As such, we cannot modify
	// them in-place, so we require to create a new instance.
	//
	// The copy is mostly shallow since we copy over pointers element but some part are deep
	// copied like ActionTrace which requires trimming.
	fullBlock := blk.(*pbcodec.Block)
	block := &pbcodec.Block{
		Id:                       fullBlock.Id,
		Number:                   fullBlock.Number,
		DposIrreversibleBlocknum: fullBlock.DposIrreversibleBlocknum,
		Header: &pbcodec.BlockHeader{
			Timestamp: fullBlock.Header.Timestamp,
			Producer:  fullBlock.Header.Producer,
		},
	}

	var newTrace func(fullTrxTrace *pbcodec.TransactionTrace) (trxTrace *pbcodec.TransactionTrace)
	newTrace = func(fullTrxTrace *pbcodec.TransactionTrace) (trxTrace *pbcodec.TransactionTrace) {
		trxTrace = &pbcodec.TransactionTrace{
			Id:        fullTrxTrace.Id,
			Receipt:   fullTrxTrace.Receipt,
			Scheduled: fullTrxTrace.Scheduled,
			Exception: fullTrxTrace.Exception,
		}

		if fullTrxTrace.FailedDtrxTrace != nil {
			trxTrace.FailedDtrxTrace = newTrace(fullTrxTrace.FailedDtrxTrace)
		}

		trxTrace.ActionTraces = make([]*pbcodec.ActionTrace, len(fullTrxTrace.ActionTraces))
		for i, fullActTrace := range fullTrxTrace.ActionTraces {
			actTrace := &pbcodec.ActionTrace{
				Receiver:                               fullActTrace.Receiver,
				ContextFree:                            fullActTrace.ContextFree,
				Exception:                              fullActTrace.Exception,
				ErrorCode:                              fullActTrace.ErrorCode,
				ActionOrdinal:                          fullActTrace.ActionOrdinal,
				CreatorActionOrdinal:                   fullActTrace.CreatorActionOrdinal,
				ClosestUnnotifiedAncestorActionOrdinal: fullActTrace.ClosestUnnotifiedAncestorActionOrdinal,
				ExecutionIndex:                         fullActTrace.ExecutionIndex,
			}

			if fullActTrace.Action != nil {
				actTrace.Action = &pbcodec.Action{
					Account:       fullActTrace.Action.Account,
					Name:          fullActTrace.Action.Name,
					Authorization: fullActTrace.Action.Authorization,
					JsonData:      fullActTrace.Action.JsonData,
				}

				if fullActTrace.Action.JsonData == "" {
					actTrace.Action.RawData = fullActTrace.Action.RawData
				}
			}

			if fullActTrace.Receipt != nil {
				actTrace.Receipt = &pbcodec.ActionReceipt{
					GlobalSequence: fullActTrace.Receipt.GlobalSequence,
				}
			}

			trxTrace.ActionTraces[i] = actTrace
		}

		return trxTrace
	}

	traces := make([]*pbcodec.TransactionTrace, len(fullBlock.TransactionTraces()))
	for i, fullTrxTrace := range fullBlock.TransactionTraces() {
		traces[i] = newTrace(fullTrxTrace)
	}

	if fullBlock.FilteringApplied {
		block.FilteredTransactionTraces = traces
	} else {
		block.UnfilteredTransactionTraces = traces
	}

	return block
}
