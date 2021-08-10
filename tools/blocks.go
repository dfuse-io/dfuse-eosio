package tools

import (
	"context"
	"fmt"
	"io"
	"path"
	"path/filepath"
	"strings"

	"github.com/dfuse-io/bstream"
	pbcodec "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/codec/v1"
	"github.com/streamingfast/dstore"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var blocksCmd = &cobra.Command{
	Use:   "print-blocks",
	Short: "Prints the content summary of a local merged blocks file",
	Args:  cobra.ExactArgs(1),
	RunE:  printBlocksE,
}

func init() {
	Cmd.AddCommand(blocksCmd)

	blocksCmd.Flags().Bool("transactions", false, "Include transaction IDs in output")
}

func printBlocksE(cmd *cobra.Command, args []string) error {
	printTransactions := viper.GetBool("transactions")
	file := args[0]
	abs, err := filepath.Abs(file)
	if err != nil {
		return err
	}

	dir := path.Dir(abs)
	storeURL := fmt.Sprintf("file://%s", dir)

	compression := ""
	if strings.HasSuffix(file, "zst") || strings.HasSuffix(file, "zstd") {
		compression = "zstd"
	}
	store, err := dstore.NewStore(storeURL, "", compression, false)
	if err != nil {
		return err
	}

	filename := path.Base(abs)
	reader, err := store.OpenObject(context.Background(), filename)
	if err != nil {
		fmt.Printf("❌ Unable to read blocks filename %s: %s\n", filename, err)
		return err
	}
	defer reader.Close()

	readerFactory, err := bstream.GetBlockReaderFactory.New(reader)
	if err != nil {
		fmt.Printf("❌ Unable to read blocks filename %s: %s\n", filename, err)
		return err
	}

	seenBlockCount := 0
	for {
		block, err := readerFactory.Read()
		if block != nil {
			seenBlockCount++

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
				fmt.Printf("Block #%d (%s) (prev: %s) (%d bytes): %d transactions (%d traces), %d actions (%d input)\n",
					block.Num(),
					block.ID(),
					block.PreviousID(),
					payloadSize,
					eosBlock.GetUnfilteredTransactionCount(),
					eosBlock.GetUnfilteredTransactionTraceCount(),
					eosBlock.GetUnfilteredExecutedTotalActionCount(),
					eosBlock.GetUnfilteredExecutedInputActionCount(),
				)
				if printTransactions {
					fmt.Println("- Transactions: ")
					for _, t := range eosBlock.Transactions() {
						fmt.Println("  * ", t.Id)
					}
					fmt.Println("- Transaction traces: ")
					for _, t := range eosBlock.TransactionTraces() {
						fmt.Println("  * ", t.Id)
					}
					fmt.Println()
				}
			}

			continue
		}

		if block == nil && err == io.EOF {
			fmt.Printf("Total blocks: %d\n", seenBlockCount)
			return nil
		}

		if err != nil {
			return err
		}
	}
}
