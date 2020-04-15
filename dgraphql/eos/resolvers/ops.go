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

package resolvers

import (
	"context"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/dfuse-io/derr"
	"github.com/dfuse-io/dfuse-eosio/dgraphql/types"
	pbeos "github.com/dfuse-io/dfuse-eosio/pb/dfuse/codecs/eos"
	commonTypes "github.com/dfuse-io/dgraphql/types"
	"github.com/dfuse-io/logging"
	abicodec "github.com/dfuse-io/pbgo/dfuse/abicodec/eosio/v1"
	"github.com/graph-gophers/graphql-go"
	"go.uber.org/zap"
)

type RAMOp struct {
	op *pbeos.RAMOp
}

func (o *RAMOp) Operation() string {
	// While the EOSWS interface uses lower case values (so returned by `op.LegacyOperation()`),
	// the GraphQL interface uses upper case characters. Hence the `ToUpper` call.
	return strings.ToUpper(o.op.LegacyOperation())
}

func (o *RAMOp) Payer() string       { return o.op.Payer }
func (o *RAMOp) Delta() types.Int64  { return types.Int64(o.op.Delta) }
func (o *RAMOp) Usage() types.Uint64 { return types.Uint64(o.op.Usage) }

type DTrxOp struct {
	op *pbeos.DTrxOp
}

func (o *DTrxOp) Operation() string           { return o.op.LegacyOperation() }
func (o *DTrxOp) Sender() *string             { return optS(o.op.Sender) }
func (o *DTrxOp) SenderID() *string           { return optS(o.op.SenderId) }
func (o *DTrxOp) Payer() *string              { return optS(o.op.Payer) }
func (o *DTrxOp) PublishedAt() *graphql.Time  { return toOptTime(o.op.PublishedAt) }
func (o *DTrxOp) DelayUntil() *graphql.Time   { return toOptTime(o.op.DelayUntil) }
func (o *DTrxOp) ExpirationAt() *graphql.Time { return toOptTime(o.op.ExpirationAt) }
func (o *DTrxOp) TrxID() *string              { return optS(o.op.TransactionId) }
func (o *DTrxOp) Transaction() *Transaction {
	return &Transaction{
		t: o.op.Transaction,
	}
}

type TableOp struct {
	op *pbeos.TableOp
}

func (o *TableOp) Operation() string { return o.op.LegacyOperation() }
func (o *TableOp) Table() *TableOpKey {
	return &TableOpKey{
		code:  o.op.Code,
		scope: o.op.Scope,
		table: o.op.TableName,
	}
}

type TableOpKey struct {
	code  string
	scope string
	table string
}

func (t *TableOpKey) Code() string  { return t.code }
func (t *TableOpKey) Table() string { return t.table }

// WARN: because of `args`, there are chances that we spin go-routines
// for nothing, perhaps we'll want to tweak `graphql-go` so it doesn't
// spin goroutines just because we have a param here.  Same for DBOpKey
func (t *TableOpKey) Scope(args struct{ Encoding string }) string {
	return nameToEncoding(t.scope, args.Encoding)
}

type DecodedObject struct {
	object *commonTypes.JSON
	err    string
}

func (t *DecodedObject) Object() *commonTypes.JSON { return t.object }
func (t *DecodedObject) Error() *string {
	if t.err == "" {
		return nil
	}
	return &t.err
}

type DBOp struct {
	op             *pbeos.DBOp
	abiCodecClient abicodec.DecoderClient
	blockNum       uint64
	key            *DBOpKey
}

func newDBOp(op *pbeos.DBOp, blockNum uint64, abiCodecClient abicodec.DecoderClient) *DBOp {
	return &DBOp{
		blockNum:       blockNum,
		abiCodecClient: abiCodecClient,
		op:             op,
		key: &DBOpKey{
			code:  op.Code,
			scope: op.Scope,
			table: op.TableName,
			key:   op.PrimaryKey,
		},
	}
}
func (o *DBOp) Operation() string { return o.op.LegacyOperation() }
func (o *DBOp) OldPayer() *string { return optS(o.op.OldPayer) }
func (o *DBOp) NewPayer() *string { return optS(o.op.NewPayer) }
func (o *DBOp) OldData() *string  { return optS(hex.EncodeToString(o.op.OldData)) }
func (o *DBOp) NewData() *string  { return optS(hex.EncodeToString(o.op.NewData)) }

func (o *DBOp) OldJSON(ctx context.Context) *DecodedObject {
	json, err := o.decode(ctx, hex.EncodeToString(o.op.OldData))

	errDesc := ""
	if err != nil {
		errDesc = err.Error()
	}
	return &DecodedObject{
		object: json,
		err:    errDesc,
	}
}

func (o *DBOp) NewJSON(ctx context.Context) *DecodedObject {
	json, err := o.decode(ctx, hex.EncodeToString(o.op.NewData))

	errDesc := ""
	if err != nil {
		errDesc = err.Error()
	}
	return &DecodedObject{
		object: json,
		err:    errDesc,
	}
}

func (o *DBOp) Key() *DBOpKey { return o.key }

func (o *DBOp) decode(ctx context.Context, data string) (*commonTypes.JSON, error) {
	if data == "" {
		return nil, nil
	}

	d, err := hex.DecodeString(data)
	if err != nil {
		return nil, derr.Wrapf(err, "invalid hex data: %s", data)
	}

	zlogger := logging.Logger(ctx, zlog)

	clientDeadline := time.Now().Add(10 * time.Second)
	ctx, cancel := context.WithDeadline(ctx, clientDeadline)
	defer cancel()

	start := time.Now()
	resp, err := o.abiCodecClient.DecodeTable(ctx, &abicodec.DecodeTableRequest{
		AtBlockNum: uint32(o.blockNum),
		Payload:    d,
		Table:      o.key.table,
		Account:    o.key.code,
	})

	if err != nil {
		zlogger.Info("failed to decode table data", zap.Uint64("block_num", o.blockNum), zap.String("account", o.key.code), zap.String("table", o.key.table), zap.String("payload", data), zap.Error(err))
		return nil, fmt.Errorf(`failed to decode code '%s' table '%s' data '%s' at block '%d': %s`, o.key.scope, o.key.table, data, o.blockNum, err.Error())
	}

	j := commonTypes.JSON([]byte(resp.JsonPayload))

	zlogger.Debug("dbops decoded", zap.Duration("in", time.Since(start)))
	return &j, nil
}

type DBOpKey struct {
	code  string
	scope string
	table string
	key   string
}

func (k *DBOpKey) Code() string  { return k.code }
func (k *DBOpKey) Table() string { return k.table }

func (k *DBOpKey) Scope(args struct{ Encoding string }) string {
	return nameToEncoding(k.scope, args.Encoding)
}
func (k *DBOpKey) Key(args struct{ Encoding string }) string {
	return nameToEncoding(k.key, args.Encoding)
}
