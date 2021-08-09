package tools

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/dfuse-io/dfuse-eosio/statedb"
	"github.com/dfuse-io/dstore"
	"github.com/dustin/go-humanize"
	"github.com/eoscanada/eos-go"
	"github.com/klauspost/compress/zstd"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/streamingfast/fluxdb"
	"github.com/streamingfast/kvdb/store"
)

var showValue = false

var statedbCmd = &cobra.Command{Use: "state", Short: "Read from StateDB"}

// Lower-level (key) calls
var statedbKeyCmd = &cobra.Command{Use: "key", Short: "Various operations on key", RunE: statedbKeyE, Args: cobra.MinimumNArgs(1)}
var statedbScanCmd = &cobra.Command{Use: "scan", Short: "Scan read from StateDB store", RunE: statedbScanE, Args: cobra.MaximumNArgs(2)}
var statedbPrefixCmd = &cobra.Command{Use: "prefix", Short: "Prefix read from StateDB store", RunE: statedbPrefixE, Args: cobra.MinimumNArgs(1)}

// Higher-level (model) calls
var statedbIndexCmd = &cobra.Command{Use: "index", Short: "Various operations related to StateDB Tablet Indexes"}
var statedbIndexPruneCmd = &cobra.Command{
	Use:   "prune",
	Short: "Prune all StateDB Tablet Index(es) before the given height (or HEAD)",
	Args:  cobra.ExactArgs(1),
	RunE:  statedbIndexPruneE,
	Example: ExamplePrefixed("dfuseeos tools state index", `
		prune --dsn="badger://./dfuse-data/storage/statedb-v1" all --frequency 3
		prune --dsn="bigkv://gcp_project.gcp_bt_instance/eos-kylin-v1" all --frequency 3
		prune --dsn="tikv://hostname.local:2379/eos-kylin-v1" all --frequency 3
	`),
}

var statedbIndexFetchCmd = &cobra.Command{Use: "fetch", Short: "Fetch and print the latest effective index for a given StateDB tablet", RunE: statedbIndexFetchE, Args: cobra.ExactArgs(1)}
var statedbIndexRegenerateCmd = &cobra.Command{Use: "regenerate", Short: "Re-index a given StateDB tablet", RunE: statedbIndexRegenerateE, Args: cobra.ExactArgs(1)}
var statedbTabletCmd = &cobra.Command{Use: "tablet", Short: "Fetch & print StateDB tablet, optionally at given height", RunE: statedbTabletE, Args: cobra.ExactArgs(1)}
var statedbShardCmd = &cobra.Command{Use: "shard", Short: "Various operations related to sharding"}
var statedbShardInspectCmd = &cobra.Command{Use: "inspect <shard-file>", Short: "Inspect given shard, printing write requests information stored in", RunE: statedbShardInspectE, Args: cobra.ExactArgs(1)}
var statedbShardCleanCmd = &cobra.Command{Use: "clean", Short: "Various operations related to shard cleaning"}
var statedbShardCleanCheckpointsCmd = &cobra.Command{Use: "checkpoints", Short: "Delete all existing shard checkpoint(s) that can exist", RunE: statedbShardCleanCheckpointsE, Args: cobra.ExactArgs((0))}

