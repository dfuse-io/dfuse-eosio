package tokenmeta

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	pbstatedb "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/statedb/v1"
	pbtokenmeta "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/tokenmeta/v1"
	"github.com/dfuse-io/dfuse-eosio/tokenmeta/cache"
	"github.com/dfuse-io/dhammer"
	"github.com/eoscanada/eos-go"
	"go.uber.org/zap"
)

func isRetryableStateDBError(err error) bool {
	switch {
	case strings.Contains(err.Error(), "connection refused"):
		return true
	case strings.Contains(err.Error(), "no such host"):
		return true
	case strings.Contains(err.Error(), "dial tcp"):
		return true
	}
	return false
}

func getSymbolFromStateDB(ctx context.Context, stateClient pbstatedb.StateClient, account eos.AccountName, startBlockNum uint32) (out []eos.SymbolCode, err error) {
	zlog.Debug("getting symbols for contract from statedb",
		zap.String("token_contract", string(account)),
		zap.Uint32("start_block_num", startBlockNum),
	)

	scopes, err := pbstatedb.FetchTableScopes(ctx, stateClient, uint64(startBlockNum), string(account), "stat")
	if err != nil {
		return nil, fmt.Errorf("statedb reading scopes list: %w", err)
	}

	for _, s := range scopes {
		symCode, err := eos.NameToSymbolCode(eos.Name(s))
		if err != nil {
			zlog.Warn("stat scope to symbol list", zap.Error(err))
			// we should just skip this token
			continue
		}
		out = append(out, symCode)
	}
	return
}

var errDecodeAccountRow = errors.New("decode account row")

func getTokenBalancesFromStateDB(ctx context.Context, stateClient pbstatedb.StateClient, contract eos.AccountName, symbols []eos.SymbolCode, startBlockNum uint32) (out []*pbtokenmeta.AccountBalance, err error) {
	zlog.Debug("getting token balances for a token account from statedb", zap.String("token_contract", string(contract)), zap.Uint32("start_block_num", startBlockNum))
	tableScopes, err := pbstatedb.FetchTableScopes(ctx, stateClient, uint64(startBlockNum), string(contract), "accounts")
	if err != nil {
		zlog.Warn("cannot get table scope", zap.Error(err))
		return nil, err
	}

	ham := dhammer.NewHammer(1500, 2, func(ctx context.Context, inScopes []interface{}) ([]interface{}, error) {
		getBalancesReq := &pbstatedb.StreamMultiScopesTableRowsRequest{
			BlockNum: uint64(startBlockNum),
			Contract: string(contract),
			Table:    "accounts",
			KeyType:  "name",
			ToJson:   true,
		}

		getBalancesReq.Scopes = make([]string, len(inScopes))
		for i, scope := range inScopes {
			getBalancesReq.Scopes[i] = scope.(string)
		}

		var out []interface{}
		row := new(accountsDbRow)
		_, err := pbstatedb.ForEachMultiScopesTableRows(ctx, stateClient, getBalancesReq, func(scope string, response *pbstatedb.TableRowResponse) error {
			err = json.Unmarshal([]byte(response.Json), &row)
			if err != nil {
				zlog.Warn("unable to decode token contract account row",
					zap.String("contract", string(contract)),
					zap.String("table", "accounts"),
					zap.String("scope", scope),
				)
				return pbstatedb.SkipTable
			}

			if !row.valid() {
				zlog.Debug("token contract accounts row is not valid", zap.String("contract", string(contract)), zap.String("scope", scope))

				// FIXME: Elsewhere, we skip the table completely, but here, we skip the row, what is the correct behavior
				return nil
			}

			out = append(out, &pbtokenmeta.AccountBalance{
				TokenContract: string(contract),
				Account:       scope,
				Amount:        uint64(row.Balance.Amount),
				Symbol:        string(row.Balance.Symbol.Symbol),
				Precision:     uint32(row.Balance.Symbol.Precision),
			})

			return nil
		})

		if err != nil {
			return nil, fmt.Errorf("unable to stream multi scopes table rows: %w", err)
		}

		return out, nil
	})

	zlog.Info("starting dhammer", zap.String("contract", string(contract)), zap.Int("scope_count", len(tableScopes)))
	ham.Start(ctx)

	// scopes -> hammer
	go func() {
		defer ham.Close()
		for _, s := range tableScopes {
			select {
			case <-ctx.Done():
				return
			case ham.In <- s:
			}
		}
	}()

	// hammer -> tokenmeta
	for {
		select {
		case v, ok := <-ham.Out:
			if !ok {
				zlog.Info("get token balances finished", zap.String("contract", string(contract)), zap.Int("account_balance_count", len(out)), zap.Int("scope_count", len(tableScopes)))
				if ham.Err() != nil && ham.Err() != context.Canceled {
					zlog.Error("hammer error", zap.Error(ham.Err()))
					return nil, ham.Err()
				}
				return
			}
			out = append(out, v.(*pbtokenmeta.AccountBalance))
		}
	}
}

var errInvalidContractSymbol = errors.New("invalid contract symbol")

