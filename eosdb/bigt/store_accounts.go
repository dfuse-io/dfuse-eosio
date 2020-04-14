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
	"encoding/json"

	pbdeos "github.com/dfuse-io/pbgo/dfuse/codecs/deos"
	"github.com/eoscanada/eos-go"
	"github.com/eoscanada/eos-go/system"
	"go.uber.org/zap"
)

func (db *EOSDatabase) storeNewAccount(blk *pbdeos.Block, trxTrace *pbdeos.TransactionTrace, actionTrace *pbdeos.ActionTrace) {
	var newAccount *system.NewAccount
	err := eos.UnmarshalBinary(actionTrace.Action.GetRawData(), &newAccount)
	if err != nil {
		zlog.Error("unable to unmarshal system newaccount action", zap.Error(err), zap.ByteString("data", actionTrace.Action.GetRawData()))
		return
	}

	creator := string(newAccount.Creator)
	creatorID, err := eos.StringToName(creator)
	if err != nil {
		zlog.Error("account creator should have been a valid EOS name", zap.Error(err), zap.String("creator", creator))
		return
	}

	created := string(newAccount.Name)
	createdID, err := eos.StringToName(created)
	if err != nil {
		zlog.Error("account created should have been a valid EOS name", zap.Error(err), zap.String("created", created))
		return
	}

	o := &AccountCreator{
		Created:   created,
		Creator:   creator,
		BlockID:   blk.Id,
		BlockNum:  blk.Number,
		BlockTime: blk.MustTime(),
		TrxID:     trxTrace.Id,
	}

	key := Keys.Account(createdID)
	content, err := json.Marshal(o)
	if err != nil {
		zlog.Error("unable to marshal account info to JSON", zap.Error(err), zap.String("trx_id", trxTrace.Id))
		return
	}

	zlog.Debug("account", zap.String("key", key), zap.String("creator", o.Creator), zap.String("content", string(content)))
	db.Accounts.PutCreator(key, o.Creator, content)

	// FIXME: this was never used, we write some schtuff but we never read that table.
	// We can safely remove it, and replace it when we need it with something else.
	zlog.Debug("link of creator to created", zap.String("creator", o.Creator), zap.String("created", o.Created))
	linkKey := Keys.AccountLink(creatorID, createdID)
	db.Accounts.PutMetaExists(linkKey)
}
