package tools

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/dfuse-io/dfuse-eosio/accounthist/keyer"

	"github.com/dfuse-io/dfuse-eosio/accounthist"

	"github.com/dfuse-io/dfuse-eosio/accounthist/injector"

	"github.com/manifoldco/promptui"
	"go.uber.org/zap"

	"github.com/dfuse-io/kvdb/store"
	"github.com/eoscanada/eos-go"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var accounthistCmd = &cobra.Command{Use: "accounthist", Short: "Read from account history"}
var accountCmd = &cobra.Command{Use: "account", Short: "Account interactions"}
var checkpointCmd = &cobra.Command{Use: "checkpoint", Short: "Shard checkpoint interactions", Args: cobra.ExactArgs(1), RunE: readCheckpointE}

// dfuseeos tools accounthist account read {account} --dsn
var readAccountCmd = &cobra.Command{
	Use:   "read {account}",
	Short: "Read an account",
	Args:  cobra.ExactArgs(1),
	RunE:  readAccountE,
}

// dfuseeos tools accounthist account scan --dsn
var scanAccountCmd = &cobra.Command{
	Use:   "scan",
	Short: "Scan accounts",
	RunE:  scanAccountE,
}

// dfuseeos tools accounthist account scan --dsn
var readCheckpointCmd = &cobra.Command{
	Use:   "read",
	Short: "Read a shard's checkpoint",
	Args:  cobra.ExactArgs(1),
	RunE:  readCheckpointE,
}

var deleteCheckpointCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a shard's checkpoint",
	Args:  cobra.ExactArgs(1),
	RunE:  deleteCheckpointE,
}

func init() {
	Cmd.AddCommand(accounthistCmd)

	accounthistCmd.AddCommand(accountCmd)
	accountCmd.AddCommand(readAccountCmd, scanAccountCmd)

	accounthistCmd.AddCommand(checkpointCmd)
	checkpointCmd.AddCommand(readCheckpointCmd, deleteCheckpointCmd)

	accounthistCmd.PersistentFlags().String("dsn", "badger:///dfuse-data/kvdb/kvdb_badger.db", "kvStore DSN")
	scanAccountCmd.Flags().Int("limit", 100, "limit the number of accounts when doing scan")
}

func readCheckpointE(cmd *cobra.Command, args []string) (err error) {
	kvdb, err := store.New(viper.GetString("dsn"))
	if err != nil {
		return fmt.Errorf("failed to setup db: %w", err)
	}

	shardStr := args[0]
	shard, err := strconv.ParseUint(shardStr, 0, 64)
	if err != nil {
		return fmt.Errorf("unable to determine shard value from: %s: %w", shardStr, err)
	}

	service := newService(kvdb, shard)

	checkpoint, err := service.GetShardCheckpoint(cmd.Context())
	if err != nil {
		return err
	}
	if checkpoint == nil {
		fmt.Printf("No checkpoint found for shard: %d\n", shard)
		return nil
	}

	fmt.Printf("Checkpoint for shard: %d\n", shard)
	fmt.Printf("Initial Start Block for shard: %d\n", checkpoint.InitialStartBlock)
	fmt.Printf("Target Stop Block for shard: %d\n", checkpoint.TargetStopBlock)
	fmt.Printf("Last Written Block Num for shard: %d\n", checkpoint.LastWrittenBlockNum)
	fmt.Printf("Last Written Block Id for shard: %s\n", checkpoint.LastWrittenBlockId)

	return nil
}

