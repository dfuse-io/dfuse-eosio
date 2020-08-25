package tools

import (
	"context"
	"encoding/hex"
	"fmt"
	"io"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/dfuse-io/bstream"
	pbcodec "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/codec/v1"
	"github.com/dfuse-io/dfuse-eosio/statedb"
	"github.com/dfuse-io/dfuse-eosio/trxdb/kv"
	"github.com/dfuse-io/dstore"
	"github.com/dfuse-io/fluxdb"
	"github.com/dfuse-io/kvdb/store"
	"github.com/dustin/go-humanize"
	"github.com/eoscanada/eos-go"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

var checkCmd = &cobra.Command{Use: "check", Short: "Various checks for deployment, data integrity & debugging"}
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

	checkMergedBlocksCmd.Flags().Bool("individual-segment", false, "Open each merged blocks segment and ensure it contains all blocks it should")
	checkMergedBlocksCmd.Flags().Bool("print-stats", false, "Natively decode each block in the segment and print statistics about it")

	checkCmd.PersistentFlags().Uint64P("start-block", "s", 0, "Block number to start at")
	checkCmd.PersistentFlags().Uint64P("end-block", "e", 4294967296, "Block number to end at")
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

type blockRange struct {
	start uint64
	stop  uint64
}

func (b blockRange) ReprocRange() string {
	return fmt.Sprintf("%d:%d", b.start, b.stop+1)
}

func (b blockRange) String() string {
	return fmt.Sprintf("%s - %s", blockNum(b.start), blockNum(b.stop))
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
	previousRange := blockRange{0, 0}
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
			zap.Stringer("range", blockRange{startBlock, stopBlock}),
			zap.String("file", filename),
		)

		if shardIndex != expectedShard {
			if len(seenShard) > 0 {
				fmt.Printf("✅ Range %s\n", blockRange{lastPrintedValidStart, previousRange.stop})
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

		if startBlock != expectedStart {
			// This happens when current covers a subset of the last seen element (previous is `100 - 299` but we are `199 - 299`)
			if startBlock <= expectedStart && stopBlock < expectedStart {
				fmt.Printf("❌ Range %s! (Subset of previous range %s)\n", blockRange{startBlock, stopBlock}, blockRange{previousRange.start, previousRange.stop})
			} else {
				fmt.Printf("✅ Range %s\n", blockRange{lastPrintedValidStart, expectedStart - 1})

				// This happens when current covers a superset of the last seen element (previous is `100 - 199` but we are `100 - 299`)
				if startBlock <= expectedStart {
					fmt.Printf("❌ Range %s! (Superset of previous range %s)\n", blockRange{startBlock, stopBlock}, blockRange{previousRange.start, previousRange.stop})
				} else {
					// Otherwise, we do not follow last seen element (previous is `100 - 199` but we are `299 - 300`)
					missingRange := blockRange{expectedStart, startBlock - 1}
					fmt.Printf("❌ Range %s! (Missing, [%s])\n", missingRange, missingRange.ReprocRange())
				}
			}

			problemDetected = true
			lastPrintedValidStart = stopBlock + 1
		} else if startBlock-lastPrintedValidStart >= 15_000_000 {
			fmt.Printf("✅ Range %s\n", blockRange{lastPrintedValidStart, stopBlock})
			lastPrintedValidStart = stopBlock + 1
		}

		previousRange = blockRange{startBlock, stopBlock}
		expectedStart = stopBlock + 1
		seenShard[shardIndex] = seenShard[shardIndex] + 1
		return nil
	})

	if len(seenShard) > 0 {
		fmt.Printf("✅ Range %s\n", blockRange{lastPrintedValidStart, previousRange.stop})
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
	storeURL := filepath.Clean(args[0])
	fileBlockSize := uint32(100)

	fmt.Printf("Checking block holes on %s\n", storeURL)

	number := regexp.MustCompile(`(\d{10})`)

	var expected uint32
	var count int
	var baseNum32 uint32
	// startTime := time.Now()
	holeFound := false
	checkIndividualSegment := viper.GetBool("individual-segment")
	printIndividualSegmentStats := viper.GetBool("print-stats")

	startBlock := viper.GetUint64("start-block")
	endBlock := viper.GetUint64("end-block")
	expected = uint32(startBlock)
	currentStartBlk := uint32(startBlock)

	fmt.Println("expected is ", expected)

	blocksStore, err := dstore.NewDBinStore(storeURL)
	if err != nil {
		return err
	}

	ctx := context.Background()
	blocksStore.Walk(ctx, "", ".tmp", func(filename string) error {
		match := number.FindStringSubmatch(filename)
		if match == nil {
			return nil
		}

		count++
		baseNum, _ := strconv.ParseUint(match[1], 10, 32)
		if baseNum < startBlock {
			return nil
		}
		if endBlock != 0 && baseNum > endBlock {
			return nil
		}
		baseNum32 = uint32(baseNum)

		if checkIndividualSegment {
			validateBlockSegment(blocksStore, filename, fileBlockSize, printIndividualSegmentStats)
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

		return nil
	})

	fmt.Printf("✅ Valid blocks range %d - %d\n", currentStartBlk, roundToBundleEndBlock(baseNum32, fileBlockSize))

	if holeFound {
		fmt.Printf("🆘 Holes found!\n")
	} else {
		fmt.Printf("🆗 No hole found\n")
	}

	return nil
}

func roundToBundleEndBlock(block, fileBlockSize uint32) uint32 {
	// From a non-rounded block `1085` and size of `100`, we remove from it the value of
	// `modulo % fileblock` (`85`) making it flush (`1000`) than adding to it the last
	// merged block num value for this size which simply `size - 1` (`99`) giving us
	// a resolved formulae of `1085 - (1085 % 100) + (100 - 1) = 1085 - (85) + (99)`.
	return block - (block % fileBlockSize) + (fileBlockSize - 1)
}

func validateBlockSegment(store dstore.Store, segment string, fileBlockSize uint32, printIndividualSegmentStats bool) {
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

	startBlock := uint64(viper.GetUint64("start-block"))
	endBlock := uint64(viper.GetUint64("end-block"))

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
