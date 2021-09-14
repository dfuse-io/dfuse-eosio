package tools

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/dfuse-io/dfuse-eosio/accounthist/purger"

	"github.com/dfuse-io/dfuse-eosio/accounthist/keyer"

	"github.com/dfuse-io/dfuse-eosio/accounthist"

	"github.com/dfuse-io/dfuse-eosio/accounthist/injector"

	"github.com/manifoldco/promptui"
	"go.uber.org/zap"

	"github.com/eoscanada/eos-go"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/streamingfast/kvdb/store"
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

// dfuseeos tools accounthist account purge {account} --dsn
var purgeAccountCmd = &cobra.Command{
	Use:   "purge {maxEntries}",
	Short: "Purge accounts",
	Args:  cobra.ExactArgs(1),
	RunE:  purgeAccountE,
}

// dfuseeos tools accounthist account scan --dsn
var scanAccountsCmd = &cobra.Command{
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
	accountCmd.AddCommand(readAccountCmd, scanAccountsCmd, purgeAccountCmd)

	accounthistCmd.AddCommand(checkpointCmd)
	checkpointCmd.AddCommand(readCheckpointCmd, deleteCheckpointCmd)

	accounthistCmd.PersistentFlags().String("mode", "account", "accountgist mode one of 'account' or 'account-contract'")
	accounthistCmd.PersistentFlags().String("dsn", "badger:///dfuse-data/kvdb/kvdb_badger.db", "kvStore DSN")
	readAccountCmd.Flags().Int("shardNum", -1, "Analyze at a specific shard number")
	scanAccountsCmd.Flags().Int("limit", 100, "limit the number of accounts when doing scan")

	purgeAccountCmd.Flags().Bool("run", false, "Run purger in non-dyr run mode")
}

func readCheckpointE(cmd *cobra.Command, args []string) (err error) {
	kvdb, mode, err := getKVDBAndMode()
	if err != nil {
		return err
	}

	shardStr := args[0]
	shard, err := strconv.ParseUint(shardStr, 0, 64)
	if err != nil {
		return fmt.Errorf("unable to determine shard value from: %s: %w", shardStr, err)
	}

	service := setupService(kvdb, shard, mode)
	checkpoint, err := service.GetShardCheckpoint(cmd.Context())
	if err != nil {
		return err
	}
	if checkpoint == nil {
		fmt.Printf("No checkpoint found for shard: %d\n", shard)
		return nil
	}

	fmt.Printf("Checkpoint for shard: %d\n", shard)
	fmt.Printf("  - Initial Start Block for shard: %d\n", checkpoint.InitialStartBlock)
	fmt.Printf("  - Target Stop Block for shard: %d\n", checkpoint.TargetStopBlock)
	fmt.Printf("  - Last Written Block Num for shard: %d\n", checkpoint.LastWrittenBlockNum)
	fmt.Printf("  - Last Written Block Id for shard: %s\n", checkpoint.LastWrittenBlockId)

	return nil
}