func deleteCheckpointE(cmd *cobra.Command, args []string) (err error) {
	kvdb, err := store.New(viper.GetString("dsn"))
	if err != nil {
		return fmt.Errorf("failed to setup db: %w", err)
	}

	shardStr := args[0]
	shard, err := strconv.ParseUint(shardStr, 0, 64)
	if err != nil {
		return fmt.Errorf("unable to determine shard value from: %s: %w", shardStr, err)
	}

	service := newService(kvdb, shard)

	checkpoint, err := service.GetShardCheckpoint(cmd.Context())
	if err != nil {
		return err
	}
	if checkpoint == nil {
		fmt.Printf("No checkpoint found for shard-%d\n", shard)
		return nil
	}
	fmt.Printf("Found checkpoint for for shard-%d\n", shard)
	fmt.Printf("  - Initial Start Block for shard: %d\n", checkpoint.InitialStartBlock)
	fmt.Printf("  - Target Stop Block for shard: %d\n", checkpoint.TargetStopBlock)
	fmt.Printf("  - Last Written Block Num for shard: %d\n", checkpoint.LastWrittenBlockNum)
	fmt.Printf("  - Last Written Block Id for shard: %s\n", checkpoint.LastWrittenBlockId)

	prompt := promptui.Prompt{
		Label:     "Are you sure you want to delete the checkpoint?",
		IsConfirm: true,
	}

	result, err := prompt.Run()
	if err != nil && err != promptui.ErrAbort {
		fmt.Printf("Error: \n")
	}

	if strings.ToLower(result) != "y" {
		fmt.Printf("aborting deletion. Goodbye!\n")
		return nil
	}

	err = service.DeleteCheckpoint(cmd.Context(), byte(shard))
	if err != nil {
		return fmt.Errorf("unable to delete checkpoint: %w", err)
	}

	fmt.Printf("checkpoint deleted successfully!\n")

	return nil
}

func readAccountE(cmd *cobra.Command, args []string) (err error) {
	kvdb, err := store.New(viper.GetString("dsn"))
	if err != nil {
		return fmt.Errorf("failed to setup db: %w", err)
	}
	kvdb = injector.NewRWCache(kvdb)

	service := newService(kvdb, 0)

	account := args[0]
	accountUint, err := eos.StringToName(account)
	if err != nil {
		return fmt.Errorf("unable to encode string %s to eos name (utin64): %w", account, err)
	}

	zlog.Info("retrieving shard summary for account",
		zap.String("account", account),
	)

	summary, err := service.KeySummary(cmd.Context(), accounthist.AccountKey(accountUint))
	if err != nil {
		return fmt.Errorf("unable to retrieve account summary: %w", err)
	}

	fmt.Printf("Account history summary for: %s\n", account)
	sum := uint64(0)
	for _, shardSummary := range summary {
		fmt.Printf("Shard %d: %d actions\n", shardSummary.ShardNum, shardSummary.SeqData.CurrentOrdinal)
		sum += shardSummary.SeqData.CurrentOrdinal
	}

	fmt.Printf("Total %d actions\n", sum)
	return nil

}

func scanAccountE(cmd *cobra.Command, args []string) (err error) {
	scanLimit := viper.GetInt("limit")
	kvdb, err := store.New(viper.GetString("dsn"))
	if err != nil {
		return fmt.Errorf("failed to setup db: %w", err)
	}
	kvdb = injector.NewRWCache(kvdb)

	fmt.Printf("Scanning accounts (limit: %d)\n", scanLimit)
	count := 0
	err = accounthist.ScanAccounts(cmd.Context(), kvdb, keyer.PrefixAccount, accounthist.AccountKeyRowDecoder, func(account uint64, shard byte, ordinalNum uint64) error {
		if count > scanLimit {
			return fmt.Errorf("scan limit reached")
		}
		fmt.Printf("Account:   %-12v   at shard: %d with last ordinal count: %d\n", eos.NameToString(account), int(shard), ordinalNum)
		count++
		return nil
	})
	if err != nil && err != fmt.Errorf("scan limit reached") {
		return err
	}
	return nil

}

func newService(kvdb store.KVStore, shardNum uint64) *injector.Injector {
	return injector.NewInjector(
		injector.NewRWCache(kvdb),
		nil,
		nil,
		byte(shardNum),
		1000,
		1,
		0,
		0,
		nil,
	)

}
