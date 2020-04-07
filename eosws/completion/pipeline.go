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

package completion

import (
	"github.com/dfuse-io/bstream"
	"github.com/dfuse-io/bstream/forkable"
	"github.com/dfuse-io/bstream/hub"
	pbdeos "github.com/dfuse-io/pbgo/dfuse/codecs/deos"
	eos "github.com/eoscanada/eos-go"
	"github.com/eoscanada/eos-go/system"
	"go.uber.org/zap"
)

type Pipeline struct {
	completionInstance Completion
	initialStartBlock  string
	initialLIB         string
	subscriptionHub    *hub.SubscriptionHub
}

func NewPipeline(completionInstance Completion, initialStartBlock string, initialLIB string, subscriptionHub *hub.SubscriptionHub) *Pipeline {
	return &Pipeline{
		completionInstance: completionInstance,
		initialStartBlock:  initialStartBlock,
		initialLIB:         initialLIB,
		subscriptionHub:    subscriptionHub,
	}
}

func (p *Pipeline) Launch() {
	libID := p.initialLIB
	startBlock := eos.BlockNum(p.initialStartBlock)

	handler := bstream.HandlerFunc(func(block *bstream.Block, obj interface{}) error {
		fObj := obj.(*forkable.ForkableObject)
		if fObj.Step != forkable.StepIrreversible {
			// Continue if not an irreversible step
			return nil
		}

		//blk := block.ToNative().(*pbdeos.Block)
		//zlog.Info("Implemented me for god sake!", zap.Any("block", blk))

		// p.processExecutedTransactions(blk.AllExecutedTransactionTraces())

		startBlock = uint32(block.Num())
		libID = fObj.ForkDB.LIBID()

		return nil
	})

	gateHandler := bstream.NewBlockNumGate(uint64(startBlock), bstream.GateExclusive, handler)
	forkableHandler := forkable.New(gateHandler, forkable.WithExclusiveLIB(bstream.BlockRefFromID(libID)))
	source := p.subscriptionHub.NewSourceFromBlockRef(bstream.BlockRefFromID(libID), forkableHandler)

	source.Run()
	source.OnTerminating(func(e error) {
		zlog.Error("completion pipeline failed and quit", zap.Error(e))
	})
}

func (p *Pipeline) processExecutedTransactions(transactions []*pbdeos.TransactionTrace) {
	for _, transaction := range transactions {
		for _, action := range transaction.ActionTraces {
			p.processExecutedAction(action)
		}
	}
}

func (p *Pipeline) processExecutedAction(action *pbdeos.ActionTrace) {
	if action.Action.Name == "newaccount" && action.FullName() == "eosio:eosio:newaccount" {
		p.updateCompletion(action)
	}
}

func (p *Pipeline) updateCompletion(actionTrace *pbdeos.ActionTrace) {
	action := actionTrace.Action
	var newAccount *system.NewAccount
	if err := action.UnmarshalData(&newAccount); err != nil {
		zlog.Error("unable to marshal action as newaccount action while we thought we could",
			zap.String("data", action.JsonData),
			zap.Error(err),
		)
	}

	p.completionInstance.AddAccount(string(newAccount.Name))
}
