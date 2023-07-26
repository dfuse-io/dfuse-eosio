// Copyright 2019 dfuse Platform Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package codec

import (
	"fmt"
	"sync"
	"time"

	"github.com/eoscanada/eos-go"
	"go.uber.org/zap"
)

var emptyCache *ABICache = nil

type ABICache struct {
	sync.RWMutex

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

func newABICache() *ABICache {
	return &ABICache{
		abis:         map[string]map[uint64]*eos.ABI{},
		abisOrdering: map[string][]uint64{},
	}
}

// addABI adds the ABI to cache assuming it follows the latest stored ABI for this
// contract. For example, assuming a series of ABI for which the latest ABI
// change was peformed at global sequence #450, then it's assumed that the receive `globalSequence`
// argument is greater than 450.
//
// If the invariant is not respected, an error is returned.
func (c *ABICache) addABI(contract string, globalSequence uint64, abi *eos.ABI) error {
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

// findABI for the given `contract` at which `globalSequence` was the most
// recent active ABI.
func (c *ABICache) findABI(contract string, globalSequence uint64) *eos.ABI {
	if c == nil {
		return nil
	}

	contractOrdering := c.abisOrdering[contract]
	if len(contractOrdering) <= 0 {
		return nil
	}

	// Walk the active global sequence in reverse order, and pick the first one that was
	// set before the request `globalSequence` (`x <= globalSequence`) but never set after.
	for i := len(contractOrdering) - 1; i >= 0; i-- {
		activeGlobalSequence := contractOrdering[i]
		if activeGlobalSequence <= globalSequence {
			return c.abis[contract][activeGlobalSequence]
		}
	}

	return nil
}

func (c *ABICache) truncateAfterOrEqualTo(globalSequence uint64) {
	if c == nil {
		return
	}

	startTime := time.Now()
	removedCount := 0

	for contract, contractOrdering := range c.abisOrdering {
		if len(contractOrdering) <= 0 {
			continue
		}

		pivot, preservedSet, cutSet := truncateAfterOrEqual(contractOrdering, globalSequence)
		if traceEnabled {
			zlog.Debug("truncating contract abi",
				zap.String("contract", contract),
				zap.Int("pivot", pivot),
				zap.Int("preserved_count", len(preservedSet)),
				zap.Int("cut_count", len(cutSet)),
			)
		}

		if len(cutSet) <= 0 {
			continue
		}

		contractAbis := c.abis[contract]
		for _, removedGlobalSequence := range cutSet {
			delete(contractAbis, removedGlobalSequence)
		}

		if len(contractAbis) <= 0 {
			delete(c.abis, contract)
		}

		if len(preservedSet) <= 0 {
			delete(c.abisOrdering, contract)
		} else {
			c.abisOrdering[contract] = preservedSet
		}

		removedCount += len(cutSet)
	}

	zlog.Debug("completed cache truncation",
		zap.Duration("elapsed", time.Since(startTime)),
		zap.Int("removed_abi", removedCount),
		zap.Uint64("truncated_at", globalSequence),
	)
}

func truncateAfterOrEqual(slice []uint64, element uint64) (pivot int, preservedSet, cutSet []uint64) {
	pivot = -1
	count := len(slice)
	if count <= 0 {
		return pivot, nil, nil
	}

	for i := count - 1; i >= 0; i-- {
		if slice[i] < element {
			pivot = i
			break
		}
	}

	// Every elements were before the searched element, everything must be preserved
	if pivot == count-1 {
		return pivot, slice, nil
	}

	// Every elements were after or equal to the searched element, everything must be removed
	if pivot == -1 {
		return pivot, nil, slice
	}

	return pivot, slice[:pivot+1], slice[pivot+1:]
}
