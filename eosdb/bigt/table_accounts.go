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

package bigt

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/bigtable"
	"github.com/abourget/llerrgroup"
	basebigt "github.com/dfuse-io/kvdb/base/bigt"
	eos "github.com/eoscanada/eos-go"
	"go.opencensus.io/trace"
	"go.uber.org/zap"
)

type AccountsTable struct {
	*basebigt.BaseTable

	ColMetaExists      string
	ColCreatorName     string
	ColCreatorJSON     string
	ColPermissionLinks string
	FamilyVerification string
}

func NewAccountsTable(name string, client *bigtable.Client) *AccountsTable {
	return &AccountsTable{
		BaseTable: basebigt.NewBaseTable(name, []string{"verifications", "meta", "perms", "creator"}, client),

		ColMetaExists:      "meta:exists",
		ColCreatorName:     "creator:name",
		ColCreatorJSON:     "creator:json",
		ColPermissionLinks: "perms:links",
		FamilyVerification: "verifications",
	}
}

func (tbl *AccountsTable) ParallelStreamRows(
	ctx context.Context,
	rowRanges []bigtable.RowSet,
	concurrentReadCount uint32,
	processor func(row *AccountResponse) bool,
	opts ...bigtable.ReadOption,
) error {
	group := llerrgroup.New(int(concurrentReadCount))

	zlog.Debug("starting group", zap.Uint32("concurrency", concurrentReadCount))
	for _, rowRange := range rowRanges {
		if group.Stop() {
			zlog.Debug("group completed")
			break
		}

		rowRange := rowRange
		group.Go(func() error {
			ctx, span := trace.StartSpan(ctx, "stream row range")
			defer span.End()
			span.AddAttributes(trace.StringAttribute("row_set", rowSetToString(rowRange)))

			return tbl.StreamRows(ctx, rowRange, processor, opts...)
		})
	}

	zlog.Debug("waiting for all parallel stream rows operation to finish")
	if err := group.Wait(); err != nil {
		return fmt.Errorf("some stream rows operation did not completed successfully: %s", err)
	}

	return nil
}

func rowSetToString(rowSet bigtable.RowSet) string {
	switch v := rowSet.(type) {
	case bigtable.RowList:
		// FIXME: Shall we print a subset of the keys?
		return strings.Join(v, ",")
	case bigtable.RowRange:
		return v.String()
	case bigtable.RowRangeList:
		// FIXME: Shall we print a subset of the keys?
		stringRanges := make([]string, len(v))
		for index, rowRange := range v {
			stringRanges[index] = rowRange.String()
		}

		return strings.Join(stringRanges, ",")
	default:
		return fmt.Sprintf("%#v", rowSet)
	}
}

func (tbl *AccountsTable) StreamRows(
	ctx context.Context,
	rowRange bigtable.RowSet,
	processor func(row *AccountResponse) bool,
	opts ...bigtable.ReadOption,
) error {
	var innerErr error
	err := tbl.BaseTable.ReadRows(ctx, rowRange, func(row bigtable.Row) bool {
		response, err := tbl.parseRowAs(row)
		if err != nil {
			innerErr = err
			return false
		}

		processor(response)
		return true
	}, opts...)

	if err != nil {
		return fmt.Errorf("stream block rows: %s", err)
	}

	if innerErr != nil {
		return fmt.Errorf("stream block rows, inner: %s", innerErr)
	}

	return nil
}

func (tbl *AccountsTable) parseRowAs(row bigtable.Row) (*AccountResponse, error) {
	name, err := Keys.ReadAccount(row.Key())
	if err != nil {
		return nil, fmt.Errorf("name from key: %s", err)
	}

	accountName := eos.NameToString(name)

	creatorName, err := EOSNameColumnItem(row, tbl.ColCreatorName)
	if err != nil && !basebigt.IsErrColumnNotPresent(err) {
		return nil, fmt.Errorf("account creator column %s: %s", tbl.ColCreatorName, err)
	}

	var creatorJSON *AccountCreator
	err = basebigt.JSONColumnItem(row, tbl.ColCreatorJSON, &creatorJSON)
	if err != nil && !basebigt.IsErrColumnNotPresent(err) {
		return nil, fmt.Errorf("account creator json: %s", err)
	}

	return &AccountResponse{
		Name:        eos.Name(accountName),
		CreatorName: creatorName,
		Creator:     creatorJSON,
	}, nil
}

func (tbl *AccountsTable) PutCreator(key string, name string, data json.RawMessage) {
	tbl.SetKey(key, tbl.ColCreatorName, []byte(name))
	if data != nil {
		tbl.SetKey(key, tbl.ColCreatorJSON, []byte(data))
	}
}

func (tbl *AccountsTable) PutPermissionLinks(key string, data json.RawMessage) {
	tbl.SetKey(key, tbl.ColPermissionLinks, []byte(data))
}

func (tbl *AccountsTable) PutVerification(key string, property string, data json.RawMessage) {
	tbl.SetKey(key, tbl.FamilyVerification+":"+property, []byte(data))
}

func (tbl *AccountsTable) PutMetaExists(key string) {
	tbl.SetKey(key, tbl.ColMetaExists, []byte(""))
}

func EOSNameColumnItem(row bigtable.Row, familyColumn string) (eos.Name, error) {
	item, present := basebigt.ColumnItem(row, familyColumn)
	if !present {
		return "", basebigt.NewErrColumnNotPresent(familyColumn)
	}

	return eos.Name(string(item.Value)), nil
}

type AccountResponse struct {
	Name        eos.Name
	CreatorName eos.Name
	Creator     *AccountCreator
}

type AccountCreator struct {
	Created   string    `json:"created"`
	Creator   string    `json:"creator"`
	BlockID   string    `json:"block_id"`
	BlockNum  uint32    `json:"block_num"`
	BlockTime time.Time `json:"block_time"`
	TrxID     string    `json:"trx_id"`
}
