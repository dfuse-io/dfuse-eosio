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

package trxdb

import (
	"fmt"

	pbtrxdb "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/trxdb/v1"
	"github.com/golang/protobuf/proto"
	"go.uber.org/zap"
)

type ProtoDecoder struct{}

func NewProtoDecoder() *ProtoDecoder {
	return &ProtoDecoder{}
}

func (d *ProtoDecoder) Into(cnt []byte, msg proto.Message) error {
	err := proto.Unmarshal(cnt, msg)
	if err != nil {
		return err
	}

	return nil
}

func (d *ProtoDecoder) MustInto(cnt []byte, msg proto.Message) {
	if err := d.Into(cnt, msg); err != nil {
		panic(fmt.Sprintf("proto decode error: %s", err.Error()))
	}
}

type ProtoEncoder struct{}

func NewProtoEncoder() *ProtoEncoder {
	return &ProtoEncoder{}
}

func (e *ProtoEncoder) MustProto(obj proto.Message) (out []byte) {
	data, err := proto.Marshal(obj)
	if err != nil {
		panic(fmt.Sprintf("proto encode failed: %s", err))
	}

	if traceEnabled {
		zlog.Debug("marshalled protobuf message to binary", zap.String("id", fmt.Sprintf("%s (%T)", messageIdentifier(obj), obj)), zap.Int("payload", len(data)))
	}

	return data
}

func messageIdentifier(obj proto.Message) string {
	switch v := obj.(type) {
	case *pbtrxdb.TrxRow:
		return v.Receipt.Id
	case *pbtrxdb.TrxTraceRow:
		return v.TrxTrace.Id
	case *pbtrxdb.BlockRow:
		return v.Block.Id
	case *pbtrxdb.AccountRow:
		return v.Name
	case *pbtrxdb.ImplicitTrxRow:
		return v.Name
	case *pbtrxdb.DtrxRow:
		return "deferred"
	}

	return "<n/a>"
}
