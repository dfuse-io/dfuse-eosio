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

package mdl

import (
	"time"

	pbcodec "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/codec/v1"
	"github.com/eoscanada/eos-go"
	"github.com/golang/protobuf/ptypes"
)

func ToV1Account(in *pbcodec.AccountCreationRef) *Account {
	account := &Account{
		AccountResp: &eos.AccountResp{},
	}

	if in != nil {
		blockTime, _ := ptypes.Timestamp(in.BlockTime)
		account.Creator = &AccountCreator{
			Created:   in.Account,
			Creator:   in.Creator,
			BlockID:   in.BlockId,
			BlockNum:  uint32(in.BlockNum),
			BlockTime: blockTime,
			TrxID:     in.TransactionId,
		}
	}
	// account.AccountVerifications = accountFromDB.AccountVerifications
	// account.LinkedPermissions = accountFromDB.LinkedPermissions
	return account
}

type Account struct {
	Creator *AccountCreator `json:"creator"`
	*eos.AccountResp
	LinkedPermissions    []*LinkedPermission   `json:"linked_permissions"`
	AccountVerifications *AccountVerifications `json:"account_verifications"`
	HasContract          bool                  `json:"has_contract"`
}

type AccountCreator struct {
	Created   string    `json:"created"`
	Creator   string    `json:"creator"`
	BlockID   string    `json:"block_id"`
	BlockNum  uint32    `json:"block_num"`
	BlockTime time.Time `json:"block_time"`
	TrxID     string    `json:"trx_id"`
}

type LinkedPermission struct {
	ActionKey      string `json:"action_key"`
	PermissionName string `json:"permission_name"`
}

type AccountResponse struct {
	Name        eos.Name
	CreatorName eos.Name
	Creator     *AccountCreator

	// TODO: trash these two, LinkedPermissions are never updated
	// except on creation, we have a lot better in the State DB, with
	// that info at each block height.  And zero process reads that
	// information across the `dfuse` product base.
	LinkedPermissions []*LinkedPermission
	// TODO: trash this, was never implemented anyway.. was just an idea.. We'll have that
	// somewhere else in the future, if ever..
	AccountVerifications *AccountVerifications
}

type Verifiable struct {
	Handle    string `json:"handle"`
	Claim     string `json:"claim"`
	Verified  bool   `json:"verified"`
	LastCheck string `json:"last_check"`
}

type AccountVerifications struct {
	Email    *Verifiable `json:"email"`
	Website  *Verifiable `json:"website"`
	Twitter  *Verifiable `json:"twitter"`
	Github   *Verifiable `json:"github"`
	Telegram *Verifiable `json:"telegram"`
	Facebook *Verifiable `json:"facebook"`
	Reddit   *Verifiable `json:"reddit"`
}
