package tokenmeta

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	pbstatedb "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/statedb/v1"

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

var maxStateDBRetry = 5

type TokenMeta struct {
	*shutter.Shutter

	source          bstream.Source
	cache           cache.Cache
	abiCodecCli     pbabicodec.DecoderClient
	abisCache       map[string]*abiItem
	saveEveryNBlock uint32
	stateClient     pbstatedb.StateClient
}

func NewTokenMeta(cache cache.Cache, abiCodecCli pbabicodec.DecoderClient, saveEveryNBlock uint32, stateClient pbstatedb.StateClient) *TokenMeta {
	return &TokenMeta{
		Shutter:         shutter.New(),
		cache:           cache,
		abisCache:       map[string]*abiItem{},
		abiCodecCli:     abiCodecCli,
		saveEveryNBlock: saveEveryNBlock,
		stateClient:     stateClient,
	}
}

func (t *TokenMeta) decodeDBOpToRow(data []byte, tableName eos.TableName, contract eos.AccountName, blocknum uint32) (json.RawMessage, error) {
	abi, err := t.getABI(contract, blocknum)
	if err != nil {
		return nil, fmt.Errorf("cannot get ABI: %w", err)
	}

	return decodeTableRow(data, tableName, abi)
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

func (i *TokenMeta) addNewTokenContract(ctx context.Context, tokenContract eos.AccountName, block bstream.BlockRef) error {
	for attempt := 1; true; attempt++ {
		tokens, bals, err := processContract(ctx, tokenContract, uint32(block.Num()), i.stateClient)
		if err != nil {
			if !isRetryableStateDBError(err) {
				zlog.Info("invalid token contract, unable to get symbols with non-retryable error",
					zap.String("token_contract", string(tokenContract)),
					zap.Error(err),
				)
				return fmt.Errorf("invalid token contract, unable to get symbols with non-retryable error: %w", err)
			}

			if attempt > maxStateDBRetry {
				return fmt.Errorf("failing after 5 attempts to get symbols from token contract: %w", err)
			}

			zlog.Warn("unable to get symbols from token contract, retrying",
				zap.String("token_contract", string(tokenContract)),
				zap.Error(err),
			)

			time.Sleep(time.Duration(attempt) * time.Second)
			continue
		}

		mutations := &cache.MutationsBatch{}
		for _, token := range tokens {
			mutations.SetToken(token)
		}

		for _, bal := range bals {
			mutations.SetBalance(bal)
		}
		i.cache.Apply(mutations, block)
		return nil
	}
	return nil
}

func shouldProcessDbop(dbop *pbcodec.DBOp, actionMatcher pbcodec.FilteringActionMatcher) bool {
	if !actionMatcher.Matched(dbop.ActionIndex) {
		return false
	}

	return dbop.TableName == string(AccountsTable) || dbop.TableName == string(StatTable)
}

func shouldProcessAction(actTrace *pbcodec.ActionTrace, actionMatcher pbcodec.FilteringActionMatcher) bool {
	// TODO should I do this check? when does actionMatcher know if it is system action
	if !actionMatcher.Matched(actTrace.ExecutionIndex) {
		return false
	}
	if actTrace.Receiver != "eosio" || actTrace.Action.Account != "eosio" {
		return false
	}
	return actTrace.Action.Name == "setabi"
}

func TokenToEOSSymbol(e *pbtokenmeta.Token) *eos.Symbol {
	return &eos.Symbol{
		Precision: uint8(e.Precision),
		Symbol:    e.Symbol,
	}
}
