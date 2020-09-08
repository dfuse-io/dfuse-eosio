package tokenmeta

import (
	"context"

	"github.com/dfuse-io/bstream"
	pbcodec "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/codec/v1"
	"github.com/dfuse-io/dfuse-eosio/tokenmeta/cache"
	"github.com/eoscanada/eos-go"
	"go.uber.org/zap"
)

func (t *TokenMeta) ProcessBlock(block *bstream.Block, obj interface{}) error {
	// forkable setup will only yield irreversible blocks
	muts := &cache.MutationsBatch{}
	blk := block.ToNative().(*pbcodec.Block)

	if (blk.Number % 600) == 0 {
		zlog.Info("process blk 1/600", zap.Stringer("block", block))
	}

	for _, trx := range blk.TransactionTraces() {
		zlogger := zlog.With(zap.String("trx_id", trx.Id))
		actionMatcher := blk.FilteringActionMatcher(trx)

		for _, actTrace := range trx.ActionTraces {
			if !shouldProcessAction(actTrace, actionMatcher) {
				continue
			}
			actionName := actTrace.Action.Name
			account := eos.AccountName(actTrace.Account())
			zlog.Debug("processing action trace",
				zap.String("action", actionName),
				zap.String("account", string(account)),
			)

			if actTrace.Action.Name == "setabi" {
				rawABI := actTrace.Action.RawData
				contractStats, err := getTokenContractStats(string(account), rawABI)
				if err != nil {
					zlog.Warn("failed decoding ABI in account",
						zap.String("account", string(account)),
						zap.String("raw_abi", string(rawABI)),
						zap.Error(err),
					)
					// TODO: what do we do here?
					return false
				}

				if contractStats.isTokenContract {
					err = t.addNewTokenContract(context.Background(), account, blk)
					zlog.Info("discovered new token contract",
						zap.String("account", string(account)),
					)
				}
			}
		}

		for _, dbop := range trx.DbOps {
			if !shouldProcessDbop(dbop, actionMatcher) {
				continue
			}
			zlog.Debug("processing dbop",
				zap.String("contract", dbop.Code),
				zap.String("table", dbop.TableName),
				zap.String("scope", dbop.Scope),
				zap.String("primary_key", dbop.PrimaryKey),
			)

			isEOSStake := dbop.Code == "eosio" && dbop.TableName == string(EOSStakeTable)

			tokenContract := eos.AccountName(dbop.Code)
			if !t.cache.IsTokenContract(tokenContract) && !isEOSStake {
				continue
			}

			symbolCode, err := eos.NameToSymbolCode(eos.Name(dbop.PrimaryKey))
			if err != nil {
				zlogger.Warn("unable to decode primary key to symbol",
					zap.String("contract", string(tokenContract)),
					zap.String("table", dbop.TableName),
					zap.String("scope", dbop.Scope),
					zap.String("primary_key", dbop.PrimaryKey),
					zap.Error(err))
				continue
			}

			rowData := dbop.NewData
			if rowData == nil {
				zlog.Info("using db row old data")
				rowData = dbop.OldData
			}
			row, err := t.decodeDBOpToRow(rowData, eos.TableName(dbop.TableName), tokenContract, uint32(block.Number))
			if err != nil {
				zlogger.Error("cannot decode table row",
					zap.String("contract", string(tokenContract)),
					zap.String("table_name", dbop.TableName),
					zap.String("transaction_id", trx.Id),
					zap.Error(err))
				continue
			}

			switch dbop.TableName {
			case string(EOSStakeTable):
				if !isEOSStake {
					zlogger.Error("something terribly wrong happened: table eosio stake but not eosio stake",
						zap.String("token_contract", string(tokenContract)),
						zap.String("symbol", symbolCode.String()))
					continue
				}
				eosStakeEntry, err := getStakeEntryFromDBRow(tokenContract, dbop.Scope, row)
				if err != nil {
					zlogger.Error("cannot apply stake entry",
						zap.String("token_contract", string(tokenContract)),
						zap.String("symbol", symbolCode.String()),
						zap.Error(err))
					continue
				}
				muts.SetStake(eosStakeEntry)
			case string(AccountsTable):

				eosToken := t.cache.TokenContract(tokenContract, symbolCode)
				if eosToken == nil {
					zlogger.Warn("unsupported token for contract",
						zap.String("token_contract", string(tokenContract)),
						zap.String("symbol", symbolCode.String()))
					continue
				}

				accountBalance, err := getAccountBalanceFromDBRow(tokenContract, TokenToEOSSymbol(eosToken), dbop.Scope, row)
				if err != nil {
					zlogger.Warn("could not create account balance from dbop row",
						zap.String("token_contract", string(tokenContract)),
						zap.String("symbol", symbolCode.String()),
						zap.String("scope", dbop.Scope),
						zap.String("dbop_row", string(row)))
					continue
				}

				if dbop.NewData == nil {
					// if the db operation has no new data so it removed it
					muts.RemoveBalance(accountBalance)
				} else {
					muts.SetBalance(accountBalance)
				}
			case string(StatTable):
				var symbol *eos.Symbol
				eosToken := t.cache.TokenContract(tokenContract, symbolCode)
				if eosToken == nil {
					zlogger.Debug("new token contract", zap.String("token_contract", string(tokenContract)), zap.String("symbol", symbolCode.String()))
				} else {
					symbol = TokenToEOSSymbol(eosToken)
				}

				token, err := getTokenFromDBRow(tokenContract, symbol, row)
				if err != nil {
					zlogger.Warn("could not create token from dbop row",
						zap.String("token_contract", string(tokenContract)),
						zap.String("symbol", symbolCode.String()),
						zap.String("scope", dbop.Scope),
						zap.String("dbop_row", string(row)))
					continue

				}
				muts.SetToken(token)
			}
		}
	}
	errs := t.cache.Apply(muts, blk)
	if len(errs) != 0 {
		// TODO eventually catch fatal errors and break or ... what can we do ?
		zlog.Warn("errors applying block",
			zap.String("block_id", block.ID()),
			zap.Errors("errors", errs),
		)
	}
	if t.saveEveryNBlock != 0 && blk.Number%t.saveEveryNBlock == 0 {
		// TODO Should this be done async? if so we would need to add locks
		t.cache.SaveToFile()
	}
	return nil
}
