package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/dfuse-io/dfuse-eosio/codec"
	"github.com/dfuse-io/dfuse-eosio/filtering"
	"github.com/dfuse-io/dfuse-eosio/launcher/cli"
	pbcodec "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/codec/v1"
	"github.com/dfuse-io/dstore"
	"github.com/dustin/go-humanize"
	"github.com/golang/protobuf/proto"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var filteringCmd = &cobra.Command{Use: "filtering", Short: "Filtering-related tools"}

var filteringEstimateCmd = &cobra.Command{Use: "estimate {merged-blocks-prefix}", Short: "Estimate the size of the different data stores for given filters. Ex: 'estimate 0125000' to test 1000 blocks around 125M.", RunE: filteringEstimateE, Args: cobra.MaximumNArgs(1)}

func init() {
	Cmd.AddCommand(filteringCmd)
	filteringCmd.AddCommand(filteringEstimateCmd)

	filteringEstimateCmd.Flags().String("blocks-store-url", cli.MergedBlocksStoreURL, "Store URL (with prefix) where to read merged blocks logs.")
	filteringEstimateCmd.Flags().String("filter-on-expr", "true", "CEL program to whitelist actions to index.")
	filteringEstimateCmd.Flags().String("filter-out-expr", "false", "CEL program to blacklist actions to index.")
	filteringEstimateCmd.Flags().String("indexed-terms", filtering.DefaultIndexedTerms, "Terms to index in Search")
	filteringEstimateCmd.Flags().Int64P("start-block", "p", 0, "Start block")
	filteringEstimateCmd.Flags().Int64P("end-block", "e", 4294967295, "End block")
}

func filteringEstimateE(cmd *cobra.Command, args []string) (err error) {
	var prefix string
	if len(args) != 0 {
		prefix = args[0]
	}

	startBlock := viper.GetInt64("start-block")
	endBlock := viper.GetInt64("end-block")
	storeURL := viper.GetString("blocks-store-url")

	fmt.Println("Setting up store", storeURL, "prefix:", prefix)
	blocksStore, err := dstore.NewDBinStore(storeURL)
	if err != nil {
		return fmt.Errorf("setting up source blocks store: %w", err)
	}

	filterOn := viper.GetString("filter-on-expr")
	filterOut := viper.GetString("filter-out-expr")
	mapper, err := filtering.NewBlockMapper("dfuseiohooks", true, filterOn, filterOut, viper.GetString("indexed-terms"))
	if err != nil {
		return fmt.Errorf("new block mapper: %w", err)
	}

	var totalBlocks int
	var totalMatchingSizeSearch int
	var totalMatchingSizeTrxdb int

	var totalTransactions int
	var totalActions int
	var totalMatchingTransactions int
	var totalMatchingActions int

	interruptSignal := errors.New("interrupt")

	ctx := context.Background()
	err = blocksStore.Walk(ctx, prefix, ".tmp", func(filename string) error {
		fmt.Println("Processing file", filename)
		readCloser, err := blocksStore.OpenObject(ctx, filename)
		if err != nil {
			return fmt.Errorf("open object: %w", err)
		}
		defer readCloser.Close()

		blkReader, err := codec.NewBlockReader(readCloser)
		if err != nil {
			return fmt.Errorf("new block reader factory: %w", err)
		}

	readfile:
		for {
			block, err := blkReader.Read()
			if err != nil && err != io.EOF {
				return fmt.Errorf("block reader failed: %s", err)
			}
			if err == io.EOF && (block == nil || block.Num() == 0) {
				break readfile
			}

			blockNum := int64(block.Num())
			if blockNum < startBlock {
				continue
			}
			if blockNum > endBlock {
				return interruptSignal
			}

			totalBlocks++

			blk := block.ToNative().(*pbcodec.Block)

			fmt.Println("Processing block", blk.Num())

			/// TrxDB
			matchingTrxs, actions, err := mapper.MapForDB(blk)
			if err != nil {
				return fmt.Errorf("map for db: %w", err)
			}

			// FIXME: Also go through ImplicitTransactions, Deferred, etc..
			for _, trx := range blk.TransactionTraces {
				totalActions += len(trx.ActionTraces)

				codec.DeduplicateTransactionTrace(trx)
				cnt, err := proto.Marshal(trx)
				if err != nil {
					return fmt.Errorf("proto marshal: %w", err)
				}
				codec.ReduplicateTransactionTrace(trx)

				// FIXME: make length compressed using Snappy or zstd.. or just estimate
				// with a fixed ratio.
				trxSize := len(cnt)
				totalTransactions++
				if matchingTrxs[trx.Id] {
					totalMatchingTransactions++
					totalMatchingSizeTrxdb += trxSize
				}
			}

			// More related to how Search consumes that data.

			totalMatchingActions += len(actions)
			for _, doc := range actions {
				// FIXME: Transform into indexed documents, Bleve style
				// before computing its size.
				cnt, err := json.Marshal(doc)
				if err != nil {
					return fmt.Errorf("json marshal: %w", err)
				}

				// FIXME: multiply it by something or compress it,
				// this is JSON serialized, in Bleve it looks very
				// different, so we need a proxy to that metric.
				docSize := len(cnt)

				totalMatchingSizeSearch += docSize
			}

		}

		return nil
	})
	if err != nil && err != interruptSignal {
		return fmt.Errorf("walking files: %w", err)
	}

	fmt.Println("Whitelist:")
	fmt.Println("  ", filterOn)
	fmt.Println("Blacklist:")
	fmt.Println("  ", filterOut)
	fmt.Println("Sample:")
	fmt.Println("  Blocks:", totalBlocks)
	fmt.Println("General matching stats:")
	fmt.Printf("* Matching transactions: %d / %d (%.2f%%)\n", totalMatchingTransactions, totalTransactions, float64(totalMatchingTransactions)/float64(totalTransactions)*100.0)
	fmt.Printf("* Matching actions: %d / %d (%.2f%%)\n", totalMatchingActions, totalActions, float64(totalMatchingActions)/float64(totalActions)*100.0)
	fmt.Println("Size estimates")
	fmt.Println("* Estimated size of filtered trxdb:", humanize.Bytes(uint64(totalMatchingSizeTrxdb)))
	fmt.Println("* Estimated size of filtered search indexes:", humanize.Bytes(uint64(totalMatchingSizeSearch)))
	return nil
}
