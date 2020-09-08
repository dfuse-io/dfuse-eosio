package tokenmeta

import (
	"context"
	"encoding/json"
	"fmt"

	pbstatedb "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/statedb/v1"
	pbtokenmeta "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/tokenmeta/v1"
	"github.com/eoscanada/eos-go"
	"go.uber.org/zap"
)

type contractStats struct {
	hasAccountsTable bool
	hasStatsTable    bool
	isTokenContract  bool
}

func getTokenContractStats(account string, rawABI []byte, isJsonEncoded bool) (*contractStats, error) {
	var abi *eos.ABI
	var err error
	if isJsonEncoded {
		err = json.Unmarshal(rawABI, &abi)
	} else {
		err = eos.UnmarshalBinary(rawABI, &abi)
	}

	if err != nil {
		return nil, fmt.Errorf("failed decoding ABI in account %q: %w", account, err)
	}

	var hasStat, hasAccounts bool
	var statStruct, accountStruct string
	for _, tbl := range abi.Tables {
		if tbl.Name == "stat" {
			statStruct = tbl.Type
			hasStat = true
		} else if tbl.Name == "accounts" {
			accountStruct = tbl.Type
			hasAccounts = true
		}
	}

	if !hasStat || !hasAccounts {
		zlog.Debug("contract does not have either stats or accounts table not a token contract",
			zap.String("account", account),
		)
		return &contractStats{
			hasAccountsTable: hasAccounts,
			hasStatsTable:    hasStat,
		}, nil
	}

	var hasStatFields, hasAccountFields bool
	for _, s := range abi.Structs {
		if s.Name == accountStruct {
			if len(s.Fields) != 0 && s.Fields[0].Type == "asset" {
				hasAccountFields = true
			}
		}
		if s.Name == statStruct && len(s.Fields) > 2 {
			if s.Fields[0].Name == "supply" &&
				s.Fields[1].Name == "max_supply" &&
				s.Fields[2].Name == "issuer" {
				hasStatFields = true
			} else {
				zlog.Debug("stat failed for", zap.String("account", account))
			}
		}
	}

	return &contractStats{
		hasAccountsTable: hasAccounts,
		hasStatsTable:    hasStat,
		isTokenContract:  (hasStatFields && hasAccountFields),
	}, nil

}

func processContract(ctx context.Context, tokenContract eos.AccountName, startBlockNum uint32, stateClient pbstatedb.StateClient) (tokens []*pbtokenmeta.Token, balances []*pbtokenmeta.AccountBalance, err error) {
	var symcodes []eos.SymbolCode
	symcodes, err = getSymbolFromStateDB(ctx, stateClient, tokenContract, startBlockNum)
	if err != nil {
		return
	}

	if len(symcodes) == 0 {
		// skip this contract no symbol was found
		return
	}

	tokens, err = getTokensFromStateDB(ctx, stateClient, tokenContract, symcodes, startBlockNum)
	if err != nil {
		return
	}

	balances, err = getTokenBalancesFromStateDB(ctx, stateClient, tokenContract, startBlockNum)
	return
}
