package tools

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/dfuse-io/bstream"
	"github.com/dfuse-io/dfuse-eosio/accounthist"
	"github.com/dfuse-io/dfuse-eosio/accounthist/injector"
	"github.com/dfuse-io/dfuse-eosio/accounthist/keyer"
	pbcodec "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/codec/v1"
	"github.com/dfuse-io/dfuse-eosio/statedb"
	"github.com/dfuse-io/dfuse-eosio/trxdb/kv"
	"github.com/dfuse-io/dstore"
	"github.com/dfuse-io/fluxdb"
	"github.com/dfuse-io/jsonpb"
	"github.com/dfuse-io/kvdb/store"
	"github.com/dustin/go-humanize"
	"github.com/eoscanada/eos-go"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

var errStopWalk = errors.New("stop walk")

var checkCmd = &cobra.Command{Use: "check", Short: "Various checks for deployment, data integrity & debugging"}

var checkAccounthistShardsCmd = &cobra.Command{
	Use:   "accounthist-shards {accounthist-mode} {dsn}",
	Short: "Checks to see if all Accounthist shard are contiguous",
	Args:  cobra.ExactArgs(2),
	RunE:  checkAccounthistShardE,
}
var checkMergedBlocksCmd = &cobra.Command{
	// TODO: Not sure, it's now a required thing, but we could probably use the same logic as `start`
	//       and avoid altogether passing the args. If this would also load the config and everything else,
	//       that would be much more seamless!
	Use:   "merged-blocks {store-url}",
	Short: "Checks for any holes in merged blocks as well as ensuring merged blocks integrity",
	Args:  cobra.ExactArgs(1),
	RunE:  checkMergedBlocksE,
}
var checkTrxdbBlocksCmd = &cobra.Command{
	Use:   "trxdb-blocks {database-dsn}",
	Short: "Checks for any holes in the trxdb database",
	Args:  cobra.ExactArgs(1),
	RunE:  checkTrxdbBlocksE,
}
var checkStateDBReprocSharderCmd = &cobra.Command{
	Use:   "statedb-reproc-sharder {store} {shard-count}",
	Short: "Checks to see if all StateDB reprocessing shards are present in the store",
	Args:  cobra.ExactArgs(2),
	RunE:  checkStateDBReprocSharderE,
}
var checkStateDBReprocInjectorCmd = &cobra.Command{
	Use:   "statedb-reproc-injector {dsn} {shard-count}",
	Short: "Checks to see if all StateDB reprocessing injector are aligned in database",
	Args:  cobra.ExactArgs(2),
	RunE:  checkStateDBReprocInjectorE,
}

func init() {
	Cmd.AddCommand(checkCmd)
	checkCmd.AddCommand(checkMergedBlocksCmd)
	checkCmd.AddCommand(checkTrxdbBlocksCmd)
	checkCmd.AddCommand(checkStateDBReprocSharderCmd)
	checkCmd.AddCommand(checkStateDBReprocInjectorCmd)
	checkCmd.AddCommand(checkAccounthistShardsCmd)

	checkCmd.PersistentFlags().StringP("range", "r", "", "Block range to use for the check, format is of the form '<start>:<stop>' (i.e. '-r 1000:2000')")

	checkMergedBlocksCmd.Flags().BoolP("print-stats", "s", false, "Natively decode each block in the segment and print statistics about it, ensuring it contains the required blocks")
	checkMergedBlocksCmd.Flags().BoolP("print-full", "f", false, "Natively decode each block and print the full JSON representation of the block, should be used with a small range only if you don't want to be overwhelmed")
}

func checkStateDBReprocInjectorE(cmd *cobra.Command, args []string) error {
	storeDSN := args[0]
	shards := args[1]
	shardsInt, err := strconv.ParseInt(shards, 10, 32)
	if err != nil {
		return fmt.Errorf("shards arg parsing: %w", err)
	}

	kvStore, err := fluxdb.NewKVStore(storeDSN)
	if err != nil {
		return fmt.Errorf("unable to create store: %w", err)
	}

	fdb := fluxdb.New(kvStore, nil, &statedb.BlockMapper{})
	fdb.SetSharding(0, int(shardsInt))
	_, lastBlock, err := fdb.VerifyAllShardsWritten(context.Background())
	if err != nil {
		return err
	}
	fmt.Println("Last block", lastBlock.String())
	return nil
}

type blockNum uint64

func (b blockNum) String() string {
	return "#" + strings.ReplaceAll(humanize.Comma(int64(b)), ",", " ")
}