func init() {
	defaultBadger := "badger://dfuse-data/storage/statedb-v1"
	cwd, err := os.Getwd()
	if err == nil {
		defaultBadger = "badger://" + cwd + "/dfuse-data/storage/statedb-v1"
	}

	statedbCmd.PersistentFlags().String("dsn", defaultBadger, "StateDB KV store DSN")
	statedbCmd.PersistentFlags().StringP("table", "t", "00", "StateDB table id (single byte, hexadecimal encoded) to query from")
	statedbCmd.PersistentFlags().Int("limit", 100, "Limit the number of rows when doing scan or prefix")

	statedbPrefixCmd.PersistentFlags().Bool("key-only", false, "Only retrieve keys and not value when performing prefix search")
	statedbPrefixCmd.PersistentFlags().Bool("unlimited", false, "Returns all results, ignore the limit value")

	statedbScanCmd.PersistentFlags().Bool("key-only", false, "Only retrieve keys and not value when performing scan")
	statedbScanCmd.PersistentFlags().Bool("unlimited", false, "Returns all results, ignore the limit value")

	statedbIndexCmd.PersistentFlags().Uint64("height", 0, "Block height where to look for the index, 0 means use latest block")

	statedbIndexPruneCmd.PersistentFlags().Uint64("frequency", 0, "Pruning frequency, 1/N indexes will be pruned from the storage engine to reclaim space, never deleted most recent and least recent indexes")
	statedbIndexPruneCmd.PersistentFlags().Bool("write", false, "Write deleted entries to storage engine and not just print it")
	statedbIndexPruneCmd.PersistentFlags().String("lower-bound", "", "Lower bound tablet where to start pruning from, will skip any index for which the tablet is before this boundary")

	statedbIndexRegenerateCmd.PersistentFlags().Bool("write", false, "Write back index to storage engine and not just print it")
	statedbIndexRegenerateCmd.PersistentFlags().String("lower-bound", "", "Lower bound tablet where to start re-indexing from, will skip any index for which the tablet is before this boundary")

	statedbTabletCmd.PersistentFlags().Uint64("height", 0, "Block height where to create the index at, 0 means use latest block")

	statedbShardInspectCmd.PersistentFlags().Uint64("height", 0, "Block height where to start inspection, 0 means everything")

	Cmd.AddCommand(statedbCmd)
	statedbCmd.AddCommand(statedbKeyCmd)
	statedbCmd.AddCommand(statedbScanCmd)
	statedbCmd.AddCommand(statedbPrefixCmd)
	statedbCmd.AddCommand(statedbIndexCmd)
	statedbCmd.AddCommand(statedbTabletCmd)
	statedbCmd.AddCommand(statedbShardCmd)

	statedbIndexCmd.AddCommand(statedbIndexFetchCmd)
	statedbIndexCmd.AddCommand(statedbIndexPruneCmd)
	statedbIndexCmd.AddCommand(statedbIndexRegenerateCmd)

	statedbShardCmd.AddCommand(statedbShardCleanCmd)
	statedbShardCmd.AddCommand(statedbShardInspectCmd)

	statedbShardCleanCmd.AddCommand(statedbShardCleanCheckpointsCmd)
}

func statedbKeyE(cmd *cobra.Command, args []string) (err error) {
	for _, arg := range args {
		keyBytes, err := hex.DecodeString(arg)
		if err != nil {
			return fmt.Errorf("invalid key: %w", err)
		}

		row, err := fluxdb.NewTabletRowFromStorage(keyBytes, nil)
		if err == nil {
			fmt.Println(row.String())
		} else {
			entry, err := fluxdb.NewSingletEntryFromStorage(keyBytes, nil)
			if err == nil {
				fmt.Println(entry.String())
			} else {
				fmt.Printf("Key %x is neither a Singlet Entry nor a Tablet Row\n", keyBytes)
			}
		}
	}

	return nil
}

func statedbScanE(cmd *cobra.Command, args []string) (err error) {
	kv, err := store.New(viper.GetString("dsn"), store.WithEmptyValue())
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

	startKey, err := stateDBStringToKey(args[0])
	if err != nil {
		return fmt.Errorf("start key: %w", err)
	}

	var endKey []byte
	if len(args) > 1 {
		endKey, err = stateDBStringToKey(args[1])
		if err != nil {
			return fmt.Errorf("end key: %w", err)
		}
	}

	start := append(table, startKey...)
	end := append(table, endKey...)

	rangeScan(kv, start, end, limit, viper.GetBool("key-only"))
	return nil
}

