package tokenmeta

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/dfuse-io/dfuse-eosio/fluxdb-client"
	pbtokenmeta "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/tokenmeta/v1"
	"github.com/dfuse-io/dfuse-eosio/tokenmeta/cache"
	"github.com/dfuse-io/dhammer"
	"github.com/eoscanada/eos-go"

	"go.uber.org/zap"
)

type fluxStakeRows []*fluxStakeRow

type fluxStakeRow struct {
	Key   string          `json:"key"`
	Payer eos.AccountName `json:"payer"`
	JSON  EOSStakeDbRow   `json:"json"`
}

type fluxBalanceRows []*fluxBalanceRow

type fluxBalanceRow struct {
	Key   string          `json:"key"`
	Payer eos.AccountName `json:"payer"`
	JSON  accountsDbRow   `json:"json"`
}

type fluxTokensResp []fluxTokenResp

type fluxTokenResp struct {
	Key   string          `json:"key"`
	Payer eos.AccountName `json:"payer"`
	JSON  statDbRow       `json:"json"`
}

func isRetryableFluxError(err error) bool {
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

func getSymbolFromFlux(ctx context.Context, fluxClient fluxdb.Client, account eos.AccountName, startBlockNum uint32) (out []eos.SymbolCode, err error) {
	zlog.Debug("getting symbols for contract from flux",
		zap.String("token_contract", string(account)),
		zap.Uint32("start_block_num", startBlockNum),
	)

	scopesReq := &fluxdb.GetTableScopesRequest{
		Account: account,
		Table:   eos.TableName("stat"),
	}

	scopesResp, err := fluxClient.GetTableScopes(ctx, startBlockNum, scopesReq)
	if err != nil {
		return nil, fmt.Errorf("flux reading scopes list: %w", err)
	}

	for _, s := range scopesResp.Scopes {
		symCode, err := eos.NameToSymbolCode(s)
		if err != nil {
			zlog.Warn("stat scope to symbol list", zap.Error(err))
			// we should just skip this token
			continue
		}
		out = append(out, symCode)
	}
	return
}

func getTokenBalancesFromFlux(ctx context.Context, fluxClient fluxdb.Client, contract eos.AccountName, symbols []eos.SymbolCode, startBlockNum uint32) (out []*pbtokenmeta.AccountBalance, err error) {
	zlog.Debug("getting token balances for a token account from flux", zap.String("token_contract", string(contract)), zap.Uint32("start_block_num", startBlockNum))

	scopesReq := &fluxdb.GetTableScopesRequest{
		Account: eos.AccountName(contract),
		Table:   eos.TableName("accounts"),
	}
	scopesResp, err := fluxClient.GetTableScopes(ctx, startBlockNum, scopesReq)
	if err != nil {
		zlog.Warn("cannot get table scope", zap.Error(err))
		return nil, err
	}

	ham := dhammer.NewHammer(1500, 2, func(ctx context.Context, inScopes []interface{}) ([]interface{}, error) {
		getBalancesReq := &fluxdb.GetTablesMultiScopesRequest{
			Account: contract,
			Table:   eos.TableName("accounts"),
			KeyType: "name",
			JSON:    true,
		}

		for _, s := range inScopes {
			getBalancesReq.Scopes = append(getBalancesReq.Scopes, s.(eos.Name))
		}

		var balancesResp *fluxdb.GetTablesMultiScopesResponse
		balancesResp, err = fluxClient.GetTablesMultiScopes(ctx, startBlockNum, getBalancesReq)
		if err != nil {
			return nil, err
		}

		var out []interface{}

		for _, table := range balancesResp.Tables {
			decodedRows := fluxBalanceRows{}

			err = json.Unmarshal(table.Rows, &decodedRows)
			if err != nil {
				// table row in
				zlog.Warn("unable to decode token contract account row",
					zap.String("contract", string(contract)),
					zap.String("table", "accounts"),
					zap.String("scope", table.Scope), zap.String("rows", string(table.Rows)))
				continue
			}

			for _, row := range decodedRows {

				if !row.JSON.valid() {
					zlog.Debug("token contract accounts row is not valid", zap.String("contract", string(contract)), zap.String("scope", string(table.Scope)))
					continue
				}

				out = append(out, &pbtokenmeta.AccountBalance{
					TokenContract: string(contract),
					Account:       string(table.Scope),
					Amount:        uint64(row.JSON.Balance.Amount),
					Symbol:        string(row.JSON.Balance.Symbol.Symbol),
					Precision:     uint32(row.JSON.Balance.Symbol.Precision),
				})
			}
		}
		return out, nil
	})

	zlog.Info("starting dhammer", zap.String("contract", string(contract)), zap.Int("scope_count", len(scopesResp.Scopes)))
	ham.Start(ctx)

	// scopes -> hammer
	go func() {
		defer ham.Close()
		for _, s := range scopesResp.Scopes {
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
				zlog.Info("get token balances finished", zap.String("contract", string(contract)), zap.Int("account_balance_count", len(out)), zap.Int("scope_count", len(scopesResp.Scopes)))
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

func getTokensFromFlux(ctx context.Context, fluxClient fluxdb.Client, contract eos.AccountName, symbols []eos.SymbolCode, startBlockNum uint32) (out []*pbtokenmeta.Token, err error) {
	zlog.Debug("getting token symbol contract account from flux", zap.String("token_contract", string(contract)))
	for _, symbol := range symbols {
		getTokensReq := fluxdb.NewGetTableRequest(contract, eos.Name(symbol.ToName()), eos.TableName("stat"), "name")
		tokensResponse, err := fluxClient.GetTable(ctx, startBlockNum, getTokensReq)
		if err != nil {
			return nil, fmt.Errorf("cannot get table from flux for contract %q and symbol %q: %w", string(contract), symbol.String(), err)
		}
		decodedTokensResp := fluxTokensResp{}
		err = json.Unmarshal(tokensResponse.Rows, &decodedTokensResp)
		if err != nil {
			return nil, fmt.Errorf("cannot decode token row table from flux for contract %q and symbol %q: %w", string(contract), symbol.String(), err)
		}

		for _, tok := range decodedTokensResp {

			if !tok.JSON.valid() {
				zlog.Debug("token contract symbol is not valid", zap.String("contract", string(contract)), zap.String("symbol", string(symbol)))
				continue
			}

			out = append(out, &pbtokenmeta.Token{
				Contract:      string(contract),
				Symbol:        string(tok.JSON.Supply.Symbol.Symbol),
				Precision:     uint32(tok.JSON.Supply.Symbol.Precision),
				Issuer:        string(tok.JSON.Issuer),
				MaximumSupply: uint64(tok.JSON.MaxSupply.Amount),
				TotalSupply:   uint64(tok.JSON.Supply.Amount),
			})
		}
	}
	return
}

func getEOSStakedFromFlux(ctx context.Context, fluxClient fluxdb.Client, startBlockNum uint32) (out []*cache.EOSStakeEntry, err error) {
	zlog.Debug("getting EOSStaked token", zap.Uint32("start_block_num", startBlockNum))

	scopesReq := &fluxdb.GetTableScopesRequest{
		Account: eos.AccountName("eosio"),
		Table:   eos.TableName("delband"),
	}
	scopesResp, err := fluxClient.GetTableScopes(ctx, startBlockNum, scopesReq)
	if err != nil {
		zlog.Warn("cannot get table scope",
			zap.String("account", "eosio"),
			zap.String("table", "delband"),
			zap.Error(err))
		return nil, err
	}

	ham := dhammer.NewHammer(20, 3, func(ctx context.Context, inScopes []interface{}) ([]interface{}, error) {
		//zlog.Debug("batching scope stakes for delband", zap.Int("len", len(inScopes)))
		getBalancesReq := &fluxdb.GetTablesMultiScopesRequest{
			Account: eos.AccountName("eosio"),
			Table:   eos.TableName("delband"),
			KeyType: "name",
			JSON:    true,
		}

		for _, s := range inScopes {
			getBalancesReq.Scopes = append(getBalancesReq.Scopes, s.(eos.Name))
		}

		var balancesResp *fluxdb.GetTablesMultiScopesResponse
		balancesResp, err = fluxClient.GetTablesMultiScopes(ctx, startBlockNum, getBalancesReq)
		if err != nil {
			return nil, err
		}

		var out []interface{}

		for _, table := range balancesResp.Tables {
			decodedRows := fluxStakeRows{}

			err = json.Unmarshal(table.Rows, &decodedRows)
			if err != nil {
				// table row in
				zlog.Warn("unable to decode stake rows",
					zap.String("account", "eosio"),
					zap.String("table", "delband"),
					zap.String("scope", table.Scope), zap.String("rows", string(table.Rows)))
				continue
			}

			for _, row := range decodedRows {

				if !row.JSON.valid() {
					zlog.Debug("stake row is not valid", zap.String("scope", string(table.Scope)))
					continue
				}

				out = append(out, &cache.EOSStakeEntry{
					From: row.JSON.From,
					To:   row.JSON.To,
					Net:  row.JSON.NetWeight.Amount,
					Cpu:  row.JSON.CPUWeight.Amount,
				})
			}
		}
		return out, nil
	})

	zlog.Info("starting dhammer", zap.String("account", "eosio"), zap.String("table", "delband"), zap.Int("scope_count", len(scopesResp.Scopes)))
	ham.Start(ctx)

	// scopes -> hammer
	go func() {
		for _, s := range scopesResp.Scopes {
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
				zlog.Info("get eos stakes finished", zap.String("account", "eosio"), zap.String("table", "delband"), zap.Int("stakes", len(out)), zap.Int("scope_count", len(scopesResp.Scopes)))
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
