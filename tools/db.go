package tools

import (
	"encoding/json"
	"fmt"
	"strconv"

	pbcodec "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/codec/v1"
	trxdb "github.com/dfuse-io/dfuse-eosio/trxdb"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/streamingfast/cli"
	"github.com/streamingfast/kvdb"
	_ "github.com/streamingfast/kvdb/store/badger"
	_ "github.com/streamingfast/kvdb/store/bigkv"
	_ "github.com/streamingfast/kvdb/store/tikv"
)

var dbCmd = &cobra.Command{Use: "db", Short: "Read from EOS Database (trxdb)"}

var dbBlkCmd = &cobra.Command{
	Use:   "blk [hash]",
	Short: "Reads a block",
	RunE:  dbReadBlockE,
	Args:  cobra.ExactArgs(1),
}

var dbTrxCmd = &cobra.Command{
	Use:   "trx [hash]",
	Short: "Reads a transaction",
	Long: Description(`
		Reads a transaction using it's hash from the database. The transaction's event are read
		from the database then stiched together to form the final transaction.

		Note: This command is not currently able to discriminate which block is the canonical block
		and which is not. This means that if a transaction appears in multiple blocks, irreversible
		vs reversible cannot be determined yet.
	`),
	RunE: dbReadTrxE,
	Args: cobra.ExactArgs(1),
	Example: ExamplePrefixed("dfuseeos tools db", `
		trx --dsn="badger://./dfuse-data/storage/trxdb-v1" 85e1e337b06954c973ef5997c572a4462dd526b10a8e3220d9cd673e8add98a7
	`),
}

var dbTrxEventsCmd = &cobra.Command{
	Use:   "trx-events [hash]",
	Short: "Reads individual transaction events",
	Long: Description(`
		Reads a transaction's events using its hash from the database. The transaction's event are read
		from the database and printed as-is, this can help diagnose stiching issues.
	`),
	RunE: dbReadTrxEventsE,
	Args: cobra.ExactArgs(1),
	Example: ExamplePrefixed("dfuseeos tools db", `
		trx-events --dsn="badger://./dfuse-data/storage/trxdb-v1" 85e1e337b06954c973ef5997c572a4462dd526b10a8e3220d9cd673e8add98a7
	`),
}

var chainDiscriminator = func(blockID string) bool {
	return true
}

func init() {
	Cmd.AddCommand(dbCmd)
	dbCmd.AddCommand(dbBlkCmd)
	dbCmd.AddCommand(dbTrxCmd)
	dbCmd.AddCommand(dbTrxEventsCmd)

	dbCmd.PersistentFlags().String("dsn", "badger:///dfuse-data/kvdb/kvdb_badger.db", "kvStore DSN")
}

func dbReadBlockE(cmd *cobra.Command, args []string) (err error) {
	db, err := trxdb.New(viper.GetString("dsn"), trxdb.WithLogger(zlog))
	if err != nil {
		return fmt.Errorf("failed to setup db: %w", err)
	}

	blocks := []*pbcodec.BlockWithRefs{}

	if blockNum, err := strconv.ParseUint(args[0], 10, 32); err == nil {
		dbBlocks, err := db.GetBlockByNum(cmd.Context(), uint32(blockNum))
		if err != nil {
			return fmt.Errorf("failed to get block: %w", err)
		}
		blocks = append(blocks, dbBlocks...)
	} else {
		dbBlock, err := db.GetBlock(cmd.Context(), args[0])
		if err != nil {
			return fmt.Errorf("failed to get block: %w", err)
		}
		blocks = append(blocks, dbBlock)
	}

	for _, blk := range blocks {
		printEntity(blk)
	}
	return nil
}

func dbReadTrxE(cmd *cobra.Command, args []string) (err error) {
	trxID := args[0]

	db, err := trxdb.New(viper.GetString("dsn"), trxdb.WithLogger(zlog))
	cli.NoError(err, "Unable to create database instance")

	evs, err := db.GetTransactionEvents(cmd.Context(), trxID)
	if err == kvdb.ErrNotFound {
		fmt.Printf("Transaction %q not found\n", trxID)
		return nil
	}

	cli.Ensure(len(evs) > 0, "No events found for transaction %q", trxID)
	cli.NoError(err, "Unable to retrieve transaction from database")
	transaction := pbcodec.MergeTransactionEvents(evs, chainDiscriminator)

	fmt.Println("Printing!")
	printEntity(transaction)
	return nil
}

func dbReadTrxEventsE(cmd *cobra.Command, args []string) (err error) {
	trxID := args[0]

	db, err := trxdb.New(viper.GetString("dsn"), trxdb.WithLogger(zlog))
	if err != nil {
		return err
	}

	evs, err := db.GetTransactionEvents(cmd.Context(), trxID)
	if err == kvdb.ErrNotFound {
		fmt.Printf("Transaction %q not found\n", trxID)
		return nil
	}

	if err != nil {
		return fmt.Errorf("failed to get transaction: %w", err)
	}

	for _, event := range evs {
		printEntity(event)
	}

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