func statedbPrefixE(cmd *cobra.Command, args []string) (err error) {
	kv, err := store.New(viper.GetString("dsn"), store.WithEmptyValue())
	if err != nil {
		return err
	}

	table, err := hex.DecodeString(viper.GetString("table"))
	if err != nil {
		return fmt.Errorf("table: %w", err)
	}

	limit := viper.GetInt("limit")
	if viper.GetBool("unlimited") {
		limit = store.Unlimited
	}

	ctx := cmd.Context()
	for i, arg := range args {
		prefixKey, err := stateDBStringToKey(arg)
		if err != nil {
			return fmt.Errorf("prefix key: %w", err)
		}

		prefix := append(table, prefixKey...)

		if i != 0 {
			fmt.Println()
		}

		err = prefixScan(ctx, kv, prefix, limit, viper.GetBool("key-only"))
		if err != nil {
			return fmt.Errorf("prefix scan %x: %w", prefix, err)
		}
	}

	return nil
}

func statedbIndexFetchE(cmd *cobra.Command, args []string) (err error) {
	store, err := fluxdb.NewKVStore(viper.GetString("dsn"))
	if err != nil {
		return fmt.Errorf("new kv store: %w", err)
	}

	tablet, err := stringToTablet(args[0])
	if err != nil {
		return fmt.Errorf("invalid argument %q: %w", args[0], err)
	}

	ctx := cmd.Context()
	fdb := fluxdb.New(store, nil, &statedb.BlockMapper{}, true)

	height := viper.GetUint64("height")
	if height == 0 {
		height, _, err = fdb.FetchLastWrittenCheckpoint(ctx)
		if err != nil {
			return fmt.Errorf("fetch last checkpoint: %w", err)
		}
	}

	index, err := fdb.ReadTabletIndexAt(ctx, tablet, height)
	if err != nil {
		return fmt.Errorf("read tablet index: %w", err)
	}

	if index == nil {
		fmt.Printf("No tablet %s index yet\n", tablet)
		return nil
	}

	rows, err := index.Rows(tablet)
	if err != nil {
		return fmt.Errorf("index rows: %w", err)
	}

	sort.Slice(rows, func(i, j int) bool {
		return bytes.Compare([]byte(rows[i].PrimaryKey()), []byte(rows[j].PrimaryKey())) < 0
	})

	indexBytes, err := index.MarshalValue()
	if err != nil {
		return fmt.Errorf("marshal index: %w", err)
	}

	fmt.Printf("Tablet %s Index (%d rows at #%d, %s [%s compressed])\n", tablet, len(rows), index.AtHeight, byteCount(indexBytes), byteCount(compressBytes(indexBytes)))
	for _, row := range rows {
		fmt.Printf("- %s (at #%d)\n", row.String(), row.Height())
	}

	return nil
}

func statedbIndexPruneE(cmd *cobra.Command, args []string) (err error) {
	store, err := fluxdb.NewKVStore(viper.GetString("dsn"))
	if err != nil {
		return fmt.Errorf("new kv store: %w", err)
	}

	ctx := cmd.Context()
	fdb := fluxdb.New(store, nil, &statedb.BlockMapper{}, false)

	height := viper.GetUint64("height")
	dryRun := !viper.GetBool("write")
	lowerBound := viper.GetString("lower-bound")
	pruneFrequency := viper.GetUint64("frequency")

	if args[0] == "all" {
		var lowerBoundTablet fluxdb.Tablet
		if lowerBound != "" {
			lowerBoundTablet, err = stringToTablet(lowerBound)
			if err != nil {
				return fmt.Errorf("invalid lower-bound argument %q: %w", lowerBound, err)
			}
		}

		fmt.Printf("Pruning tablet indexes (dry run: %t)\n", dryRun)
		if dryRun {
			fmt.Println("You are doing a dry run, use --write flag to perform actual deletion of indexes")
		}

		tabletCount, indexCount, deletedCount, err := fdb.PruneTabletIndexes(ctx, int(pruneFrequency), height, lowerBoundTablet, dryRun)
		if err != nil {
			return fmt.Errorf("pruning failed: %w", err)
		}

		if dryRun {
			fmt.Printf("Tablet indexes NOT deleted, would have deleted %d out of %d indexes across %d tablets\n", deletedCount, indexCount, tabletCount)
		} else {
			fmt.Printf("Deleted %d out of %d indexes across %d tablets\n", deletedCount, indexCount, tabletCount)
		}

		return nil
	}

	return fmt.Errorf(`only "all" argument is accepted for now`)
}

