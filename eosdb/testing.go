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

package eosdb

import (
	"context"

	pbeos "github.com/dfuse-io/dfuse-eosio/pb/dfuse/codecs/eos"
)

type TestTransactionsReader struct {
	content map[string][]*pbeos.TransactionEvent
}

func NewTestTransactionsReader(content map[string][]*pbeos.TransactionEvent) *TestTransactionsReader {
	return &TestTransactionsReader{content: content}
}

func (r *TestTransactionsReader) GetTransactionTraces(ctx context.Context, idPrefix string) ([]*pbeos.TransactionEvent, error) {
	return r.content[idPrefix], nil
}

func (r *TestTransactionsReader) GetTransactionTracesBatch(ctx context.Context, idPrefixes []string) (out [][]*pbeos.TransactionEvent, err error) {
	for _, prefix := range idPrefixes {
		out = append(out, r.content[prefix])
	}
	return
}

func (r *TestTransactionsReader) GetTransactionEvents(ctx context.Context, idPrefix string) ([]*pbeos.TransactionEvent, error) {
	panic("not implemented")
}

func (r *TestTransactionsReader) GetTransactionEventsBatch(ctx context.Context, idPrefixes []string) ([][]*pbeos.TransactionEvent, error) {
	panic("not implemented")
}
