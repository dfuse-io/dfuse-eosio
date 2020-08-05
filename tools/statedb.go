package tools

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/dfuse-io/fluxdb"

	// Let's register all known Singlet & Tablet
	_ "github.com/dfuse-io/dfuse-eosio/statedb"

	"github.com/dfuse-io/kvdb/store"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var showValue = false

var statedbCmd = &cobra.Command{Use: "state", Short: "Read from StateDB"}
var statedbScanCmd = &cobra.Command{Use: "scan", Short: "scan read from StateDB store", RunE: statedbScanE, Args: cobra.ExactArgs(2)}
var statedbPrefixCmd = &cobra.Command{Use: "prefix", Short: "prefix read from StateDB store", RunE: prefixScanE, Args: cobra.ExactArgs(1)}

func init() {
	Cmd.AddCommand(statedbCmd)
	statedbCmd.AddCommand(statedbScanCmd)
	statedbCmd.AddCommand(statedbPrefixCmd)

	defaultBadger := "badger://dfuse-data/storage/statedb-v1"
	cwd, err := os.Getwd()
	if err == nil {
		defaultBadger = "badger://" + cwd + "/dfuse-data/storage/statedb-v1"
	}

	statedbCmd.PersistentFlags().String("dsn", defaultBadger, "StateDB KV store DSN")
	statedbCmd.PersistentFlags().StringP("table", "t", "00", "StateDB table id (single byte, hexadecimal encoded) to query from")

	statedbScanCmd.Flags().Bool("unlimited", false, "scan will ignore the limit")
	statedbCmd.PersistentFlags().Int("limit", 100, "limit the number of rows when doing scan or prefix")
}

func statedbScanE(cmd *cobra.Command, args []string) (err error) {
	kv, err := store.New(viper.GetString("dsn"))
	if err != nil {
		return err
	}

	limit := viper.GetInt("limit")
	if viper.GetBool("unlimited") {
		limit = store.Unlimited
	}

	table, err := hex.DecodeString(viper.GetString("table"))
	if err != nil {
		return fmt.Errorf("table: %w", err)
	}

	startKey, err := hex.DecodeString(args[0])
	if err != nil {
		return fmt.Errorf("start key: %w", err)
	}

	endKey, err := hex.DecodeString(args[1])
	if err != nil {
		return fmt.Errorf("end key: %w", err)
	}

	start := append(table, startKey...)
	end := append(table, endKey...)

	rangeScan(kv, start, end, limit)
	return nil
}

func prefixScanE(cmd *cobra.Command, args []string) (err error) {
	kv, err := store.New(viper.GetString("dsn"))
	if err != nil {
		return err
	}

	table, err := hex.DecodeString(viper.GetString("table"))
	if err != nil {
		return fmt.Errorf("table: %w", err)
	}

	prefixKey, err := hex.DecodeString(args[0])
	if err != nil {
		return fmt.Errorf("prefix key: %w", err)
	}

	prefix := append(table, prefixKey...)
	prefixScan(kv, prefix, viper.GetInt("limit"))
	return nil
}

func prefixScan(store store.KVStore, prefix []byte, limit int) error {
	prefixCtx, cancelScan := context.WithCancel(context.Background())
	defer cancelScan()

	return printIterator(store.Prefix(prefixCtx, prefix, limit))
}

func rangeScan(store store.KVStore, keyStart, keyEnd []byte, limit int) error {
	prefixCtx, cancelScan := context.WithCancel(context.Background())
	defer cancelScan()

	return printIterator(store.Scan(prefixCtx, keyStart, keyEnd, limit))
}
func printIterator(it *store.Iterator) error {
	count := 0
	start := time.Now()
	for it.Next() {
		count++
		kv := it.Item()
		key, err := formatKey(kv.Key)
		if err != nil {
			return err
		}

		cnt, err := json.Marshal(map[string]interface{}{
			"key": map[string]string{
				"hex":   hex.EncodeToString(kv.Key[1:]),
				"human": key,
			},
			"value": hex.EncodeToString(kv.Value),
		})
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

	return nil
}

var tableRows = []byte{0x00}
var tableCheckpoint = []byte{0x01}

func formatKey(key []byte) (string, error) {
	if bytes.Equal(key[0:1], tableRows) {
		return formatRowsKey(key)
	}

	if bytes.Equal(key[0:1], tableCheckpoint) {
		return formatCheckpointKey(key)
	}

	return "", fmt.Errorf("unknown key table")
}

func formatRowsKey(key []byte) (string, error) {
	key = key[1:]
	if len(key) == 0 {
		return "", nil
	}

	collection := binary.BigEndian.Uint16(key)
	if (collection >= 0xA000 && collection <= 0xAFFF) || collection == 0xFFFF {
		singlet, err := fluxdb.NewSinglet(key)
		if err != nil {
			return "", fmt.Errorf("invalid singlet: %w", err)
		}

		// We are interested by the key only
		entry, err := fluxdb.NewSingletEntry(singlet, key, nil)
		if err != nil {
			return "", fmt.Errorf("invalid singlet entry: %w", err)
		}

		return entry.String(), nil
	}

	if collection >= 0xB000 && collection <= 0xBFFF {
		tablet, err := fluxdb.NewTablet(key)
		if err != nil {
			return "", fmt.Errorf("invalid tablet: %w", err)
		}

		// We are interested by the key only
		row, err := fluxdb.NewTabletRow(tablet, key, nil)
		if err != nil {
			return "", fmt.Errorf("invalid tablet row: %w", err)
		}

		return row.String(), nil
	}

	return "", fmt.Errorf("unknown key collection")
}

func formatCheckpointKey(key []byte) (string, error) {
	return string(key[1:]), nil
}