func statedbIndexRegenerateE(cmd *cobra.Command, args []string) (err error) {
	store, err := fluxdb.NewKVStore(viper.GetString("dsn"))
	if err != nil {
		return fmt.Errorf("new kv store: %w", err)
	}

	ctx := cmd.Context()
	fdb := fluxdb.New(store, nil, &statedb.BlockMapper{}, false)

	height := viper.GetUint64("height")
	write := viper.GetBool("write")
	lowerBound := viper.GetString("lower-bound")

	if args[0] == "all" {
		return statedbReindexAll(ctx, fdb, height, lowerBound, write)
	}

	tablet, err := stringToTablet(args[0])
	if err != nil {
		return fmt.Errorf("invalid argument %q: %w", args[0], err)
	}

	return statedbReindexTablet(ctx, fdb, height, tablet, write)
}

func statedbReindexTablet(ctx context.Context, fdb *fluxdb.FluxDB, height uint64, tablet fluxdb.Tablet, write bool) (err error) {
	index, written, err := fdb.ReindexTablet(ctx, height, tablet, write)
	if err != nil {
		return fmt.Errorf("reindex tablet %s: %w", tablet, err)
	}

	rows, err := index.Rows(tablet)
	if err != nil {
		return fmt.Errorf("index rows: %w", err)
	}

	if written {
		fmt.Printf("Tablet %s Index (%d rows at #%d) written back to storage\n", tablet, len(rows), index.AtHeight)
	} else {
		sort.Slice(rows, func(i, j int) bool {
			return bytes.Compare([]byte(rows[i].PrimaryKey()), []byte(rows[j].PrimaryKey())) < 0
		})

		fmt.Printf("Tablet %s Index (%d rows at #%d)\n", tablet, len(rows), index.AtHeight)
		for _, row := range rows {
			fmt.Printf("- %s (at #%d)\n", row.String(), row.Height())
		}
	}

	return nil
}

func statedbReindexAll(ctx context.Context, fdb *fluxdb.FluxDB, height uint64, lowerBound string, write bool) (err error) {
	var lowerBoundTablet fluxdb.Tablet
	if lowerBound != "" {
		lowerBoundTablet, err = stringToTablet(lowerBound)
		if err != nil {
			return fmt.Errorf("invalid lower-bound argument %q: %w", lowerBound, err)
		}
	}

	fmt.Printf("Re-indexing all tablets (dry run: %t)\n", !write)
	tabletCount, indexCount, err := fdb.ReindexTablets(ctx, height, lowerBoundTablet, !write)
	if !write {
		fmt.Printf("Not re-writing indexes, would have affected %d tablet and %d overall indexes\n", tabletCount, indexCount)
	}

	return nil
}

