package tokenmeta

import (
	"encoding/json"
	"fmt"

	"github.com/dfuse-io/bstream"
	pbabicodec "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/abicodec/v1"
	pbcodec "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/codec/v1"
	pbtokenmeta "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/tokenmeta/v1"
	"github.com/dfuse-io/dfuse-eosio/tokenmeta/cache"
	"github.com/dfuse-io/shutter"
	"github.com/eoscanada/eos-go"
	"go.uber.org/zap"
)

const AccountsTable eos.TableName = eos.TableName("accounts")
const StatTable eos.TableName = eos.TableName("stat")
const EOSStakeTable eos.TableName = eos.TableName("delband")

type TokenMeta struct {
	*shutter.Shutter

	source          bstream.Source
	cache           cache.Cache
	abiCodecCli     pbabicodec.DecoderClient
	abisCache       map[string]*abiItem
	saveEveryNBlock uint32
}

func (t *TokenMeta) decodeDBOpToRow(data []byte, tableName eos.TableName, contract eos.AccountName, blocknum uint32) (json.RawMessage, error) {
	abi, err := t.getABI(contract, blocknum)
	if err != nil {
		return nil, fmt.Errorf("cannot get ABI: %w", err)
	}

	return decodeTableRow(data, tableName, abi)
}

func NewTokenMeta(cache cache.Cache, abiCodecCli pbabicodec.DecoderClient, saveEveryNBlock uint32) *TokenMeta {
	return &TokenMeta{
		Shutter:         shutter.New(),
		cache:           cache,
		abisCache:       map[string]*abiItem{},
		abiCodecCli:     abiCodecCli,
		saveEveryNBlock: saveEveryNBlock,
	}
}

func (t *TokenMeta) ProcessBlock(block *bstream.Block, obj interface{}) error {
	// forkable setup will only yield irreversible blocks
	muts := &cache.MutationsBatch{}
	blk := block.ToNative().(*pbcodec.Block)

	if (blk.Number % 120) == 0 {
		zlog.Info("process blk 1/120", zap.String("block_id", block.ID()), zap.Uint64("blocker_number", block.Number))
	}

	for _, trx := range blk.TransactionTraces() {
		zlogger := zlog.With(zap.Uint64("blk_id", block.Num()), zap.String("trx_id", trx.Id))

		for _, dbop := range trx.DbOps {
			if !shouldProcessDbop(dbop) {
				continue
			}
			zlog.Debug("processing dbop", zap.String("contract", dbop.Code), zap.String("table", dbop.TableName), zap.String("scope", dbop.Scope), zap.String("primary_key", dbop.PrimaryKey))

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
		zlog.Warn("errors applying block", zap.String("block_id", block.ID()), zap.Errors("errors", errs))
	}
	if t.saveEveryNBlock != 0 && blk.Number%t.saveEveryNBlock == 0 {
		// TODO Should this be done async? if so we would need to add locks
		t.cache.SaveToFile()
	}
	return nil
}

func (i *TokenMeta) Launch() error {
	zlog.Info("launching pipeline")
	go i.source.Run()

	<-i.source.Terminated()
	zlog.Info("source is done")

	zlog.Info("export cache")
	err := i.cache.SaveToFile()
	if err != nil {
		zlog.Error("error exporting cache on shutdown", zap.Error(err))
	}

	if err := i.source.Err(); err != nil {
		zlog.Error("source shutdown with error", zap.Error(err))
		return err
	}

	return nil
}

func shouldProcessDbop(dbop *pbcodec.DBOp) bool {
	if dbop.TableName == string(AccountsTable) || dbop.TableName == string(StatTable) {
		return true
	}
	return false
}

func shouldProcessAction(actionTrace *pbcodec.ActionTrace) bool {
	if actionTrace.Action.Name == "close" {
		return true
	}
	return false
}

func TokenToEOSSymbol(e *pbtokenmeta.Token) *eos.Symbol {
	return &eos.Symbol{
		Precision: uint8(e.Precision),
		Symbol:    e.Symbol,
	}
}
