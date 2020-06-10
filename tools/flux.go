package tools

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/spf13/viper"

	"github.com/dfuse-io/kvdb/store"
	"github.com/spf13/cobra"
)

var showValue = false

var fluxCmd = &cobra.Command{Use: "flux", Short: "Read from flux"}

var fluxScanCmd = &cobra.Command{Use: "scan", Short: "scan read from flux KVStore", RunE: fluxScanE, Args: cobra.ExactArgs(2)}
var fluxPrefixCmd = &cobra.Command{Use: "prefix", Short: "prefix read from flux KVStore", RunE: prefixScanE, Args: cobra.ExactArgs(1)}

func init() {
	Cmd.AddCommand(fluxCmd)
	fluxCmd.AddCommand(fluxScanCmd)
	fluxCmd.AddCommand(fluxPrefixCmd)
	fluxCmd.PersistentFlags().String("dsn", "badger:///dfuse-data/kvdb/kvdb_badger.db", "KVStore DSN")

	fluxScanCmd.Flags().Bool("unlimited", false, "scan will ignore the limit")
	fluxCmd.PersistentFlags().Int("limit", 100, "limit the number of rows when doing scan or prefix")
}

func fluxScanE(cmd *cobra.Command, args []string) (err error) {
	kv, err := store.New(viper.GetString("dsn"))
	if err != nil {
		return err
	}

	limit := viper.GetInt("limit")
	if viper.GetBool("unlimited") {
		limit = store.Unlimited
	}

	start := append([]byte{0x00}, []byte(args[0])...)
	end := append([]byte{0x00}, []byte(args[1])...)

	rangeScan(kv, start, end, limit)
	return nil
}

func prefixScanE(cmd *cobra.Command, args []string) (err error) {
	kv, err := store.New(viper.GetString("dsn"))
	if err != nil {
		return err
	}

	prefix := append([]byte{0x00}, []byte(args[0])...)
	prefixScan(kv, prefix, viper.GetInt("limit"))
	return nil
}

func prefixScan(store store.KVStore, prefix []byte, limit int) {
	prefixCtx, cancelScan := context.WithCancel(context.Background())
	defer cancelScan()
	printIterator(store.Prefix(prefixCtx, prefix, limit))
}
func rangeScan(store store.KVStore, keyStart, keyEnd []byte, limit int) {
	prefixCtx, cancelScan := context.WithCancel(context.Background())
	defer cancelScan()
	printIterator(store.Scan(prefixCtx, keyStart, keyEnd, limit))
}
func printIterator(it *store.Iterator) {
	count := 0
	start := time.Now()
	for it.Next() {
		count++
		kv := it.Item()
		key := formatKey(kv.Key)
		row := map[string]interface{}{
			"key": key,
		}
		row["value"] = hex.EncodeToString(kv.Value)

		cnt, err := json.Marshal(row)
		if err != nil {
			fmt.Printf("unable to marshall row: %s\n", key)
		} else {
			fmt.Println(string(cnt))
		}
	}
	fmt.Println()
	if err := it.Err(); err != nil {
		fmt.Printf("Iteration error: %s\n", err)
	} else {
		fmt.Printf("Found %d keys\n", count)
	}
	fmt.Printf("In %ss\n", time.Since(start))
}
func formatKey(key []byte) string {
	return string(key[1:])
}