func statedbTabletE(cmd *cobra.Command, args []string) (err error) {
	store, err := fluxdb.NewKVStore(viper.GetString("dsn"))
	if err != nil {
		return fmt.Errorf("new kv store: %w", err)
	}

	tablet, err := stringToTablet(args[0])
	if err != nil {
		return fmt.Errorf("invalid argument %q: %w", args[0], err)
	}

	ctx := cmd.Context()
	fdb := fluxdb.New(store, nil, &statedb.BlockMapper{}, true)

	height := viper.GetUint64("height")
	if height == 0 {
		height, _, err = fdb.FetchLastWrittenCheckpoint(ctx)
		if err != nil {
			return fmt.Errorf("fetch last checkpoint: %w", err)
		}
	}

	tabletRows, err := fdb.ReadTabletAt(ctx, height, tablet, nil)
	if err != nil {
		return fmt.Errorf("read tablet: %w", err)
	}

	if len(tabletRows) == 0 {
		fmt.Printf("Tablet %s has no row\n", tablet)
		return
	}

	fmt.Printf("Tablet %s (%d rows at #%d)\n", tablet, len(tabletRows), height)
	for _, tabletRow := range tabletRows {
		fmt.Printf("- %s (at #%d)\n", tabletRow.String(), tabletRow.Height())
	}

	return nil
}

func statedbShardInspectE(cmd *cobra.Command, args []string) (err error) {
	shardFile := args[0]
	compression := dstore.Compression("none")
	if strings.HasSuffix(shardFile, ".zst") {
		compression = dstore.Compression("zstd")
	}

	reader, _, _, err := dstore.OpenObject(cmd.Context(), shardFile, compression)
	if err != nil {
		return fmt.Errorf("open shard file: %w", err)
	}
	defer reader.Close()

	height := viper.GetUint64("height")
	requests, err := fluxdb.ReadShard(reader, height)

	fmt.Println("Singlets")
	for _, request := range requests {
		for _, singletEntry := range request.SingletEntries {
			fmt.Printf("- %s (deletion?: %t)\n", singletEntry, singletEntry.IsDeletion())
		}
	}

	fmt.Println()
	fmt.Println("Tablets")
	for _, request := range requests {
		for _, tabletRow := range request.TabletRows {
			fmt.Printf("- %s (deletion?: %t)\n", tabletRow, tabletRow.IsDeletion())
		}
	}

	return nil
}

func statedbShardCleanCheckpointsE(cmd *cobra.Command, args []string) (err error) {
	store, err := fluxdb.NewKVStore(viper.GetString("dsn"))
	if err != nil {
		return fmt.Errorf("new kv store: %w", err)
	}

	ctx := cmd.Context()
	fdb := fluxdb.New(store, nil, &statedb.BlockMapper{}, true)

	err = fdb.DeleteAllShardCheckpoints(ctx)
	if err != nil {
		return fmt.Errorf("delete shard checkpoints: %w", err)
	}

	fmt.Println("Completed deletion of all existing shard checkpoints")
	return nil
}

func prefixScan(ctx context.Context, kvStore store.KVStore, prefix []byte, limit int, keyOnly bool) error {
	prefixCtx, cancelScan := context.WithCancel(ctx)
	defer cancelScan()

	var options []store.ReadOption
	if keyOnly {
		options = []store.ReadOption{store.KeyOnly()}
	}

	return printIterator(kvStore.Prefix(prefixCtx, prefix, limit, options...))
}

