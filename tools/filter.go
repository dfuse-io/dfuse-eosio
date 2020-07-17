package tools

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/dfuse-io/dfuse-eosio/codec"
	"github.com/dfuse-io/dfuse-eosio/filtering"
	"github.com/dfuse-io/dstore"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var filterCmd = &cobra.Command{Use: "filter", Short: "Various filters for deployment, data integrity & debugging"}
var filterMergedBlocksCmd = &cobra.Command{
	Use:   "merged-blocks {src-store-url} {dest-store-url}",
	Short: "Filters for any holes in merged blocks as well as ensuring merged blocks integrity",
	Args:  cobra.ExactArgs(2),
	RunE:  filterMergedBlocksE,
}

func init() {
	Cmd.AddCommand(filterCmd)
	filterCmd.AddCommand(filterMergedBlocksCmd)

	filterMergedBlocksCmd.Flags().StringP("include-filter-expr", "i", "true", "CEL expression to filter on")
	filterMergedBlocksCmd.Flags().StringP("exclude-filter-expr", "x", "false", "CEL expression to filter out")
	filterMergedBlocksCmd.Flags().Int64P("start-block", "s", 0, "Block number to start at")
	filterMergedBlocksCmd.Flags().Int64P("end-block", "e", 4294967296, "Block number to end at")
}

func filterMergedBlocksE(cmd *cobra.Command, args []string) error {
	cmd.SilenceUsage = true

	srcStoreURL := args[0]
	destStoreURL := args[1]

	srcStore, err := dstore.NewDBinStore(srcStoreURL)
	if err != nil {
		return err
	}
	fmt.Printf("✅ Reading from: %s\n", srcStoreURL)

	destStore, err := dstore.NewDBinStore(destStoreURL)
	if err != nil {
		return err
	}
	fmt.Printf("✅ Writing to (overwriting disabled): %s\n", destStoreURL)

	startBlock := viper.GetUint64("start-block")
	endBlock := viper.GetUint64("end-block")

	if startBlock%100 != 0 {
		return fmt.Errorf("start-block should be rounded to 100")
	}

	if endBlock%100 != 0 {
		return fmt.Errorf("end-block should be rounded to 100")
	}

	filter, err := filtering.NewBlockFilter(viper.GetString("include-filter-expr"), viper.GetString("exclude-filter-expr"))
	if err != nil {
		return err
	}

	ctx := context.Background()

	currentBase := startBlock
	var lastPrinted string
	for {
		currentBaseFile := fmt.Sprintf("%010d", currentBase)

		if lastPrinted != currentBaseFile {
			fmt.Printf("Processing %s\n", currentBaseFile)
		}
		lastPrinted = currentBaseFile

		destExists, err := destStore.FileExists(ctx, currentBaseFile)
		if err != nil {
			return err
		}
		if !destExists {

			srcExists, err := srcStore.FileExists(ctx, currentBaseFile)
			if err != nil {
				return err
			}

			if !srcExists {
				fmt.Printf("⏲  Waiting for %q\n", currentBaseFile)
				time.Sleep(5 * time.Second)
				continue
			}

			var count int
			err = func() error {
				ctx, cancel := context.WithCancel(ctx)
				defer cancel()

				readCloser, err := srcStore.OpenObject(ctx, currentBaseFile)
				if err != nil {
					return err
				}
				defer readCloser.Close()

				blkReader, err := codec.NewBlockReader(readCloser)
				if err != nil {
					return err
				}

				readPipe, writePipe, err := os.Pipe()
				if err != nil {
					return err
				}

				writeObjectDone := make(chan error, 1)
				go func() {
					writeObjectDone <- destStore.WriteObject(context.Background(), currentBaseFile, readPipe)
				}()

				blkWriter, err := codec.NewBlockWriter(writePipe)
				if err != nil {
					return err
				}

				for {
					blk, err := blkReader.Read()
					if err == io.EOF {
						break
					}
					if err != nil {
						return err
					}
					count++

					if err = filter.TransformInPlace(blk); err != nil {
						return err
					}

					if err = blkWriter.Write(blk); err != nil {
						return err
					}
				}

				err = writePipe.Close()
				if err != nil {
					return err
				}

				err = <-writeObjectDone
				if err != nil {
					return err
				}

				return nil
			}()
			if err != nil {
				return err
			}

			fmt.Printf("✅ Uploaded filtered %q, containing %d blocks\n", currentBaseFile, count)
		} else {
			fmt.Printf("✅ File exists %q\n", currentBaseFile)
		}

		currentBase += 100
		if currentBase >= endBlock {
			break
		}
	}

	return nil
}
