package sqlsync

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/dfuse-io/bstream"
	"github.com/dfuse-io/dfuse-eosio/fluxdb-client"
	pbcodec "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/codec/v1"
	"github.com/dfuse-io/dstore"
	"github.com/dfuse-io/shutter"
	"github.com/eoscanada/eos-go"
	"go.uber.org/zap"
)

type SQLSync struct {
	*shutter.Shutter
	db     *DB
	fluxdb fluxdb.Client

	source bstream.Source

	watchedAccounts map[eos.AccountName]*account
	blockstreamAddr string
	blocksStore     dstore.Store
}

func (t *SQLSync) getABI(contract eos.AccountName, blockNum uint32) (*eos.ABI, error) {
	resp, err := t.fluxdb.GetABI(context.Background(), blockNum, contract)
	if err != nil {
		return nil, err
	}

	return resp.ABI, nil
}

func (t *SQLSync) decodeDBOpToRow(data []byte, tableName eos.TableName, contract eos.AccountName, blocknum uint32) (json.RawMessage, error) {
	abi, err := t.getABI(contract, blocknum)
	if err != nil {
		return nil, fmt.Errorf("cannot get ABI: %w", err)
	}

	return decodeTableRow(data, tableName, abi)
}

func NewSQLSync(db *DB, fluxCli fluxdb.Client, blockstreamAddr string, blocksStore dstore.Store) *SQLSync {
	return &SQLSync{
		Shutter:         shutter.New(),
		blockstreamAddr: blockstreamAddr,
		blocksStore:     blocksStore,
		db:              db,
		fluxdb:          fluxCli,
	}
}

func (t *SQLSync) ProcessBlock(block *bstream.Block, obj interface{}) error {
	// forkable setup will only yield irreversible blocks
	blk := block.ToNative().(*pbcodec.Block)

	if (blk.Number % 120) == 0 {
		zlog.Info("process blk 1/120", zap.String("block_id", block.ID()), zap.Uint64("blocker_number", block.Number))
	}

	for _, trx := range blk.TransactionTraces {
		zlogger := zlog.With(zap.Uint64("blk_id", block.Num()), zap.String("trx_id", trx.Id))

		for _, dbop := range trx.DbOps {
			if !shouldProcessDbop(dbop) {
				continue
			}
			zlog.Debug("processing dbop", zap.String("contract", dbop.Code), zap.String("table", dbop.TableName), zap.String("scope", dbop.Scope), zap.String("primary_key", dbop.PrimaryKey))

			rowData := dbop.NewData
			if rowData == nil {
				zlog.Info("using db row old data")
				rowData = dbop.OldData
			}
			contract := eos.AccountName("whatever")
			row, err := t.decodeDBOpToRow(rowData, eos.TableName(dbop.TableName), contract, uint32(block.Number))
			if err != nil {
				zlogger.Error("cannot decode table row",
					zap.String("contract", string(contract)),
					zap.String("table_name", dbop.TableName),
					zap.String("transaction_id", trx.Id),
					zap.Error(err))
				continue
			}
			_ = row

			switch dbop.TableName {
			}
		}
	}
	return nil
}

func shouldProcessDbop(dbop *pbcodec.DBOp) bool {
	//	if dbop.TableName == string(...) {
	//		return true
	//	}
	//	return false
	return false
}

func shouldProcessAction(actionTrace *pbcodec.ActionTrace) bool {
	if actionTrace.Action.Name == "close" {
		return true
	}
	return false
}

func (s *SQLSync) HealthzHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if false {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("not ready"))
			return
		}
		w.Write([]byte("ok"))
		return
	})
}
