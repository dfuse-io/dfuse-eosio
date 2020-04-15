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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestTimelineTableKey(t *testing.T) {
	assert.Equal(t, "bf:14055441475:00000001a", Keys.TimelineBlockForward(time.Unix(1405544147, 500000000), "00000001a"))
	assert.Equal(t, "br:85944558524:00000001a", Keys.TimelineBlockReverse(time.Unix(1405544147, 500000000), "00000001a"))

	blockTime, blockID, err := Keys.ReadTimelineBlockForward("bf:14055441475:00000001a")
	assert.NoError(t, err)
	assert.True(t, time.Unix(1405544147, 500000000).Equal(blockTime))
	assert.Equal(t, "00000001a", blockID)

	blockTime, blockID, err = Keys.ReadTimelineBlockReverse("br:85944558524:00000001a")
	assert.NoError(t, err)
	assert.True(t, time.Unix(1405544147, 500000000).Equal(blockTime))
	assert.Equal(t, "00000001a", blockID)
}

func TestAccountsTableKey(t *testing.T) {
	assert.Equal(t, "a:0000000000000063", Keys.Account(99))

	account, err := Keys.ReadAccount("a:0000000000000063")
	assert.NoError(t, err)
	assert.Equal(t, uint64(99), account)
}

func TestBlocksTableKey(t *testing.T) {
	expectedOutput := "ffff7ff71122334455"
	assert.Equal(t, expectedOutput, Keys.Block("000080081122334455"))

	ret := Keys.ReadBlock(expectedOutput)
	assert.Equal(t, "000080081122334455", ret)
}

func TestTransactionsTableKey(t *testing.T) {
	assert.Equal(t, "trxid:blockid", Keys.Transaction("trxid", "blockid"))

	trxID, blockID, err := Keys.ReadTransaction("trxid:blockid")
	assert.NoError(t, err)
	assert.Equal(t, "trxid", trxID)
	assert.Equal(t, "blockid", blockID)
}
