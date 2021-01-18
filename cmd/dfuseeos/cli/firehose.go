package cli

import (
	"fmt"
	"strings"
	"time"

	"github.com/dfuse-io/bstream"
	blockstreamv2 "github.com/dfuse-io/bstream/blockstream/v2"
	dauthAuthenticator "github.com/dfuse-io/dauth/authenticator"
	"github.com/dfuse-io/dfuse-eosio/filtering"
	pbcodec "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/codec/v1"
	"github.com/dfuse-io/dlauncher/launcher"
	"github.com/dfuse-io/dmetering"
	"github.com/dfuse-io/dmetrics"
	firehoseApp "github.com/dfuse-io/firehose/app/firehose"
	"github.com/dfuse-io/logging"
	pbbstream "github.com/dfuse-io/pbgo/dfuse/bstream/v1"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

var metricset = dmetrics.NewSet()
var headBlockNumMetric = metricset.NewHeadBlockNumber("firehose")
var headTimeDriftmetric = metricset.NewHeadTimeDrift("firehose")

func init() {
	appLogger := zap.NewNop()
	logging.Register("github.com/dfuse-io/dfuse-eosio/firehose", &appLogger)

	launcher.RegisterApp(&launcher.AppDef{
		ID:          "firehose",
		Title:       "Block Firehose",
		Description: "Provides on-demand filtered blocks, depends on common-blocks-store-url and common-blockstream-addr",
		MetricsID:   "merged-filter",
		Logger:      launcher.NewLoggingDef("github.com/dfuse-io/dfuse-eosio/firehose.*", nil),
		RegisterFlags: func(cmd *cobra.Command) error {
			cmd.Flags().String("firehose-grpc-listen-addr", FirehoseGRPCServingAddr, "Address on which the firehose will listen")
			cmd.Flags().StringSlice("firehose-blocks-store-urls", nil, "If non-empty, overrides common-blocks-store-url with a list of blocks stores")
			return nil
		},

		FactoryFunc: func(runtime *launcher.Runtime) (launcher.App, error) {
			dfuseDataDir := runtime.AbsDataDir
			tracker := runtime.Tracker.Clone()
			blockstreamAddr := viper.GetString("common-blockstream-addr")
			if blockstreamAddr != "" {
				tracker.AddGetter(bstream.BlockStreamLIBTarget, bstream.StreamLIBBlockRefGetter(blockstreamAddr))
			}

			// FIXME: That should be a shared dependencies across `dfuse for EOSIO`
			authenticator, err := dauthAuthenticator.New(viper.GetString("common-auth-plugin"))
			if err != nil {
				return nil, fmt.Errorf("unable to initialize dauth: %w", err)
			}

			// FIXME: That should be a shared dependencies across `dfuse for EOSIO`, it will avoid the need to call `dmetering.SetDefaultMeter`
			metering, err := dmetering.New(viper.GetString("common-metering-plugin"))
			if err != nil {
				return nil, fmt.Errorf("unable to initialize dmetering: %w", err)
			}
			dmetering.SetDefaultMeter(metering)

			firehoseBlocksStoreURLs := viper.GetStringSlice("firehose-blocks-store-urls")
			if len(firehoseBlocksStoreURLs) == 0 {
				firehoseBlocksStoreURLs = []string{viper.GetString("common-blocks-store-url")}
			} else if len(firehoseBlocksStoreURLs) == 1 && strings.Contains(firehoseBlocksStoreURLs[0], ",") {
				// Providing multiple elements from config doesn't work with `viper.GetStringSlice`, so let's also handle the case where a single element has separator
				firehoseBlocksStoreURLs = strings.Split(firehoseBlocksStoreURLs[0], ",")
			}

			for _, url := range firehoseBlocksStoreURLs {
				url = mustReplaceDataDir(dfuseDataDir, url)
			}

			shutdownSignalDelay := viper.GetDuration("common-system-shutdown-signal-delay")
			grcpShutdownGracePeriod := time.Duration(0)
			if shutdownSignalDelay.Seconds() > 5 {
				grcpShutdownGracePeriod = shutdownSignalDelay - (5 * time.Second)
			}

			filterPreprocessorFactory := func(includeExpr, excludeExpr string) (bstream.PreprocessFunc, error) {
				filter, err := filtering.NewBlockFilter([]string{includeExpr}, []string{excludeExpr}, nil)
				if err != nil {
					return nil, fmt.Errorf("parsing filter expressions: %w", err)
				}

				preproc := &filtering.FilteringPreprocessor{Filter: filter}
				return preproc.PreprocessBlock, nil
			}

			return firehoseApp.New(appLogger, &firehoseApp.Config{
				BlockStoreURLs:          firehoseBlocksStoreURLs,
				BlockStreamAddr:         blockstreamAddr,
				GRPCListenAddr:          viper.GetString("firehose-grpc-listen-addr"),
				GRPCShutdownGracePeriod: grcpShutdownGracePeriod,
			}, &firehoseApp.Modules{
				Authenticator:             authenticator,
				BlockTrimmer:              blockstreamv2.BlockTrimmerFunc(trimBlock),
				FilterPreprocessorFactory: filterPreprocessorFactory,
				HeadTimeDriftMetric:       headTimeDriftmetric,
				HeadBlockNumberMetric:     headBlockNumMetric,
				Tracker:                   tracker,
			}), nil
		},
	})
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
