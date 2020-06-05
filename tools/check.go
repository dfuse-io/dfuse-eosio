package tools

import (
	"context"
	"fmt"
	"io"
	"regexp"
	"strconv"

	"github.com/dfuse-io/bstream"
	"github.com/dfuse-io/dfuse-eosio/fluxdb"
	pbcodec "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/codec/v1"
	"github.com/dfuse-io/dstore"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var checkCmd = &cobra.Command{Use: "check", Short: "Various checks for deployment, data integrity & debugging"}
var checkBlocksCmd = &cobra.Command{
	// TODO: Not sure, it's now a required thing, but we could probably use the same logic as `start`
	//       and avoid altogether passing the args. If this would also load the config and everything else,
	//       that would be much more seamless!
	Use:   "blocks {store-url}",
	Short: "Checks for any holes in merged blocks as well as ensuring merged blocks integrity",
	Args:  cobra.ExactArgs(1),
	RunE:  checkBlocksE,
}
var checkFluxShardsCmd = &cobra.Command{
	Use:   "flux-shards {dsn} {shard-count}",
	Short: "Checks to see if all shards are aligned in flux reprocessing",
	Args:  cobra.ExactArgs(2),
	RunE:  checkFluxShardsE,
}

func init() {
	Cmd.AddCommand(checkCmd)
	checkCmd.AddCommand(checkBlocksCmd)
	checkCmd.AddCommand(checkFluxShardsCmd)

	checkBlocksCmd.Flags().Bool("individual-segment", false, "Open each merged blocks segment and ensure it contains all blocks it should")
	checkBlocksCmd.Flags().Bool("print-stats", false, "Natively decode each block in the segment and print statistics about it")

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

func checkBlocksE(cmd *cobra.Command, args []string) error {
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
					eosBlock.GetTransactionCount(),
					eosBlock.GetTransactionTraceCount(),
					eosBlock.GetExecutedTotalActionCount(),
					eosBlock.GetExecuteInputActionCount(),
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
