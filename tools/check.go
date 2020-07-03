package tools

import (
	"context"
	"encoding/hex"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"time"

	"github.com/dfuse-io/bstream"
	"github.com/dfuse-io/dfuse-eosio/fluxdb"
	pbcodec "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/codec/v1"
	"github.com/dfuse-io/dfuse-eosio/trxdb/kv"
	"github.com/dfuse-io/dstore"
	"github.com/dfuse-io/kvdb/store"
	"github.com/eoscanada/eos-go"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
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

var checkFluxShardsCmd = &cobra.Command{
	Use:   "flux-shards {dsn} {shard-count}",
	Short: "Checks to see if all shards are aligned in flux reprocessing",
	Args:  cobra.ExactArgs(2),
	RunE:  checkFluxShardsE,
}

func init() {
	Cmd.AddCommand(checkCmd)
	checkCmd.AddCommand(checkMergedBlocksCmd)
	checkCmd.AddCommand(checkTrxdbBlocksCmd)
	checkCmd.AddCommand(checkFluxShardsCmd)

	checkMergedBlocksCmd.Flags().Bool("individual-segment", false, "Open each merged blocks segment and ensure it contains all blocks it should")
	checkMergedBlocksCmd.Flags().Bool("print-stats", false, "Natively decode each block in the segment and print statistics about it")

	checkTrxdbBlocksCmd.Flags().Int64P("start-block", "s", 0, "Block number to start at")
	checkTrxdbBlocksCmd.Flags().Int64P("end-block", "e", 4294967296, "Block number to end at")
}

func checkFluxShardsE(cmd *cobra.Command, args []string) error {
	cmd.SilenceUsage = true

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

	fdb := fluxdb.New(kvStore)
	fdb.SetSharding(0, int(shardsInt))
	lastBlock, err := fdb.VerifyAllShardsWritten(context.Background())
	if err != nil {
		return err
	}
	fmt.Println("last block: ", lastBlock)
	return nil
}

func checkMergedBlocksE(cmd *cobra.Command, args []string) error {
	cmd.SilenceUsage = true

	// FIXME: Seems `./dfuse-data/...` something doesn't work but `dfuse-data/...` works
	storeURL := args[0]
	fileBlockSize := uint32(100)

	fmt.Printf("Checking block holes on %s\n", storeURL)

	number := regexp.MustCompile(`(\d{10})`)

	var expected uint32
	var count int
	var baseNum32 uint32
	currentStartBlk := uint32(0)
	// startTime := time.Now()
	holeFound := false
	checkIndividualSegment := viper.GetBool("individual-segment")
	printIndividualSegmentStats := viper.GetBool("print-stats")

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
		baseNum32 = uint32(baseNum)

		if checkIndividualSegment {
			validateBlockSegment(blocksStore, filename, fileBlockSize, printIndividualSegmentStats)
		}

		if baseNum32 != expected {
			fmt.Printf("✅ Valid blocks range %d - %d\n", currentStartBlk, expected-fileBlockSize)
			fmt.Printf("❌ Missing blocks range %d - %d!\n", baseNum32-fileBlockSize, expected)
			currentStartBlk = baseNum32

			holeFound = true
		}
		expected = baseNum32 + fileBlockSize

		if count%10000 == 0 {
			fmt.Printf("✅ Valid blocks range %d - %d\n", currentStartBlk, baseNum32)
			currentStartBlk = baseNum32 + fileBlockSize
		}

		return nil
	})

	fmt.Printf("✅ Valid blocks range %d - %d\n", currentStartBlk, baseNum32)

	if holeFound {
		fmt.Printf("🆘 Holes found!\n")
	} else {
		fmt.Printf("🆗 No hole found\n")
	}

	return nil
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
				fmt.Printf("Block %s (%d bytes): %d transactions (%d traces), %d actions (%d input)\n",
					block,
					payloadSize,
					eosBlock.GetUnfilteredTransactionCount(),
					eosBlock.GetUnfilteredTransactionTraceCount(),
					eosBlock.GetUnfilteredExecutedTotalActionCount(),
					eosBlock.GetUnfilteredExecutedInputActionCount(),
				)
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
	cmd.SilenceUsage = true

	// FIXME: Seems `./dfuse-data/...` something doesn't work but `dfuse-data/...` works
	dsn := args[0]

	startBlock := uint64(viper.GetInt64("start-block"))
	endBlock := uint64(viper.GetInt64("end-block"))

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