func checkStateDBReprocSharderE(cmd *cobra.Command, args []string) error {
	storeURL := args[0]
	shards := args[1]
	shardCount, err := strconv.ParseUint(shards, 10, 64)
	if err != nil {
		return fmt.Errorf("shard count parsing: %w", err)
	}

	fmt.Printf("Checking statedb sharder holes on %s\n", storeURL)
	number := regexp.MustCompile(`(\d{3})/(\d{10})-(\d{10})`)

	shardsStore, err := dstore.NewStore(storeURL, "", "", false)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	seenShard := map[uint64]int{}
	expectedShard := uint64(0)
	expectedStart := uint64(0)
	lastPrintedValidStart := uint64(0)
	previousRange := BlockRange{0, 0}
	problemDetected := false

	err = shardsStore.Walk(ctx, "", ".tmp", func(filename string) error {
		match := number.FindStringSubmatch(filename)
		if match == nil {
			zlog.Debug("skipping file not matching regex", zap.String("file", filename))
			return nil
		}

		shardIndex, _ := strconv.ParseUint(match[1], 10, 32)
		startBlock, _ := strconv.ParseUint(match[2], 10, 64)
		stopBlock, _ := strconv.ParseUint(match[3], 10, 64)

		zlog.Debug("dealing with element",
			zap.Uint64("shard_index", shardIndex),
			zap.Stringer("expected_start", blockNum(expectedStart)),
			zap.Stringer("range", BlockRange{startBlock, stopBlock}),
			zap.String("file", filename),
		)

		if shardIndex != expectedShard {
			if len(seenShard) > 0 {
				fmt.Printf("✅ Range %s\n", BlockRange{lastPrintedValidStart, previousRange.Stop})
			}

			offsetToExpected := shardIndex - expectedShard

			// If we never seen any shard or the shard is not the direct next one, we are missing some shards
			if len(seenShard) <= 0 || offsetToExpected > 1 {
				if len(seenShard) > 0 {
					fmt.Println()
				}

				for i := expectedShard; i < shardIndex; i++ {
					fmt.Printf("❌ Shard #%03d! (Missing)\n", i)
				}
				problemDetected = true
			}

			expectedShard = shardIndex
			expectedStart = 0
			lastPrintedValidStart = 0

			fmt.Println()
			fmt.Printf("✅ Shard #%03d\n", shardIndex)
		} else if len(seenShard) == 0 && shardIndex == 0 {
			fmt.Printf("✅ Shard #%03d\n", shardIndex)
		}

		brokenRange := false
		if stopBlock <= startBlock {
			brokenRange = true
			problemDetected = true

			fmt.Printf("✅ Range %s\n", BlockRange{lastPrintedValidStart, expectedStart - 1})
			lastPrintedValidStart = previousRange.Stop + 1

			fmt.Printf("❌ Range %s! (Broken range, start block is greater or equal to stop block)\n", BlockRange{startBlock, stopBlock})
		} else if startBlock != expectedStart {
			problemDetected = true

			// This happens when current covers a subset of the last seen element (previous is `100 - 299` but we are `199 - 299`)
			if startBlock <= expectedStart && stopBlock < expectedStart {
				fmt.Printf("❌ Range %s! (Subset of previous range %s)\n", BlockRange{startBlock, stopBlock}, BlockRange{previousRange.Start, previousRange.Stop})
			} else {
				if lastPrintedValidStart != expectedStart {
					fmt.Printf("✅ Range %s\n", BlockRange{lastPrintedValidStart, expectedStart - 1})
					lastPrintedValidStart = stopBlock + 1
				} else {
					lastPrintedValidStart = startBlock
				}

				// This happens when current covers a superset of the last seen element (previous is `100 - 199` but we are `100 - 299`)
				if startBlock <= expectedStart {
					fmt.Printf("❌ Range %s! (Superset of previous range %s)\n", BlockRange{startBlock, stopBlock}, BlockRange{previousRange.Start, previousRange.Stop})
				} else {
					// Otherwise, we do not follow last seen element (previous is `100 - 199` but we are `299 - 300`)
					missingRange := BlockRange{expectedStart, startBlock - 1}
					fmt.Printf("❌ Range %s! (Missing, [%s])\n", missingRange, missingRange.ReprocRange())
				}
			}
		} else if startBlock-lastPrintedValidStart >= 15_000_000 {
			fmt.Printf("✅ Range %s\n", BlockRange{lastPrintedValidStart, stopBlock})
			lastPrintedValidStart = stopBlock + 1
		}

		seenShard[shardIndex] = seenShard[shardIndex] + 1
		if !brokenRange {
			previousRange = BlockRange{startBlock, stopBlock}
			expectedStart = stopBlock + 1
		}
		return nil
	})

	if len(seenShard) > 0 {
		fmt.Printf("✅ Range %s\n", BlockRange{lastPrintedValidStart, previousRange.Stop})
	}

	if uint64(len(seenShard)) != shardCount {
		for i := uint64(len(seenShard)); i < shardCount; i++ {
			fmt.Printf("❌ Shard #%03d is completely missing!\n", i)
		}
		problemDetected = true
	}

	fmt.Println("")
	if problemDetected {
		fmt.Printf("🆘 Problem(s) detected!\n")
	} else {
		fmt.Printf("🆗 All good, no problem detected\n")
	}

	return nil
}