func rangeScan(kvStore store.KVStore, keyStart, keyEnd []byte, limit int, keyOnly bool) error {
	prefixCtx, cancelScan := context.WithCancel(context.Background())
	defer cancelScan()

	var options []store.ReadOption
	if keyOnly {
		options = []store.ReadOption{store.KeyOnly()}
	}

	return printIterator(kvStore.Scan(prefixCtx, keyStart, keyEnd, limit, options...))
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

	if err := it.Err(); err != nil {
		fmt.Printf("Iteration error: %s (in %s)\n", err, time.Since(start))
	} else {
		fmt.Printf("Found %d keys (in %s)\n", count, time.Since(start))
	}

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

// stringToTablet receives a string format containing human readable form
// and turn it into the appropriate Tablet implementation. Highly manual for now.
func stringToTablet(in string) (fluxdb.Tablet, error) {
	parts := strings.Split(in, ":")
	if len(parts) <= 1 {
		return nil, fmt.Errorf("invalid format, expecting at least a table prefix like 'cst:...'")
	}

	mapper := partsToStateDBTabletMap[parts[0]]
	if mapper == nil {
		return nil, fmt.Errorf("unknown (or not yet handled) prefix %q", parts[0])
	}

	if len(parts)-1 != mapper.partCount {
		return nil, fmt.Errorf("invalid format, expecting %d parts, got %d", mapper.partCount, len(parts)-1)
	}

	return mapper.factory(parts[1:]), nil
}

func stateDBStringToKey(in string) ([]byte, error) {
	// We assume it's a string key to convert
	if strings.Contains(in, ":") {
		parts := strings.Split(in, ":")
		if len(parts) <= 1 {
			return nil, fmt.Errorf("invalid format, expecting at least a prefix and subsequent element like 'cst:...'")
		}

		transformer := partsToStateDBKeyMap[parts[0]]
		if transformer == nil {
			return nil, fmt.Errorf("unknown (or not yet handled) prefix %q", parts[0])
		}

		return transformer(parts[1:])
	}

	key, err := hex.DecodeString(in)
	if err != nil {
		return nil, fmt.Errorf("invalid hex %q: %w", in, err)
	}

	return key, nil
}

type partsToStateDBTablet struct {
	partCount int
	factory   func(parts []string) fluxdb.Tablet
}

var partsToStateDBTabletMap = map[string]*partsToStateDBTablet{
	"cst": {
		partCount: 3, factory: func(parts []string) fluxdb.Tablet {
			return statedb.NewContractStateTablet(parts[0], parts[1], parts[2])
		},
	},
	"ctscp": {
		partCount: 2, factory: func(parts []string) fluxdb.Tablet {
			return statedb.NewContractTableScopeTablet(parts[0], parts[1])
		},
	},
}

var partsToStateDBKeyMap = map[string]func(parts []string) ([]byte, error){
	"cst": func(parts []string) (out []byte, err error) {
		out = []byte{0xb0, 0x00}
		for i, part := range parts {
			switch {
			case i <= 2:
				out = append(out, nameToBytes(mustExtendedStringToName, part)...)
			case i == 3:
				bytes, err := heightToBytes(part)
				if err != nil {
					return nil, fmt.Errorf("invalid height %q: %w", part, err)
				}

				out = append(out, bytes...)
			default:
				out = append(out, nameToBytes(mustExtendedStringToName, part)...)
			}
		}

		return out, nil
	},
	"ctscp": func(parts []string) (out []byte, err error) {
		out = []byte{0xb2, 0x00}
		for i, part := range parts {
			switch {
			case i <= 1:
				out = append(out, nameToBytes(mustExtendedStringToName, part)...)
			case i == 2:
				bytes, err := heightToBytes(part)
				if err != nil {
					return nil, fmt.Errorf("invalid height %q: %w", part, err)
				}

				out = append(out, bytes...)
			default:
				out = append(out, nameToBytes(mustExtendedStringToName, part)...)
			}
		}

		return out, nil
	},
}

func nameToBytes(converter func(in string) uint64, names ...string) (out []byte) {
	out = make([]byte, 8*len(names))
	moving := out
	for _, name := range names {
		binary.BigEndian.PutUint64(moving, converter(name))
		moving = moving[8:]
	}

	return
}

func heightToBytes(heights ...string) (out []byte, err error) {
	out = make([]byte, 8*len(heights))
	moving := out
	for _, height := range heights {
		value, err := strconv.ParseUint(height, 16, 64)
		if err != nil {
			return nil, err
		}

		binary.BigEndian.PutUint64(moving, value)
		moving = moving[8:]
	}

	return
}

func mustExtendedStringToName(name string) uint64 {
	val, err := eos.ExtendedStringToName(name)
	if err != nil {
		panic(err)
	}

	return val
}

var enc, _ = zstd.NewWriter(nil)

func compressBytes(in []byte) (out []byte) {
	return enc.EncodeAll(in, out)
}

func byteCount(in []byte) string {
	return humanize.Bytes(uint64(len(in)))
}
