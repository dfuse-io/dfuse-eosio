package tools

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"

	"github.com/coreos/etcd/store"
	"github.com/dfuse-io/jsonpb"
	"github.com/dfuse-io/kvdb"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/protobuf/proto"
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

	kvCmd.PersistentFlags().StringP("store", "s", "badger:///dfuse-data/kvdb/kvdb_badger.db", "KVStore DSN")
	kvCmd.PersistentFlags().IntP("depth", "d", 1, "Depth of decoding. 0 = top-level block, 1 = kind-specific blocks, 2 = future!")

	kvScanCmd.Flags().IntP("limit", "l", 100, "limit the number of rows when doing scan")
}

func kvPrefix(cmd *cobra.Command, args []string) (err error) {
	kv, err := store.New(viper.GetString("store")) // FIXME: grab the right flags
	if err != nil {
		return err
	}

	prefix, err := hex.DecodeString(args[0])
	if err != nil {
		return fmt.Errorf("error decoding prefix %q: %s", args[0], err)
	}
	it := kv.Prefix(context.Background(), prefix)
	for it.Next() {
		item := it.Item()

		doKVPrint(item.Key, item.Value)
	}
	if err := it.Err(); err != nil {
		return err
	}

	return nil
}

func kvScan(cmd *cobra.Command, args []string) (err error) {
	kv, err := store.New(viper.GetString("store"))
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

		doKVPrint(item.Key, item.Value)
	}
	if err := it.Err(); err != nil {
		return err
	}

	return nil
}

func kvGet(cmd *cobra.Command, args []string) (err error) {
	kv, err := store.New(viper.GetString("store"))
	if err != nil {
		return err
	}

	key, err := hex.DecodeString(args[0])
	if err != nil {
		return fmt.Errorf("error decoding range start %q: %s", args[0], err)
	}

	val, err := kv.Get(context.Background(), key)
	if err == kvdb.ErrNotFound {
		os.Exit(1)
	}

	doKVPrint(key, val)

	return nil
}

func doKVPrint(key, val []byte, asHex bool, indented bool) error {
	if asHex {
		fmt.Println(hex.EncodeToString(key), hex.EncodeToString(val))
	}

	row := map[string]interface{}{
		"key": hex.EncodeToString(key),
	}

	switch key[0] {
	case 0x00:
		// decode as a transaction, add the key elements to the formatted row as _ fields
		//protoMessage := getProtoMap(protocol, key)
		row["data"] = nil
	}

	cnt, err := json.Marshal(formatedRow)
	if err != nil {
		return err
	}
	fmt.Println(string(cnt))
}

func decodePayload(marshaler jsonpb.Marshaler, obj proto.Message, bytes []byte) (out json.RawMessage, err error) {
	err = proto.Unmarshal(bytes, obj)
	if err != nil {
		return nil, fmt.Errorf("proto unmarshal: %s", err)
	}

	cnt, err := marshaler.MarshalToString(obj)
	if err != nil {
		return nil, fmt.Errorf("json marshal: %s", err)
	}

	return json.RawMessage(cnt), nil
}