func getTokensFromStateDB(ctx context.Context, stateClient pbstatedb.StateClient, contract eos.AccountName, symbols []eos.SymbolCode, startBlockNum uint32) (out []*pbtokenmeta.Token, err error) {
	zlog.Debug("getting token symbol contract account from statedb", zap.String("token_contract", string(contract)))
	for _, symbol := range symbols {
		getTokensReq := &pbstatedb.StreamTableRowsRequest{
			BlockNum: uint64(startBlockNum),
			Contract: string(contract),
			Table:    "stat",
			Scope:    symbol.ToName(),
			KeyType:  "name",
			ToJson:   true,
		}

		row := new(statDbRow)
		_, err := pbstatedb.ForEachTableRows(ctx, stateClient, getTokensReq, func(response *pbstatedb.TableRowResponse) error {
			err = json.Unmarshal([]byte(response.Json), row)
			if err != nil {
				return fmt.Errorf("cannot decode token row table from statedb for contract %q and symbol %q: %w", string(contract), symbol.String(), err)
			}

			if !row.valid() {
				return errInvalidContractSymbol
			}

			out = append(out, &pbtokenmeta.Token{
				Contract:      string(contract),
				Symbol:        string(row.Supply.Symbol.Symbol),
				Precision:     uint32(row.Supply.Symbol.Precision),
				Issuer:        string(row.Issuer),
				MaximumSupply: uint64(row.MaxSupply.Amount),
				TotalSupply:   uint64(row.Supply.Amount),
			})
			return nil
		})

		if err != nil {
			if err == errInvalidContractSymbol {
				zlog.Debug("token contract symbol is not valid", zap.String("contract", string(contract)), zap.String("symbol", string(symbol)))
				continue
			}

			return nil, fmt.Errorf("cannot stream table from statedb for contract %q and symbol %q: %w", string(contract), symbol.String(), err)
		}
	}

	return out, nil
}

func getEOSStakedFromStateDB(ctx context.Context, stateClient pbstatedb.StateClient, startBlockNum uint32) (out []*cache.EOSStakeEntry, err error) {
	zlog.Debug("getting EOSStaked token", zap.Uint32("start_block_num", startBlockNum))

	tableScopes, err := pbstatedb.FetchTableScopes(ctx, stateClient, uint64(startBlockNum), "eosio", "delband")
	if err != nil {
		zlog.Warn("cannot get table scope", zap.String("account", "eosio"), zap.String("table", "delband"), zap.Error(err))
		return nil, err
	}

	ham := dhammer.NewHammer(20, 3, func(ctx context.Context, inScopes []interface{}) ([]interface{}, error) {
		//zlog.Debug("batching scope stakes for delband", zap.Int("len", len(inScopes)))
		getBalancesReq := &pbstatedb.StreamMultiScopesTableRowsRequest{
			BlockNum: uint64(startBlockNum),
			Contract: "eosio",
			Table:    "delband",
			KeyType:  "name",
			ToJson:   true,
		}

		getBalancesReq.Scopes = make([]string, len(inScopes))
		for i, scope := range inScopes {
			getBalancesReq.Scopes[i] = scope.(string)
		}

		var out []interface{}
		row := new(EOSStakeDbRow)
		_, err := pbstatedb.ForEachMultiScopesTableRows(ctx, stateClient, getBalancesReq, func(scope string, response *pbstatedb.TableRowResponse) error {
			err = json.Unmarshal([]byte(response.Json), &row)
			if err != nil {
				zlog.Warn("unable to decode stake rows",
					zap.String("contract", "eosio"),
					zap.String("table", "delband"),
					zap.String("scope", scope),
				)
				return pbstatedb.SkipTable
			}

			if !row.valid() {
				zlog.Debug("stake row is not valid", zap.String("scope", scope))
				// FIXME: Elsewhere, we skip the table completely, but here, we skip the row, what is the correct behavior
				return nil
			}

			out = append(out, &cache.EOSStakeEntry{
				From: row.From,
				To:   row.To,
				Net:  row.NetWeight.Amount,
				Cpu:  row.CPUWeight.Amount,
			})

			return nil
		})

		if err != nil {
			return nil, fmt.Errorf("unable to stream multi scopes table rows: %w", err)
		}

		return out, nil
	})

	zlog.Info("starting dhammer", zap.String("account", "eosio"), zap.String("table", "delband"), zap.Int("scope_count", len(tableScopes)))
	ham.Start(ctx)

	// scopes -> hammer
	go func() {
		for _, s := range tableScopes {
			// TODO had a way for context to cancel ?
			ham.In <- s
		}
		ham.Close()
	}()

	// hammer -> tokenmeta
	for {
		select {
		case v, ok := <-ham.Out:
			if !ok {
				zlog.Info("get eos stakes finished", zap.String("account", "eosio"), zap.String("table", "delband"), zap.Int("stakes", len(out)), zap.Int("scope_count", len(tableScopes)))
				if ham.Err() != nil && ham.Err() != context.Canceled {
					zlog.Error("hammer error", zap.Error(ham.Err()))
					return nil, ham.Err()
				}
				return
			}
			out = append(out, v.(*cache.EOSStakeEntry))
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
}
