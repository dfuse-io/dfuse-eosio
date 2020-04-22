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
package codec

import (
	"fmt"
	"math"
	"sort"

	"github.com/dfuse-io/bstream"
	"github.com/dfuse-io/bstream/forkable"
	pbcodec "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/codec/v1"
	"github.com/eoscanada/eos-go"
	"github.com/eoscanada/eos-go/system"
	"go.uber.org/zap"
)

type ABIDecoder struct {
	// We will need to deal with reversible segments and undo/redo on the cache. Maybe a keeping a forkdb
	// and updating it correctly keep track of "setabi" operation and correctly undo/redo on our cache based
	// on the forkdb segments might be the best solution. Specially since we have `PostProcessBlock`
	forkSource *forkable.Forkable

	// The actual handler that post-process block to add decoded information to.
	handler bstream.Handler

	// Represents the actual cache information. The map structure is for each `contract`, keep a mapping
	// of `globalSequenceNumber` to `ABI` (i.e. `map[<contract>]map[<globalSequenceNumber>]<ABI>`). Here
	// an actual map content example to get a better idea.
	//
	// ```
	// {
	// 	"eosio": {
	// 		4000: `ABI #3`,
	// 		0: `ABI #1`,
	// 		100: `ABI #2`,
	// 	},
	// 	"eosio.token": {
	// 		0: `ABI #1`,
	// 		1000: `ABI #2`,
	// 	}
	// }
	// ```
	//
	// **Important** The second inner map is un-ordered, to retrieve correct ABI based on sequential
	//               ordering, you must use `abisOrdering` element.
	abis map[string]map[uint64]*eos.ABI

	// Represents the ABIs ordering values based on `globalSequence`. The map structure is for each
	// `contract`, keep a slice of ordered ABI global sequence number
	// (i.e. `map[<contract>][<index>]<globalSequenceNumber>`). By using this ordering structure, we
	// can inside the `[]uint64` perfrom a binary search to find the correct `<globalSequenceNumber>`
	// then retrieve the corresponding ABI in `abis` element.
	abisOrdering map[string][]uint64
}

func newABIDecoder() *ABIDecoder {
	cache := &ABIDecoder{
		abis:         map[string]map[uint64]*eos.ABI{},
		abisOrdering: map[string][]uint64{},
	}

	// FIXME: What to do about how we initialize our LIB?
	cache.forkSource = forkable.New(bstream.HandlerFunc(cache.processBlock))

	return cache
}

func (c *ABIDecoder) postProcessBlock(block *pbcodec.Block) error {
	return c.forkSource.ProcessBlock(&bstream.Block{
		Id:         block.ID(),
		Number:     block.Num(),
		PreviousId: block.PreviousID(),
		Timestamp:  block.MustTime(),
		LibNum:     block.LIBNum(),
	}, block)
}

func (c *ABIDecoder) processBlock(rawBlock *bstream.Block, obj interface{}) error {
	fObj := obj.(*forkable.ForkableObject)

	// FIXME: Handle undo/redo that removes/adds ABIs operation to the cache
	if fObj.Step != forkable.StepNew {
		return nil
	}

	block := fObj.Obj.(*pbcodec.Block)

	// We first build the ABI cache for the whole block
	zlog.Debug("building abi cache for block", zap.Stringer("block", rawBlock))
	for _, trxTrace := range block.TransactionTraces {
		// FIXME: Add support for failed_dtrx_trace, think about the correct meaning. Answers the
		//        following questions/use cases:
		//        - Assumes dtrx that fails with 3 actions in it. Action@450 (setabi) Action@451 (data with new ABI) Action@0 (fails)
		//          We are currently building the full cache for the block, does it mean we cannot do it? Maybe we should only accumulated
		//          committed block state and for failure causes, we resolve in the transaction trace it self?.
		//        - Think and test weird case that a `eosio:setabi` is called in a successufl `onerror` handler.

		for i, actionTrace := range trxTrace.ActionTraces {
			if actionTrace.FullName() == "eosio:eosio:setabi" {
				setABI := &system.SetABI{}
				err := eos.UnmarshalBinary(actionTrace.Action.RawData, setABI)
				if err != nil {
					return fmt.Errorf("unable to read action trace 'setabi' at index %d in transaction %s: %w", i, trxTrace.Id, err)
				}

				// All sort of garbage can be in this field, skip if we cannot properly decode to an eos.ABI object
				abi := &eos.ABI{}
				err = eos.UnmarshalBinary(setABI.ABI, abi)
				if err != nil {
					zlog.Info("skipping action trace 'setabi' since abi content cannot be unmarshalled correctly", zap.Int("action_index", i), zap.String("trx_id", trxTrace.Id))
					continue
				}

				err = c.addABISequentially(string(setABI.Account), actionTrace.Receipt.GlobalSequence, abi)
				if err != nil {
					return fmt.Errorf("failed to add ABI in action trace at index %d in transaction %s: %w", i, trxTrace.Id, err)
				}
			}
		}
	}

	// FIXME: Optimization: We could optimize notification inside a transaction. We could have a two-pass algorithm.
	//                      In the first pass we loop on all `non-notification` action, decoding them against the ABI.
	//                      In the second pass, we loop on all `notification` action this time and now instead of
	//                      decoding them, we find the action that created the notification and use it's already decoded
	//                      action. This would save us 2 decoding for each `eosio.token` for example.

	// FIXME: Optimization This can then be peformed in parallel since we build the cache locally. Use `dhammer` and hammer
	//                     all transactions traces in parralel!
	zlog.Debug("post-processing all transaction traces", zap.Stringer("block", rawBlock))
	for _, trxTrace := range block.TransactionTraces {
		for i, actionTrace := range trxTrace.ActionTraces {
			globalSequence := uint64(math.MaxUint64)
			if actionTrace.Receipt != nil && actionTrace.Receipt.GlobalSequence != 0 {
				globalSequence = actionTrace.Receipt.GlobalSequence
			}

			err := c.postProcessAction(actionTrace.Action, globalSequence)
			if err != nil {
				return fmt.Errorf("unable to post-process action at index %d with global sequence %d on transaction trace %s: %w", i, actionTrace.Receipt.GlobalSequence, trxTrace.Id, err)
			}
		}

		// FIXME: Performed also for `dtrxOps` and `trxOps`
		// FIXME: How about `dbOps`, do we check them right now?
	}

	// Technically, the next block that will process stuff after this one must be don

	return nil
}

