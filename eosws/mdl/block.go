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
	"fmt"

	"github.com/dfuse-io/dfuse-eosio/codecs/deos"
	pbdeos "github.com/dfuse-io/pbgo/dfuse/codecs/deos"
	eos "github.com/eoscanada/eos-go"
)

// BlockSummary is the dfuse summary information for a given block
// Candidate for a move in eosws-go directly once the related REST API are made
// public.
type BlockSummary struct {
	ID               string                `json:"id"`
	BlockNum         uint32                `json:"block_num"`
	Irreversible     bool                  `json:"irreversible"`
	Header           *eos.BlockHeader      `json:"header"`
	ActiveSchedule   *eos.ProducerSchedule `json:"active_schedule"`
	TransactionCount int                   `json:"transaction_count"`
	SiblingBlocks    []*BlockSummary       `json:"sibling_blocks"`
	DPoSLIBNum       uint32                `json:"dpos_lib_num"`
}

func ToV1BlockSummary(in *pbdeos.BlockWithRefs) (*BlockSummary, error) {
	summary := &BlockSummary{
		ID:               in.Id,
		Irreversible:     in.Irreversible,
		Header:           deos.BlockHeaderToEOS(in.Block.Header),
		TransactionCount: int(in.Block.TransactionCount),
		BlockNum:         in.Block.Number,
		DPoSLIBNum:       in.Block.DposIrreversibleBlocknum,
	}

	// FIXME: At some point, we will like to maybe change the output format so that it fits EOSIO 2.0
	//        format. Will need to think about how all this should be.
	if in.Block.ActiveScheduleV2 != nil {
		downgradedSchedule, err := downgradeActiveScheduleV2ToV1(in.Block.ActiveScheduleV2)
		if err != nil {
			return nil, err
		}
		summary.ActiveSchedule = deos.ProducerScheduleToEOS(downgradedSchedule)
	} else if in.Block.ActiveScheduleV1 != nil {
		summary.ActiveSchedule = deos.ProducerScheduleToEOS(in.Block.ActiveScheduleV1)
	}

	return summary, nil
}

func downgradeActiveScheduleV2ToV1(in *pbdeos.ProducerAuthoritySchedule) (*pbdeos.ProducerSchedule, error) {
	newProducers := make([]*pbdeos.ProducerKey, len(in.Producers))
	for i, producer := range in.Producers {
		pubKey, err := extractFirstPublicKeyFromAuthority(producer.BlockSigningAuthority)
		if err != nil {
			return nil, fmt.Errorf("failed to downgrade schedule: %w", err)
		}
		newProducers[i] = &pbdeos.ProducerKey{
			AccountName:     producer.AccountName,
			BlockSigningKey: pubKey,
		}
	}

	return &pbdeos.ProducerSchedule{
		Version:   in.Version,
		Producers: newProducers,
	}, nil
}

func extractFirstPublicKeyFromAuthority(in *pbdeos.BlockSigningAuthority) (string, error) {
	if in.GetV0() == nil {
		return "", fmt.Errorf("only knowns how to deal with BlockSigningAuthority_V0 type, got %t", in.Variant)
	}

	keys := in.GetV0().GetKeys()
	if len(keys) <= 0 {
		return "", nil
	}

	return keys[0].PublicKey, nil
}
