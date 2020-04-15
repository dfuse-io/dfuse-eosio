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
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/dfuse-io/kvdb"
)

var maxUnixTimestampDeciSeconds = int64(99999999999)

var Keys Keyer

type Keyer struct{}

// Accounts table

func (Keyer) Account(account uint64) string {
	return "a:" + kvdb.HexName(account)
}

func (Keyer) AccountLink(creator, created uint64) string {
	return "l:" + kvdb.HexName(creator) + ":" + kvdb.HexName(created)
}

func (Keyer) ReadAccount(key string) (account uint64, err error) {
	if !strings.HasPrefix(key, "a:") {
		return 0, errors.New("expected account key starting with 'a:'")
	}

	value, err := strconv.ParseUint(key[2:], 16, 64)
	if err != nil {
		return 0, fmt.Errorf("parsing key for accounts table: %s", err)
	}
	return value, nil
}

// timeline
func (Keyer) TimelineBlockForward(blockTime time.Time, blockID string) string {
	return fmt.Sprintf("bf:%d:%s", blockTime.UnixNano()/100000000, blockID)
}

func (Keyer) TimelineBlockReverse(blockTime time.Time, blockID string) string {
	return fmt.Sprintf("br:%d:%s", maxUnixTimestampDeciSeconds-(blockTime.UnixNano()/100000000), blockID)
}

func (Keyer) ReadTimelineBlockForward(key string) (blockTime time.Time, blockID string, err error) {
	return Keys.ReadTimelineBlock(key, false)
}

func (Keyer) ReadTimelineBlockReverse(key string) (blockTime time.Time, blockID string, err error) {
	return Keys.ReadTimelineBlock(key, true)
}

func (Keyer) ReadTimelineBlock(key string, reversed bool) (blockTime time.Time, blockID string, err error) {
	chunks := strings.Split(key, ":")
	if len(chunks) != 3 {
		err = fmt.Errorf("should have found 3 elements in key %q, found %d", key, len(chunks))
		return
	}
	if reversed && chunks[0] != "br" {
		err = fmt.Errorf("reverse block [%s] key should start with 'br'", key)
		return
	}
	if !reversed && chunks[0] != "bf" {
		err = fmt.Errorf("forward block key [%s] should start with 'bf'", key)
		return
	}

	t, _ := strconv.ParseInt(chunks[1], 10, 64)
	if reversed {
		t = maxUnixTimestampDeciSeconds - t
	}
	ns := (t % 10) * 100000000
	blockTime = time.Unix(t/10, ns)
	blockID = chunks[2]
	return
}

func (Keyer) ReadAccountLink(key string) (creator, created uint64, err error) {
	if !strings.HasPrefix(key, "l:") {
		return 0, 0, errors.New("expected account key starting with 'l:'")
	}

	creator, err = strconv.ParseUint(key[2:10], 16, 64)
	if err != nil {
		return 0, 0, fmt.Errorf("parsing key for accounts table: %s", err)
	}

	created, err = strconv.ParseUint(key[10:], 16, 64)
	if err != nil {
		return 0, 0, fmt.Errorf("parsing key for accounts table: %s", err)
	}

	return
}

// Transactions table

func (Keyer) Transaction(transactionID, blockID string) string {
	return transactionID + ":" + blockID // this should be fixed length
}

func (Keyer) TransactionPrefix(transactionID string) string {
	return transactionID + ":"
}

func (Keyer) ReadTransaction(key string) (transactionID, blockID string, err error) {
	chunks := strings.Split(key, ":")
	if len(chunks) != 2 {
		return "", "", fmt.Errorf("should have found two elements in key %q, found %d", key, len(chunks))
	}

	return chunks[0], chunks[1], nil
}

// Blocks

func (Keyer) Block(blockID string) string {
	return kvdb.ReversedBlockID(blockID)
}

func (Keyer) ReadBlock(key string) (blockID string) {
	return kvdb.ReversedBlockID(key)
}
