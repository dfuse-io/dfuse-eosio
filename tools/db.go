package tools

import (
	"encoding/json"
	"fmt"

	pbcodec "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/codec/v1"
	"github.com/dfuse-io/kvdb"
	"go.uber.org/zap"

	trxdb "github.com/dfuse-io/dfuse-eosio/trxdb/kv"
	_ "github.com/dfuse-io/kvdb/store/badger"
	_ "github.com/dfuse-io/kvdb/store/bigkv"
	_ "github.com/dfuse-io/kvdb/store/tikv"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var dbCmd = &cobra.Command{Use: "db", Short: "Read from EOS db"}
var dbBlkCmd = &cobra.Command{Use: "blk", Short: "Read a Blk", RunE: dbReadBlockE, Args: cobra.ExactArgs(1)}
var dbTrxCmd = &cobra.Command{Use: "trx", Short: "Reads a Trx", RunE: dbReadTrxE, Args: cobra.ExactArgs(1)}

var chainDiscriminator = func(blockID string) bool {
	return true
}

func init() {
	Cmd.AddCommand(dbCmd)
	dbCmd.AddCommand(dbBlkCmd)
	dbCmd.AddCommand(dbTrxCmd)

	dbCmd.PersistentFlags().String("dsn", "badger:///dfuse-data/kvdb/kvdb_badger.db", "KVStore DSN")
}

func dbReadBlockE(cmd *cobra.Command, args []string) (err error) {
	db, err := trxdb.New(viper.GetString("dsn"), zap.NewNop())
	if err != nil {
		return fmt.Errorf("failed to setup db: %w", err)
	}

	dbBlock, err := db.GetBlock(cmd.Context(), args[0])
	if err != nil {
		return fmt.Errorf("failed to get block: %w", err)
	}

	printEntity(dbBlock)
	return nil
}

func dbReadTrxE(cmd *cobra.Command, args []string) (err error) {
	db, err := trxdb.New(viper.GetString("dsn"), zap.NewNop())
	if err != nil {
		return err
	}
	trxID := args[0]

	evs, err := db.GetTransactionEvents(cmd.Context(), trxID)
	if err == kvdb.ErrNotFound {
		return fmt.Errorf("Transaction %q not found", trxID)
	}
	if err != nil {
		return fmt.Errorf("Failed to get transaction: %w", err)
	}

	transaction := pbcodec.MergeTransactionEvents(evs, chainDiscriminator)

	printEntity(transaction)
	return nil
}

func printEntity(obj interface{}) (err error) {
	cnt, err := json.Marshal(obj)
	if err != nil {
		return err
	}
	fmt.Println(string(cnt))
	return nil
}
