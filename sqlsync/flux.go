package sqlsync

import (
	"encoding/json"

	"github.com/eoscanada/eos-go"
)

//	scopesReq := &fluxdb.GetTableScopesRequest{
//		Account: account,
//		Table:   eos.TableName("..."),
//	}

//	scopesResp, err := fluxClient.GetTableScopes(ctx, startBlockNum, scopesReq)
//	if err != nil {
//		zlog.Warn("cannot get table scope", zap.Error(err))
//		return nil, err
//	}

func decodeTableRow(data []byte, tableName eos.TableName, abi *eos.ABI) (json.RawMessage, error) {
	out, err := abi.DecodeTableRow(tableName, data)
	if err != nil {
		return nil, err
	}
	return out, err
}
