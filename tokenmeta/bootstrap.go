package tokenmeta

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/dfuse-io/bstream"
	pbstatedb "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/statedb/v1"
	pbtokenmeta "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/tokenmeta/v1"
	"github.com/dfuse-io/dfuse-eosio/tokenmeta/cache"
	"github.com/eoscanada/eos-go"
	"github.com/tidwall/gjson"
	"go.uber.org/zap"
)

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

	balances, err = getTokenBalancesFromStateDB(ctx, stateClient, tokenContract, symcodes, startBlockNum)
	return
}

func Bootstrap(abisFileContent []byte, stateClient pbstatedb.StateClient, bootstrapblockOffset uint64) (tokens []*pbtokenmeta.Token, balances []*pbtokenmeta.AccountBalance, stakeds []*cache.EOSStakeEntry, startBlock bstream.BlockRef, err error) {

	startBlock = bstream.NewBlockRef("", 0)
	tokenContracts := parseContractFromABIs(abisFileContent)
	abiStartBlock, err := parseCursorFromABIs(abisFileContent)
	if err != nil {
		zlog.Warn("parsing cursor from ABIs Cache", zap.Error(err))
	} else {
		// the cursor located in the ABIs cached (exported by abicodec) is following the live, thus there is no guaranty that it is
		// irreversible. Tokenmeta start with a forkable which only processes irreversible blocks, so to avoid
		// stalling forkable (starting it off with an ID within a fork block) we disregard the block id from abi codec cursor.
		// Furthermore, we added a bootstrap block offeset to ensure when you are querying StateDB you do not hit reversible blocks
		startBlock = bstream.NewBlockRef("", (abiStartBlock.Num() - bootstrapblockOffset))
	}

	ctx := context.Background()

	sta, err := getEOSStakedFromStateDB(ctx, stateClient, uint32(startBlock.Num()))
	stakeds = append(stakeds, sta...)

	zlog.Info("looping through valid contracts", zap.Uint64("start_block_num", startBlock.Num()), zap.String("start_block_id", startBlock.ID()), zap.Int("valid_contracts_count", len(tokenContracts)))

	for _, tokenContract := range tokenContracts {
		for attempt := 1; true; attempt++ {
			toks, bals, err := processContract(ctx, tokenContract, uint32(startBlock.Num()), stateClient)
			if err == nil {
				if toks == nil {
					zlog.Info("skipping empty token contract",
						zap.String("token_contract", string(tokenContract)),
					)
					break
				}
				tokens = append(tokens, toks...)
				balances = append(balances, bals...)
				break
			}
			if !isRetryableStateDBError(err) {
				zlog.Info("skipping invalid token contract, unable to get symbols with non-retryable error",
					zap.String("token_contract", string(tokenContract)),
					zap.Error(err),
				)
				break
			}
			if attempt > 5 {
				return nil, nil, nil, nil, fmt.Errorf("failing after 5 attempts to get symbols from token contract: %w", err)
			}
			zlog.Warn("unable to get symbols from token contract, retrying", zap.String("token_contract", string(tokenContract)), zap.Error(err))
			time.Sleep(time.Duration(attempt) * time.Second)

		}

	}

	return tokens, balances, stakeds, startBlock, nil
}

func parseContractFromABIs(cnt []byte) (out []eos.AccountName) {
	var accounts, withTableAccounts, withTableStat, fullBlown int
	gjson.GetBytes(cnt, "Abis").ForEach(func(k, v gjson.Result) bool {
		accounts++
		account := k.String()

		var lastABI gjson.Result
		v.ForEach(func(k, v gjson.Result) bool {
			lastABI = v
			return true
		})

		var abi *eos.ABI
		rawABI := lastABI.Get("ABI").Raw
		if rawABI == "" {
			zlog.Info("skipping missing ABI in account, probably normal", zap.String("account", account))
			return true
		}
		err := json.Unmarshal([]byte(rawABI), &abi)
		if err != nil {
			zlog.Warn("failed decoding ABI in account", zap.String("account", account), zap.Error(err), zap.String("raw_abi", rawABI))
			return false
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

		if hasStat {
			withTableStat++
		}
		if hasAccounts {
			withTableAccounts++
		}
		if !hasStat || !hasAccounts {
			return true
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

		if hasStatFields && hasAccountFields {
			out = append(out, eos.AccountName(account))
			fullBlown++
		}
		return true
	})
	zlog.Info("ABIS content stats",
		zap.Int("accounts_count", accounts),
		zap.Int("accounts_with_accounts_table", withTableAccounts),
		zap.Int("accounts_with_stat_table", withTableStat),
		zap.Int("accounts_with_stat_table_and_accounts_table", fullBlown))
	return
}

func parseCursorFromABIs(cnt []byte) (bstream.BlockRef, error) {
	cursor := gjson.GetBytes(cnt, "cursor").String()
	if cursor == "" {
		return nil, fmt.Errorf("cursor expected in ABIs cached file")
	} else {
		blockNum, headBlockId, _, err := parseCursor(cursor)
		if err != nil {
			return nil, fmt.Errorf("unable to parse cursor %q in ABIs cached file: %w", cursor, err)
		} else {
			return bstream.NewBlockRef(headBlockId, blockNum), nil
		}
	}
}
