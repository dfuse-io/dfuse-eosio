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

package eosws

import (
	"context"
	"encoding/hex"
	"fmt"

	pbcodec "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/codec/v1"
	"github.com/eoscanada/eos-go"
	"github.com/streamingfast/bstream"
	"github.com/streamingfast/bstream/forkable"
)

type ABIChangeHandler struct {
	stack ABIStack
	code  eos.AccountName
	next  bstream.Handler
}

func NewABIChangeHandler(abiGetter ABIGetter, blockNum uint32, code eos.AccountName, next bstream.Handler, ctx context.Context) (*ABIChangeHandler, error) {
	abiStack := ABIStack{}
	abi, err := abiGetter.GetABI(ctx, blockNum, code)
	if err != nil {
		return nil, err
	}

	if abi == nil {
		return nil, fmt.Errorf("no abi found for block_num %d, code %s", blockNum, code)
	}

	abiStack = abiStack.Push(abi)
	return &ABIChangeHandler{
		stack: abiStack,
		code:  code,
		next:  next,
	}, nil
}

func (h *ABIChangeHandler) ProcessBlock(block *bstream.Block, obj interface{}) error {
	blk := block.ToNative().(*pbcodec.Block)
	abi, err := abiFromBlock(blk, h.code)
	if err != nil {
		return err
	}

	if abi != nil {
		fObj := obj.(*forkable.ForkableObject)
		switch fObj.Step {
		case forkable.StepNew, forkable.StepRedo:
			h.stack = h.stack.Push(abi)
		case forkable.StepUndo:
			var poppedABI *eos.ABI
			h.stack, poppedABI = h.stack.Pop()
			if poppedABI.Version != abi.Version {
				return fmt.Errorf("popped abi version differ from abi version from block")
			}
		}
	}

	return h.next.ProcessBlock(block, obj)
}

func (h *ABIChangeHandler) CurrentABI() *eos.ABI {
	return h.stack.Peek()
}

type ABIStack []*eos.ABI

func (s ABIStack) Push(abi *eos.ABI) ABIStack {
	return append(s, abi)
}

func (s ABIStack) Pop() (ABIStack, *eos.ABI) {
	if len(s) == 0 {
		return s, nil
	}
	n := len(s) - 1 // Top element
	abi := s[n]
	return s[:n], abi
}

func (s ABIStack) Peek() *eos.ABI {
	if len(s) == 0 {
		return nil
	}
	n := len(s) - 1 // Top element
	popABI := s[n]
	return popABI
}

func abiFromBlock(blk *pbcodec.Block, code eos.AccountName) (*eos.ABI, error) {
	for _, trxTrace := range blk.TransactionTraces() {
		for _, actionTrace := range trxTrace.ActionTraces {
			// We process action trace regardless of the block filtering applied
			if actionTrace.Receiver == "eosio" && actionTrace.Action.Account == "eosio" && actionTrace.Action.Name == "setabi" {
				candidateCode := eos.AccountName(actionTrace.GetData("account").String())
				if code != candidateCode {
					continue
				}

				abiBytes, err := hex.DecodeString(actionTrace.GetData("abi").String())
				if err != nil {
					return nil, fmt.Errorf("unable to transform ABI hex data into bytes: %s", err)
				}

				var abi *eos.ABI
				err = eos.NewDecoder(abiBytes).Decode(&abi)
				if err != nil {
					return nil, fmt.Errorf("unable to decode action ABI hex data: %s", err)
				}
				return abi, nil
			}
		}
	}

	return nil, nil
}