func checkMergedBlocksE(cmd *cobra.Command, args []string) error {
	storeURL := args[0]
	fileBlockSize := uint32(100)

	fmt.Printf("Checking block holes on %s\n", storeURL)

	number := regexp.MustCompile(`(\d{10})`)

	var expected uint32
	var count int
	var baseNum32 uint32
	holeFound := false
	printIndividualSegmentStats := viper.GetBool("print-stats")
	printFullBlock := viper.GetBool("print-full")

	blockRange, err := getBlockRangeFromFlag()
	if err != nil {
		return err
	}

	expected = uint32(blockRange.Start)
	currentStartBlk := uint32(blockRange.Start)
	seenFilters := map[string]FilteringFilters{}

	blocksStore, err := dstore.NewDBinStore(storeURL)
	if err != nil {
		return err
	}

	ctx := context.Background()
	walkPrefix := walkBlockPrefix(blockRange, fileBlockSize)

	zlog.Debug("walking merged blocks", zap.Stringer("block_range", blockRange), zap.String("walk_prefix", walkPrefix))
	err = blocksStore.Walk(ctx, walkPrefix, ".tmp", func(filename string) error {
		match := number.FindStringSubmatch(filename)
		if match == nil {
			return nil
		}

		zlog.Debug("received merged blocks", zap.String("filename", filename))

		count++
		baseNum, _ := strconv.ParseUint(match[1], 10, 32)
		if baseNum+uint64(fileBlockSize) < blockRange.Start {
			zlog.Debug("base num lower then block range start, quitting")
			return nil
		}

		baseNum32 = uint32(baseNum)

		if printIndividualSegmentStats || printFullBlock {
			newSeenFilters := validateBlockSegment(blocksStore, filename, fileBlockSize, blockRange, printIndividualSegmentStats, printFullBlock)
			for key, filters := range newSeenFilters {
				seenFilters[key] = filters
			}
		}

		if baseNum32 != expected {
			// There is no previous valid block range if we are the ever first seen file
			if count > 1 {
				fmt.Printf("✅ Valid blocks range %d - %d\n", currentStartBlk, roundToBundleEndBlock(expected-fileBlockSize, fileBlockSize))
			}

			fmt.Printf("❌ Missing blocks range %d - %d!\n", expected, roundToBundleEndBlock(baseNum32-fileBlockSize, fileBlockSize))
			currentStartBlk = baseNum32

			holeFound = true
		}
		expected = baseNum32 + fileBlockSize

		if count%10000 == 0 {
			fmt.Printf("✅ Valid blocks range %d - %d\n", currentStartBlk, roundToBundleEndBlock(baseNum32, fileBlockSize))
			currentStartBlk = baseNum32 + fileBlockSize
		}

		if !blockRange.Unbounded() && roundToBundleEndBlock(baseNum32, fileBlockSize) >= uint32(blockRange.Stop-1) {
			return errStopWalk
		}

		return nil
	})
	if err != nil && err != errStopWalk {
		return err
	}

	actualEndBlock := roundToBundleEndBlock(baseNum32, fileBlockSize)
	if !blockRange.Unbounded() {
		actualEndBlock = uint32(blockRange.Stop)
	}

	fmt.Printf("✅ Valid blocks range %d - %d\n", currentStartBlk, actualEndBlock)

	if len(seenFilters) > 0 {
		fmt.Println()
		fmt.Println("Seen filters")
		for _, filters := range seenFilters {
			fmt.Printf("- [Include %q, Exclude %q, System %q]\n", filters.Include, filters.Exclude, filters.System)
		}
		fmt.Println()
	}

	if holeFound {
		fmt.Printf("🆘 Holes found!\n")
	} else {
		fmt.Printf("🆗 No hole found\n")
	}

	return nil
}

