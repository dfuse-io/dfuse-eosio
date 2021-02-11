package tools

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/eoscanada/eos-go"

	"github.com/spf13/viper"

	"github.com/dfuse-io/dfuse-eosio/tokenmeta/cache"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var tokenmetaCmd = &cobra.Command{Use: "tokenmeta", Short: "Tokenmeta helper functions"}
var toJsonCmd = &cobra.Command{
	Use:   "to_json",
	Short: "Outputs gob file in json",
	Args:  cobra.ExactArgs(0),
	RunE:  toJSONE,
}

var findCmd = &cobra.Command{Use: "find", Short: "Find accounts and contracts"}
var findAccountCmd = &cobra.Command{
	Use:   "account {owner}",
	Short: "Returns assets owned by given account",
	Args:  cobra.ExactArgs(1),
	RunE:  findAccountE,
}

var findContractCmd = &cobra.Command{
	Use:   "contract {contract}",
	Short: "Returns contract's asset holders",
	Args:  cobra.ExactArgs(1),
	RunE:  findContractE,
}

var findAccountContractCmd = &cobra.Command{
	Use:   "account-contract {owner} {contract}",
	Short: "Returns assets owned by a given account for a given contract",
	Args:  cobra.ExactArgs(2),
	RunE:  findAccountContractE,
}

func init() {
	Cmd.AddCommand(tokenmetaCmd)

	tokenmetaCmd.AddCommand(toJsonCmd)
	tokenmetaCmd.AddCommand(findCmd)
	findCmd.AddCommand(findAccountCmd)
	findCmd.AddCommand(findContractCmd)
	findCmd.AddCommand(findAccountContractCmd)

	tokenmetaCmd.PersistentFlags().String("tokenmeta-cache", "/data/token-cache-v1.gob", "Path to tokenmeta cache GOB file")
}

func findAccountE(cmd *cobra.Command, args []string) (err error) {
	account := eos.AccountName(args[0])
	tokenCache, err := loadgob()
	if err != nil {
		return fmt.Errorf("unable to load tokenmeta cache GOB file: %w", err)
	}

	assets := tokenCache.AccountBalances(account, cache.EOSIncludeStakedAccOpt)
	fmt.Printf("%q's assets:\n", account)
	for _, a := range assets {
		fmt.Printf("contract %q -> %s\n", a.Asset.Contract, a.Asset.Asset.String())
	}

	return nil
}

func findContractE(cmd *cobra.Command, args []string) (err error) {
	contract := eos.AccountName(args[0])
	tokenCache, err := loadgob()
	if err != nil {
		return fmt.Errorf("unable to load tokenmeta cache GOB file: %w", err)
	}
	assets := tokenCache.TokenBalances(contract, cache.EOSIncludeStakedTokOpt)
	fmt.Printf("Assets managed by contract %q:\n", contract)
	for _, a := range assets {
		fmt.Printf("owner %q -> %s\n", a.Owner, a.Asset.Asset.String())
	}
	return nil
}

func findAccountContractE(cmd *cobra.Command, args []string) (err error) {
	account := eos.AccountName(args[0])
	contract := eos.AccountName(args[1])
	tokenCache, err := loadgob()
	if err != nil {
		return fmt.Errorf("unable to load tokenmeta cache GOB file: %w", err)
	}
	assets := tokenCache.TokenBalances(contract, cache.EOSIncludeStakedTokOpt)

	fmt.Printf("%q's assets in contract %q:\n", account, contract)
	for _, a := range assets {
		if a.Owner == account {
			fmt.Printf("%s\n", a.Asset.Asset.String())
		}
	}

	return nil
}

func loadgob() (*cache.DefaultCache, error) {
	filename := viper.GetString("tokenmeta-cache")
	zlog.Info("trying to load from token cache file",
		zap.String("filename", filename),
	)
	fmt.Printf("Loading tokenmeta cache file %q (...this may take a minute or two)\n", filename)
	return cache.LoadDefaultCacheFromFile(filename)
}

func toJSONE(cmd *cobra.Command, args []string) (err error) {
	tokenCache, err := loadgob()
	if err != nil {
		return fmt.Errorf("unable to load tokenmeta cache GOB file: %w", err)
	}

	cnt, err := json.Marshal(tokenCache)
	if err != nil {
		zlog.Error("unable to JSON marshall token cache content", zap.Error(err))
		return fmt.Errorf("unable to JSON marshall token cache content : %w", err)
	}

	fmt.Println(string(cnt))
	return nil
}

func isNotExits(err error) bool {
	for {
		if os.IsNotExist(err) {
			return true
		}

		err = errors.Unwrap(err)
		if err == nil {
			return false
		}
	}
}
