package tools

import (
	"fmt"

	"go.uber.org/zap"

	"github.com/dfuse-io/dfuse-eosio/accounthist"
	"github.com/dfuse-io/kvdb/store"
	"github.com/eoscanada/eos-go"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var accounthistCmd = &cobra.Command{Use: "accounthist", Short: "Read from accout history", RunE: dmeshE}
var accountReadCmd = &cobra.Command{Use: "account", Short: "Read an account", RunE: accountReadE, Args: cobra.ExactArgs(1)}
var accountScanCmd = &cobra.Command{Use: "scan", Short: "Scan accounts", RunE: accountScanE}

func init() {
	Cmd.AddCommand(accounthistCmd)
	accounthistCmd.AddCommand(accountReadCmd)
	accounthistCmd.AddCommand(accountScanCmd)

	accounthistCmd.PersistentFlags().String("dsn", "badger:///dfuse-data/kvdb/kvdb_badger.db", "KVStore DSN")
	accountScanCmd.Flags().Int("limit", 100, "limit the number of accounts when doing scan")

}

func accountReadE(cmd *cobra.Command, args []string) (err error) {
	kvdb, err := store.New(viper.GetString("dsn"))
	if err != nil {
		return fmt.Errorf("failed to setup db: %w", err)
	}
	kvdb = accounthist.NewRWCache(kvdb)
	service := accounthist.NewService(
		kvdb,
		nil,
		nil,
		0,
		1000,
		1,
		0,
		0,
		nil,
	)

	account := args[0]
	accountUint, err := eos.StringToName(account)
	if err != nil {
		return fmt.Errorf("unable to encode string %s to eos name (utin64): %w", account, err)
	}

	zlog.Info("retrieving shard summary for account",
		zap.String("account", account),
	)
	summary, err := service.ShardSummary(cmd.Context(), accountUint)
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

func accountScanE(cmd *cobra.Command, args []string) (err error) {
	scanLimit := viper.GetInt("limit")
	kvdb, err := store.New(viper.GetString("dsn"))
	if err != nil {
		return fmt.Errorf("failed to setup db: %w", err)
	}
	kvdb = accounthist.NewRWCache(kvdb)
	service := accounthist.NewService(
		kvdb,
		nil,
		nil,
		0,
		1000,
		1,
		0,
		0,
		nil,
	)

	fmt.Printf("Scanning accounts (limit: %d)\n", scanLimit)
	count := 0
	err = service.ScanAccounts(cmd.Context(), func(account uint64, shard byte, ordinalNum uint64) error {
		if count > scanLimit {
			return fmt.Errorf("scan limit reached")
		}
		fmt.Printf("Account:   %-12v   at shard: %d with action count: %d\n", eos.NameToString(account), int(shard), ordinalNum)
		count++
		return nil
	})
	if err != nil && err != fmt.Errorf("scan limit reached") {
		return err
	}
	return nil

}
