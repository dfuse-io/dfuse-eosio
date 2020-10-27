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
	"testing"
	"time"

	pbstatedb "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/statedb/v1"
	"github.com/dfuse-io/dstore"
	"github.com/stretchr/testify/assert"
)

func Test_onGetHeadInfo(t *testing.T) {
	t.Skip("get from somewhere")

	stateClient := pbstatedb.NewMockStateClient()
	blockStore, err := dstore.NewDBinStore("gs://example/blocks")
	assert.NoError(t, err)

	cases := []struct {
		name           string
		msg            string
		expectedOutput []string
		expectedError  string
	}{
		{
			name:           "sunny path",
			msg:            `{"type":"get_head_info","req_id":"abc","listen":true,"fetch":true}`,
			expectedOutput: []string{},
		},
	}

	for _, c := range cases {

		t.Run(c.name, func(t *testing.T) {

			subscriptionHub := newTestSubscriptionHub(t, 0, blockStore)
			headInfoHub := NewHeadInfoHub("00000002a", "00000001a", subscriptionHub)
			go headInfoHub.Launch(context.Background())

			handler := NewWebsocketHandler(
				nil,
				nil,
				nil,
				subscriptionHub,
				stateClient,
				nil,
				headInfoHub,
				nil,
				NewTestIrreversibleFinder("00000002a", nil),
				0,
				12,
			)

			conn, closer := newTestConnection(t, handler)
			defer closer()

			conn.WriteMessage(1, []byte(c.msg))

			validateOutput(t, "", c.expectedOutput, conn, 5*time.Second)

		})
	}
}
