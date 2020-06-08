package tools

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"

	pbtrxdb "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/trxdb/v1"
	trxdb "github.com/dfuse-io/dfuse-eosio/trxdb/kv"
	"github.com/dfuse-io/jsonpb"
	"github.com/dfuse-io/kvdb/store"
	_ "github.com/dfuse-io/kvdb/store/badger"
	_ "github.com/dfuse-io/kvdb/store/bigkv"
	_ "github.com/dfuse-io/kvdb/store/tikv"
	"github.com/golang/protobuf/proto"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var kvCmd = &cobra.Command{Use: "kv", Short: "Read from a KVStore"}

var kvPrefixCmd = &cobra.Command{Use: "prefix", Short: "prefix read from KVStore", RunE: kvPrefix, Args: cobra.ExactArgs(1)}
var kvScanCmd = &cobra.Command{Use: "scan", Short: "scan read from KVStore", RunE: kvScan, Args: cobra.ExactArgs(2)}
var kvGetCmd = &cobra.Command{Use: "get", Short: "get key from KVStore", RunE: kvGet, Args: cobra.ExactArgs(1)}

func init() {
	Cmd.AddCommand(kvCmd)
	kvCmd.AddCommand(kvPrefixCmd)
	kvCmd.AddCommand(kvScanCmd)
	kvCmd.AddCommand(kvGetCmd)

	kvCmd.PersistentFlags().String("dsn", "badger:///dfuse-data/kvdb/kvdb_badger.db", "KVStore DSN")
	kvCmd.PersistentFlags().Int("depth", 1, "Depth of decoding. 0 = top-level block, 1 = kind-specific blocks, 2 = future!")
	kvScanCmd.Flags().Int("limit", 100, "limit the number of rows when doing scan")
}

func kvPrefix(cmd *cobra.Command, args []string) (err error) {
	prefix, err := hex.DecodeString(args[0])
	if err != nil {
		return fmt.Errorf("error decoding prefix %q: %s", args[0], err)
	}
	return getPrefix(prefix)
}

func kvScan(cmd *cobra.Command, args []string) (err error) {
	kv, err := store.New(viper.GetString("dsn"))
	if err != nil {
		return err
	}

	start, err := hex.DecodeString(args[0])
	if err != nil {
		return fmt.Errorf("error decoding range start %q: %s", args[0], err)
	}
	end, err := hex.DecodeString(args[1])
	if err != nil {
		return fmt.Errorf("error decoding range end %q: %s", args[1], err)
	}

	limit := viper.GetInt("limit")

	it := kv.Scan(context.Background(), start, end, limit)
	for it.Next() {
		item := it.Item()
		printKVEntity(item.Key, item.Value, false, true)
	}
	if err := it.Err(); err != nil {
		return err
	}

	return nil
}

func kvGet(cmd *cobra.Command, args []string) (err error) {
	key, err := hex.DecodeString(args[0])
	if err != nil {
		return fmt.Errorf("error decoding range start %q: %s", args[0], err)
	}
	return get(key)
}

func get(key []byte) error {
	kv, err := store.New(viper.GetString("dsn"))
	if err != nil {
		return err
	}

	val, err := kv.Get(context.Background(), key)
	if err == store.ErrNotFound {
		fmt.Printf("key %q not found\n", hex.EncodeToString(key))
		return nil
	}

	printKVEntity(key, val, false, true)

	return nil
}

func getPrefix(prefix []byte) error {
	kv, err := store.New(viper.GetString("dsn"))
	if err != nil {
		return err
	}

	it := kv.Prefix(context.Background(), prefix, store.Unlimited)
	for it.Next() {
		item := it.Item()
		printKVEntity(item.Key, item.Value, false, true)
	}
	if err := it.Err(); err != nil {
		return err
	}

	return nil
}

func printKVEntity(key, val []byte, asHex bool, indented bool) (err error) {
	if asHex {
		fmt.Println(hex.EncodeToString(key), hex.EncodeToString(val))
		return nil
	}

	pbmarsh := jsonpb.Marshaler{
		EnumsAsInts:  false,
		EmitDefaults: true,
		OrigName:     true,
	}

	row := map[string]interface{}{
		"key": hex.EncodeToString(key),
	}

	switch key[0] {
	case trxdb.TblPrefixTrxs:
		protoMessage := &pbtrxdb.TrxRow{}
		row["data"], err = decodePayload(pbmarsh, protoMessage, val)
	case trxdb.TblPrefixBlocks:
		protoMessage := &pbtrxdb.BlockRow{}
		row["data"], err = decodePayload(pbmarsh, protoMessage, val)
	case trxdb.TblPrefixIrrBlks:
		row["data"] = val[0] == 0x01
	case trxdb.TblPrefixImplTrxs:
		protoMessage := &pbtrxdb.ImplicitTrxRow{}
		row["data"], err = decodePayload(pbmarsh, protoMessage, val)
	case trxdb.TblPrefixDtrxs:
		protoMessage := &pbtrxdb.DtrxRow{}
		row["data"], err = decodePayload(pbmarsh, protoMessage, val)
	case trxdb.TblPrefixTrxTraces:
		protoMessage := &pbtrxdb.TrxTraceRow{}
		row["data"], err = decodePayload(pbmarsh, protoMessage, val)
	case trxdb.TblPrefixAccts:
		protoMessage := &pbtrxdb.AccountRow{}
		row["data"], err = decodePayload(pbmarsh, protoMessage, val)
	}

	cnt, err := json.Marshal(row)
	if err != nil {
		return err
	}
	fmt.Println(string(cnt))
	return nil
}

func decodePayload(marshaler jsonpb.Marshaler, obj proto.Message, bytes []byte) (out json.RawMessage, err error) {
	err = proto.Unmarshal(bytes, obj)
	if err != nil {
		return nil, fmt.Errorf("proto unmarshal: %s", err)
	}

	cnt, err := marshaler.MarshalToString(
		obj)
	if err != nil {
		return nil, fmt.Errorf("json marshal: %s", err)
	}

	return json.RawMessage(cnt), nil
}
