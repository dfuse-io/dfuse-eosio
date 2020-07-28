// Copyright 2020 dfuse Platform Inc.
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

package eosrest

import (
	"context"
	"net/url"
	"time"

	"github.com/dfuse-io/derr"
	"github.com/dfuse-io/kvdb"
	"github.com/dfuse-io/logging"
	"go.uber.org/zap"
)

type IrreversibleFinder interface {
	IrreversibleIDAtBlockNum(ctx context.Context, blockNum uint32) (string, error)
	IrreversibleIDAtBlockID(ctx context.Context, blockID string) (string, error)
}

type DefaultIrreversibleFinder struct {
	db DB
}

func NewDBReaderBaseIrrFinder(db DB) *DefaultIrreversibleFinder {
	return &DefaultIrreversibleFinder{
		db: db,
	}
}

func (f *DefaultIrreversibleFinder) IrreversibleIDAtBlockNum(ctx context.Context, blockNum uint32) (out string, err error) {
	logging.Logger(ctx, zlog).Debug("fetching irreversible block num", zap.Uint32("block_num", blockNum))

	if blockNum == 0 {
		return "", derr.RequestValidationError(ctx, url.Values{
			"block_num": []string{"The block_num field must be greater than 0"},
		})
	}

	err = Retry(ctx, 5, 500*time.Millisecond, func() error {
		blockRef, err := f.db.GetClosestIrreversibleIDAtBlockNum(ctx, blockNum)
		if err != nil || blockRef == nil {
			if err != kvdb.ErrNotFound {
				logging.Logger(ctx, zlog).Error("cannot get irreversible ID at blocknum", zap.Uint32("block_num", blockNum), zap.Error(err))
			}
			return AppUnableToGetIrreversibleBlockIDError(ctx, string(blockNum))
		}

		out = blockRef.ID()

		return err
	})
	return
}

func (f *DefaultIrreversibleFinder) IrreversibleIDAtBlockID(ctx context.Context, blockID string) (out string, err error) {
	logging.Logger(ctx, zlog).Debug("fetching irreversible block ID", zap.String("block_id", blockID))

	err = Retry(ctx, 5, 500*time.Millisecond, func() error {
		blockRef, err := f.db.GetIrreversibleIDAtBlockID(ctx, blockID)
		if err != nil || blockRef == nil {
			if err != kvdb.ErrNotFound {
				logging.Logger(ctx, zlog).Error("cannot get irreversible ID at blocknum", zap.String("block_id", blockID), zap.Error(err))
			}
			return AppUnableToGetIrreversibleBlockIDError(ctx, blockID)
		}

		out = blockRef.ID()

		return err
	})
	return
}

type TestIrreversibleFinder struct {
	irrID string
	err   error
}

func NewTestIrreversibleFinder(irrID string, err error) *TestIrreversibleFinder {
	return &TestIrreversibleFinder{
		irrID: irrID,
		err:   err,
	}
}

func (f *TestIrreversibleFinder) IrreversibleIDAtBlockNum(ctx context.Context, blockNum uint32) (string, error) {
	return f.irrID, f.err
}

func (f *TestIrreversibleFinder) IrreversibleIDAtBlockID(ctx context.Context, blockID string) (string, error) {
	return f.irrID, f.err
}