func walkBlockPrefix(blockRange BlockRange, fileBlockSize uint32) string {
	if blockRange.Unbounded() {
		return ""
	}

	startString := fmt.Sprintf("%010d", roundToBundleStartBlock(uint32(blockRange.Start), fileBlockSize))
	endString := fmt.Sprintf("%010d", roundToBundleEndBlock(uint32(blockRange.Stop-1), fileBlockSize)+1)

	offset := 0
	for i := 0; i < len(startString); i++ {
		if startString[i] != endString[i] {
			return string(startString[0:i])
		}

		offset++
	}

	// At this point, the two strings are equal, to return the string
	return startString
}

func roundToBundleStartBlock(block, fileBlockSize uint32) uint32 {
	// From a non-rounded block `1085` and size of `100`, we remove from it the value of
	// `modulo % fileblock` (`85`) making it flush (`1000`).
	return block - (block % fileBlockSize)
}

func roundToBundleEndBlock(block, fileBlockSize uint32) uint32 {
	// From a non-rounded block `1085` and size of `100`, we remove from it the value of
	// `modulo % fileblock` (`85`) making it flush (`1000`) than adding to it the last
	// merged block num value for this size which simply `size - 1` (`99`) giving us
	// a resolved formulae of `1085 - (1085 % 100) + (100 - 1) = 1085 - (85) + (99)`.
	return block - (block % fileBlockSize) + (fileBlockSize - 1)
}

func validateBlockSegment(
	store dstore.Store,
	segment string,
	fileBlockSize uint32,
	blockRange BlockRange,
	printIndividualSegmentStats bool,
	printFullBlock bool,
) (seenFilters map[string]FilteringFilters) {
	reader, err := store.OpenObject(context.Background(), segment)
	if err != nil {
		fmt.Printf("❌ Unable to read blocks segment %s: %s\n", segment, err)
		return
	}
	defer reader.Close()

	readerFactory, err := bstream.GetBlockReaderFactory.New(reader)
	if err != nil {
		fmt.Printf("❌ Unable to read blocks segment %s: %s\n", segment, err)
		return
	}

	// FIXME: Need to track block continuity (100, 101, 102a, 102b, 103, ...) and report which one are missing
	seenBlockCount := 0
	for {
		block, err := readerFactory.Read()
		if block != nil {
			if !blockRange.Unbounded() {
				if block.Number >= blockRange.Stop {
					return
				}

				if block.Number < blockRange.Start {
					continue
				}
			}

			seenBlockCount++

			if printIndividualSegmentStats {
				payloadSize := len(block.PayloadBuffer)
				eosBlock := block.ToNative().(*pbcodec.Block)

				if eosBlock.FilteringApplied {
					fmt.Printf("Filtered Block %s (%d bytes): %d/%d transactions (%d/%d traces), %d/%d actions (%d/%d input)\n",
						block,
						payloadSize,
						eosBlock.GetFilteredTransactionCount(),
						eosBlock.GetUnfilteredTransactionCount(),
						eosBlock.GetFilteredTransactionTraceCount(),
						eosBlock.GetUnfilteredTransactionTraceCount(),
						eosBlock.GetFilteredExecutedTotalActionCount(),
						eosBlock.GetUnfilteredExecutedTotalActionCount(),
						eosBlock.GetFilteredExecutedInputActionCount(),
						eosBlock.GetUnfilteredExecutedInputActionCount(),
					)
				} else {
					fmt.Printf("Block %s (%d bytes): %d transactions (%d traces), %d actions (%d input)\n",
						block,
						payloadSize,
						eosBlock.GetUnfilteredTransactionCount(),
						eosBlock.GetUnfilteredTransactionTraceCount(),
						eosBlock.GetUnfilteredExecutedTotalActionCount(),
						eosBlock.GetUnfilteredExecutedInputActionCount(),
					)
				}

				if seenFilters == nil {
					seenFilters = map[string]FilteringFilters{}
				}

				filters := FilteringFilters{
					eosBlock.FilteringIncludeFilterExpr,
					eosBlock.FilteringExcludeFilterExpr,
					eosBlock.FilteringSystemActionsIncludeFilterExpr,
				}
				seenFilters[filters.Key()] = filters
			}

			if printFullBlock {
				eosBlock := block.ToNative().(*pbcodec.Block)

				fmt.Printf(jsonpb.MarshalIndentToString(eosBlock, "  "))
			}

			continue
		}

		if block == nil && err == io.EOF {
			if seenBlockCount < expectedBlockCount(segment, fileBlockSize) {
				fmt.Printf("❌ Segment %s contained only %d blocks, expected at least 100\n", segment, seenBlockCount)
			}

			return
		}

		if err != nil {
			fmt.Printf("❌ Unable to read all blocks from segment %s after reading %d blocks: %s\n", segment, seenBlockCount, err)
			return
		}
	}
}