func deleteCheckpointE(cmd *cobra.Command, args []string) (err error) {
	kvdb, mode, err := getKVDBAndMode()
	if err != nil {
		return err
	}

	shardStr := args[0]
	shard, err := strconv.ParseUint(shardStr, 0, 64)
	if err != nil {
		return fmt.Errorf("unable to determine shard value from: %s: %w", shardStr, err)
	}

	service := setupService(kvdb, shard, mode)
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
	kvdb, mode, err := getKVDBAndMode()
	if err != nil {
		return err
	}

	account := args[0]
	accountUint, err := eos.StringToName(account)
	if err != nil {
		return fmt.Errorf("unable to encode string %s to eos name (utin64): %w", account, err)
	}

	service := setupService(kvdb, 0, mode)

	zlog.Info("retrieving shard summary for account",
		zap.String("account", account),
	)

	shardNum := viper.GetInt("shardNum")
	if shardNum >= 0 {
		summary, err := service.FacetShardSummary(cmd.Context(), accounthist.AccountFacet(accountUint), byte(shardNum))
		if err != nil {
			return fmt.Errorf("unable to retrieve account shard summary: %w", err)
		}

		fmt.Printf("Account %s summary at shard %d\n", account, shardNum)
		fmt.Printf("Latest Seq Data:\n")
		fmt.Printf("  - Current Ordinal Number: %d\n", summary.LatestSeqData.CurrentOrdinal)
		fmt.Printf("  - Last Global Seq: %d\n", summary.LatestSeqData.LastGlobalSeq)
		fmt.Printf("  - Last Deleted Ordinal shard: %d\n", summary.LatestSeqData.LastDeletedOrdinal)
		fmt.Printf("Total Facet keys found in shard: %d:\n", summary.RowKeyCount)
		return nil
	}

	summary, err := service.FacetShardsSummary(cmd.Context(), accounthist.AccountFacet(accountUint))
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

func purgeAccountE(cmd *cobra.Command, args []string) (err error) {
	kvdb, mode, err := getKVDBAndMode()
	if err != nil {
		return err
	}

	runMode := viper.GetBool("run")

	maxEntriesStr := args[0]
	maxEntries, err := strconv.ParseUint(maxEntriesStr, 10, 64)
	if err != nil {
		return fmt.Errorf("unable to parse max entry value string %s: %w", maxEntriesStr, err)
	}

	var facatoryAsset accounthist.FacetFactory
	switch mode {
	case accounthist.AccounthistModeAccount:
		facatoryAsset = &accounthist.AccountFactory{}
	case accounthist.AccounthistModeAccountContract:
		facatoryAsset = &accounthist.AccountContractFactory{}
	}

	p := purger.NewPurger(kvdb, facatoryAsset, !runMode)

	if runMode {
		fmt.Println("Purging accounts")
	} else {
		fmt.Println("Purging accounts -- DRY RUN")
	}
	p.PurgeAccounts(cmd.Context(), maxEntries, func(facet accounthist.Facet, belowShardNum int, currentCount uint64) {
		fmt.Println(fmt.Sprintf("Purging facet %s below shard %d current seen action count %d", facet.String(), belowShardNum, currentCount))
	})
	return nil
}

func scanAccountE(cmd *cobra.Command, args []string) (err error) {
	scanLimit := viper.GetInt("limit")
	kvdb, mode, err := getKVDBAndMode()
	if err != nil {
		return err
	}
	kvdb = injector.NewRWCache(kvdb)

	var prefix byte
	var facetFactory accounthist.FacetFactory
	switch mode {
	case accounthist.AccounthistModeAccount:
		prefix = keyer.PrefixAccount
		facetFactory = &accounthist.AccountFactory{}
	case accounthist.AccounthistModeAccountContract:
		prefix = keyer.PrefixAccountContract
		facetFactory = &accounthist.AccountContractFactory{}
	}

	fmt.Printf("Scanning accounts (limit: %d)\n", scanLimit)
	count := 0
	err = accounthist.ScanFacets(cmd.Context(), kvdb, prefix, facetFactory.DecodeRow, func(facet accounthist.Facet, shard byte, ordinalNum uint64) error {
		if count > scanLimit {
			return fmt.Errorf("scan limit reached")
		}
		fmt.Printf("Facet: %s at shard: %d with last ordinal count: %d\n", facet.String(), int(shard), ordinalNum)
		count++
		return nil
	})
	if err != nil && err != fmt.Errorf("scan limit reached") {
		return err
	}
	return nil

}

func setupService(kvdb store.KVStore, shardNum uint64, mode accounthist.AccounthistMode) *injector.Injector {
	i := injector.NewInjector(
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

	switch mode {
	case accounthist.AccounthistModeAccount:
		i.SetFacetFactory(&accounthist.AccountFactory{})
	case accounthist.AccounthistModeAccountContract:
		i.SetFacetFactory(&accounthist.AccountContractFactory{})
	}
	return i
}

func getKVDBAndMode() (store.KVStore, accounthist.AccounthistMode, error) {
	kvdb, err := store.New(viper.GetString("dsn"))
	if err != nil {
		return nil, "", fmt.Errorf("failed to setup db: %w", err)
	}

	switch accounthist.AccounthistMode(viper.GetString("mode")) {
	case accounthist.AccounthistModeAccount:
		return kvdb, accounthist.AccounthistModeAccount, nil
	case accounthist.AccounthistModeAccountContract:
		return kvdb, accounthist.AccounthistModeAccountContract, nil
	default:
		return nil, "", fmt.Errorf("unknown acounthist mode: %s", viper.GetString("mode"))

	}
}