// addABISequentially adds the ABI to cache assuming it follows the latest stored ABI for this
// contract. For example, assuming a series of ABI for which the latest ABI
// change was peformed at global sequence #450, then it's assumed that the receive `globalSequence`
// argument is greater than 450.
//
// If the invariant is not respected, an error is returned.
func (c *ABIDecoder) addABISequentially(contract string, globalSequence uint64, abi *eos.ABI) error {
	zlog.Debug("adding new abi", zap.String("account", contract), zap.Uint64("global_sequence", globalSequence))

	contractOrdering, found := c.abisOrdering[contract]
	if found && len(contractOrdering) > 0 && contractOrdering[len(contractOrdering)-1] > globalSequence {
		return fmt.Errorf("abi is not sequential against latest ABI's global sequence, latest is %d and trying to add %d which is in the past", contractOrdering[len(contractOrdering)-1], globalSequence)
	}

	contractAbis, found := c.abis[contract]
	if !found {
		contractAbis = map[uint64]*eos.ABI{}
		c.abis[contract] = contractAbis
	}

	contractAbis[globalSequence] = abi
	c.abisOrdering[contract] = append(contractOrdering, globalSequence)

	return nil
}

func (c *ABIDecoder) postProcessAction(action *pbcodec.Action, globalSequence uint64) error {
	if len(action.RawData) <= 0 {
		// Nothing to do, there is no action data at all
		return nil
	}

	abi := c.findABI(action.Account, globalSequence)
	if abi == nil {
		return nil
	}

	actionDef := abi.ActionForName(eos.ActionName(action.Name))
	if actionDef == nil {
		if traceEnabled {
			zlog.Debug("skipping action since, ABI found for it but action is not in it", zap.String("action", action.Account+":"+action.Name), zap.Uint64("global_sequence", globalSequence))
		}

		return nil
	}

	decoder := eos.NewDecoder(action.RawData)
	jsonData, err := abi.Decode(decoder, actionDef.Type)
	if err != nil {
		return err
	}

	action.JsonData = string(jsonData)

	// FIXME: Do we need to keep both here? I'm not sure, reading `eos_to_proto` did not give me the final answer (it coded that late!)
	action.RawData = nil

	return nil
}

// findABI for the given `contract` at which `globalSequence` was the most
// recent active ABI.
func (c *ABIDecoder) findABI(contract string, globalSequence uint64) *eos.ABI {
	contractOrdering := c.abisOrdering[contract]
	if len(contractOrdering) <= 0 {
		return nil
	}

	activeABIIndex := sort.Search(len(contractOrdering)-1, func(i int) bool { return contractOrdering[i] <= globalSequence })
	activeABIGlobalSequence := contractOrdering[activeABIIndex]
	if activeABIIndex == 0 && activeABIGlobalSequence > globalSequence {
		// If the search returned index 0, it might be because the func returned false for all global
		// sequence. Hence, we cannot assume that index 0 necessarly respect our conditions. As such,
		// when 0, we also check the actual value at index 0 and ensure is comes before our currently
		// search global sequence. So that searching an ABI at global sequence 50 where actual first
		// one we have was set at 100 returns `nil`.
		return nil
	}

	return c.abis[contract][activeABIGlobalSequence]
}
