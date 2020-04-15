// Copyright 2019 dfuse Platform Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package abicodec

import (
	"context"
	"encoding/json"

	"github.com/dfuse-io/derr"
	pbabicodec "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/abicodec/v1"
	"github.com/eoscanada/eos-go"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
)

type Decoder struct {
	cache Cache
}

func NewDecoder(cache Cache) *Decoder {
	return &Decoder{
		cache: cache,
	}
}

func (d *Decoder) decodeAction(account string, action string, data []byte, blockNum uint32) ([]byte, uint32, error) {
	abiItem := d.cache.ABIAtBlockNum(account, blockNum)
	if abiItem != nil {
		zlog.Debug("Found abi", zap.String("account", account), zap.Uint32("at_block_num", blockNum))
		out, err := abiItem.ABI.DecodeAction(data, eos.ActionName(action))
		if err != nil {
			return nil, 0, derr.Status(codes.InvalidArgument, err.Error())
		}
		return out, abiItem.BlockNum, nil
	}

	return nil, 0, derr.Statusf(codes.NotFound, "no ABI found for account: %s at block %d", account, blockNum)
}

func (d *Decoder) decodeTable(account string, table string, data []byte, blockNum uint32) ([]byte, uint32, error) {
	abiItem := d.cache.ABIAtBlockNum(account, blockNum)
	if abiItem != nil {
		zlog.Debug("Found abi", zap.String("account", account), zap.Uint32("at_block_num", blockNum))
		out, err := abiItem.ABI.DecodeTableRow(eos.TableName(table), data)
		if err != nil {
			zlog.Info("Failed to decode table data", zap.Error(err), zap.ByteString("data", data))
			return nil, 0, derr.Status(codes.InvalidArgument, err.Error())
		}
		return out, abiItem.BlockNum, nil
	}

	return nil, 0, derr.Statusf(codes.NotFound, "no ABI found for account: %s at block %d", account, blockNum)
}

func (d *Decoder) getABI(account string, blockNum uint32) ([]byte, uint32, error) {
	abiItem := d.cache.ABIAtBlockNum(account, blockNum)
	if abiItem != nil {
		zlog.Debug("Found abi", zap.String("account", account), zap.Uint32("at_block_num", blockNum))
		data, err := json.Marshal(abiItem.ABI)
		if err != nil {
			zlog.Info("Failed to decode abi", zap.Error(err), zap.ByteString("data", data))
			return nil, 0, derr.Status(codes.InvalidArgument, err.Error())
		}
		return data, abiItem.BlockNum, nil
	}
	return nil, 0, derr.Statusf(codes.NotFound, "no ABI found for account: %s at block %d", account, blockNum)
}

func (d *Decoder) DecodeTable(cxt context.Context, req *pbabicodec.DecodeTableRequest) (*pbabicodec.Response, error) {

	out, abiBlockNum, err := d.decodeTable(req.Account, req.Table, req.Payload, req.AtBlockNum)

	if err != nil {
		return nil, err
	}

	resp := &pbabicodec.Response{
		JsonPayload: string(out),
		AbiBlockNum: abiBlockNum,
	}

	return resp, nil
}

func (d *Decoder) DecodeAction(ctx context.Context, req *pbabicodec.DecodeActionRequest) (*pbabicodec.Response, error) {

	out, abiBlockNum, err := d.decodeTable(req.Account, req.Action, req.Payload, req.AtBlockNum)

	if err != nil {
		return nil, err
	}

	resp := &pbabicodec.Response{
		JsonPayload: string(out),
		AbiBlockNum: abiBlockNum,
	}

	return resp, nil
}

func (d *Decoder) GetAbi(ctx context.Context, req *pbabicodec.GetAbiRequest) (*pbabicodec.Response, error) {

	out, abiBlockNum, err := d.getABI(req.Account, req.AtBlockNum)
	if err != nil {
		return nil, err
	}

	resp := &pbabicodec.Response{
		JsonPayload: string(out),
		AbiBlockNum: abiBlockNum,
	}

	return resp, nil
}
