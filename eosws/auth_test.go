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
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/dfuse-io/dfuse-eosio/eosws/wsmsg"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type testCredentials struct {
	startBlock int64
}

func (c *testCredentials) GetLogFields() []zap.Field {
	return nil
}

func (c *testCredentials) AuthenticatedStartBlock() int64 {
	return c.startBlock
}

func TestAuthRequest(t *testing.T) {
	tests := []struct {
		authStartBlock   int64
		startBlock       int64
		currentBlock     string
		expectedAbsBlock uint32
		expectedError    bool
	}{
		{-1000, -1000, forgeBlockID(10000), 9000, false},
		{-1000, -1001, forgeBlockID(10000), 0, true},
		{-1000, 9000, forgeBlockID(10000), 9000, false},
		{-1000, 8999, forgeBlockID(10000), 0, true},
		{9000, 9000, forgeBlockID(10000), 9000, false},
		{9000, 8999, forgeBlockID(10000), 0, true},
		{9000, -1000, forgeBlockID(10000), 9000, false},
		{9000, -1001, forgeBlockID(10000), 9000, true},
	}

	for idx, test := range tests {
		t.Run(fmt.Sprintf("test %d", idx), func(t *testing.T) {
			ws := &WSConn{
				creds:            &testCredentials{startBlock: test.authStartBlock},
				Context:          context.Background(),
				WebsocketHandler: &WebsocketHandler{},
			}
			authReq, err := ws.authorizeRequest(wsmsg.CommonIn{StartBlock: test.startBlock}, test.currentBlock)

			if test.expectedError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, test.expectedAbsBlock, authReq.StartBlockNum)
			}
		})

	}
}

func forgeBlockID(blockNum uint32) string {
	data := make([]byte, 4)
	binary.BigEndian.PutUint32(data, blockNum)
	id := hex.EncodeToString(data)
	return id
}