func expectedBlockCount(segment string, fileBlockSize uint32) int {
	// True only on EOSIO, on other chains, it's probably different from 1 to X
	if segment == "0000000000" {
		return int(fileBlockSize) - 2
	}

	return int(fileBlockSize)
}

func checkTrxdbBlocksE(cmd *cobra.Command, args []string) error {
	// FIXME: Seems `./dfuse-data/...` something doesn't work but `dfuse-data/...` works
	dsn := args[0]

	blockRange, err := getBlockRangeFromFlag()
	if err != nil {
		return err
	}

	startBlock := blockRange.Start
	endBlock := blockRange.Stop

	fmt.Printf("Checking block holes in trxdb at %s, from %d to %d\n", dsn, startBlock, endBlock)

	store, err := store.New(dsn)
	if err != nil {
		return err
	}

	ctx := context.Background()
	startKey := kv.Keys.PackIrrBlockNumPrefix(uint32(endBlock))
	endKey := kv.Keys.PackIrrBlockNumPrefix(uint32(startBlock))

	count := int64(0)
	started := false
	previousNum := uint64(0)
	holeFound := false
	t0 := time.Now()
	chunkSize := 100000

	for {
		it := store.Scan(ctx, startKey, endKey, chunkSize)
		localCount := 0
		for it.Next() {
			count++
			localCount++

			it := it.Item()

			blockID := kv.Keys.UnpackIrrBlocksKey(it.Key)
			blockNum := uint64(eos.BlockNum(blockID))

			if blockNum%100000 == 0 {
				fmt.Println("Reading irr block", blockNum)
			}

			if !started {
				fmt.Println("First block seen:", blockNum)
				previousNum = blockNum + 1
				started = true
			}

			difference := previousNum - blockNum

			if difference > 1 {
				fmt.Printf("❌ Missing blocks range %d - %d\n", blockNum+1, previousNum-1)
				holeFound = true
			}

			previousNum = blockNum

			startKey = append(it.Key[:], 0x00)
		}
		if it.Err() != nil {
			return fmt.Errorf("scanning table %s-%s: %w", hex.EncodeToString(startKey), hex.EncodeToString(endKey), it.Err())
		}

		if localCount != chunkSize {
			break
		}
	}

	delta := time.Since(t0)

	fmt.Printf("Scanned %d rows in %s\n", count, delta)

	if holeFound {
		fmt.Printf("🆘 Holes found!\n")
	} else {
		fmt.Printf("🆗 No hole found\n")
	}

	return nil
}

func checkAccounthistShardE(cmd *cobra.Command, args []string) error {
	storeURL := args[1]
	kvdb, err := store.New(storeURL)
	if err != nil {
		return fmt.Errorf("failed to setup db: %w", err)
	}
	kvdb = injector.NewRWCache(kvdb)
	mode := accounthist.AccounthistMode(args[0])
	service := setupService(kvdb, 0, mode)

	var prefix byte
	switch mode {
	case accounthist.AccounthistModeAccount:
		prefix = keyer.PrefixAccountCheckpoint
	case accounthist.AccounthistModeAccountContract:
		prefix = keyer.PrefixAccountContractCheckpoint
	default:
		return fmt.Errorf("invalid account hist more: %s", args[0])
	}

	out, err := service.ShardCheckpointAnalysis(cmd.Context(), prefix)
	if err != nil {
		return err
	}

	expectedShard := 0
	hasSeenFirstShard := false
	priorStartBlock := uint64(0)
	fmt.Printf("Account History Shard Summary:\n")
	if len(out) == 0 {
		fmt.Printf("No shards found\n")
	}
	for _, shard := range out {
		shardNum := int(shard.ShardNum)
		if expectedShard != shardNum {
			for i := 0; i < (shardNum - expectedShard); i++ {
				fmt.Printf("❌ expected shard-%d\n", (expectedShard + i))
			}
			expectedShard = shardNum
		}
		shardValid := true
		if hasSeenFirstShard {
			shardValid = (shard.Checkpoint.LastWrittenBlockNum == priorStartBlock-1)
		}

		if shardValid {
			fmt.Printf("✅ shard-%d %s\n", shardNum, BlockRange{shard.Checkpoint.InitialStartBlock, shard.Checkpoint.LastWrittenBlockNum})
		} else {
			fmt.Printf("❌ shard-%d %s (uncontiguous shard)\n", shardNum, BlockRange{shard.Checkpoint.InitialStartBlock, shard.Checkpoint.LastWrittenBlockNum})
		}
		expectedShard++
		priorStartBlock = shard.Checkpoint.InitialStartBlock
		hasSeenFirstShard = true

	}
	return nil
}
