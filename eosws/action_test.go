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
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/dfuse-io/bstream"
	"github.com/dfuse-io/bstream/forkable"
	"github.com/dfuse-io/bstream/hub"
	"github.com/dfuse-io/dauth"
	"github.com/dfuse-io/dfuse-eosio/codecs/deos"
	"github.com/dfuse-io/dfuse-eosio/eosws/wsmsg"
	fluxdb "github.com/dfuse-io/dfuse-eosio/fluxdb-client"
	pbeos "github.com/dfuse-io/dfuse-eosio/pb/dfuse/codecs/eos"
	"github.com/dfuse-io/dstore"
	pbbstream "github.com/dfuse-io/pbgo/dfuse/bstream/v1"
	eos "github.com/eoscanada/eos-go"
	"github.com/golang/protobuf/ptypes"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/sjson"
)

func TestOnGetActionsTraces(t *testing.T) {
	statusExecuted := pbeos.TransactionStatus_TRANSACTIONSTATUS_EXECUTED
	statusHardFail := pbeos.TransactionStatus_TRANSACTIONSTATUS_HARDFAIL

	rand.Seed(time.Now().UnixNano())
	actionTracesMsg := func(reqID string, data string) string {
		return fmt.Sprintf(`{"type":"get_action_traces","req_id":%q,"listen":true,"start_block":2,"data":%s}`, reqID, data)
	}

	listeningResp := func(reqID string) string {
		return fmt.Sprintf(`{"type":"listening","req_id":%q,"data":{"next_block":2}}`, reqID)
	}

	actionTraceRespWithBlock := func(blockID string, reqID string, index int, trace string) string {
		ref := bstream.BlockRefFromID(blockID)

		out := fmt.Sprintf(`{"type":"action_trace","req_id":%q,"data":{"block_num":%d,"block_id":"%s","block_time":"0001-01-01T00:00:00Z","trx_id":"trx.1","idx":%d,"trace":%s}`, reqID, ref.Num(), blockID, index, trace)
		out, _ = sjson.SetRaw(out, "data.trace.closest_unnotified_ancestor_action_ordinal", "0")
		out, _ = sjson.SetRaw(out, "data.trace.console", `""`)
		out, _ = sjson.SetRaw(out, "data.trace.block_num", `0`)
		out, _ = sjson.SetRaw(out, "data.trace.creator_action_ordinal", `0`)
		out, _ = sjson.SetRaw(out, "data.trace.action_ordinal", `0`)
		out, _ = sjson.SetRaw(out, "data.trace.context_free", `false`)
		out, _ = sjson.SetRaw(out, "data.trace.account_ram_deltas", `[]`)
		out, _ = sjson.SetRaw(out, "data.trace.producer_block_id", `""`)
		out, _ = sjson.SetRaw(out, "data.trace.trx_id", `""`)
		out, _ = sjson.SetRaw(out, "data.trace.elapsed", `0`)
		out, _ = sjson.SetRaw(out, "data.trace.except", `null`)
		out, _ = sjson.SetRaw(out, "data.trace.error_code", `null`)
		out, _ = sjson.SetRaw(out, "data.trace.block_time", `"1970-01-01T00:00:00"`)
		return out
	}

	actionTraceResp := func(reqID string, index int, trace string) string {
		return actionTraceRespWithBlock("00000002a", reqID, index, trace)
	}

	tests := []struct {
		name               string
		blocks             []archiveFiles
		listenData         string
		expectedMsgFactory func(reqID string) []string
	}{
		{
			name: "multi accounts, multi action_names",
			blocks: []archiveFiles{{"0000000000", acceptedBlockWithActions(t, "00000002a", statusExecuted,
				"eosioknights:eosioknights:transfer",
				"eosioknights:eosioknights:rebirth",
				"eosiobuddies:eosiobuddies:transfer",
				"eosioknights:eosioknights:rebirth",
			)}},
			listenData: `{"accounts":"eosioknights|eosiobuddies","action_names":"transfer"}`,
			expectedMsgFactory: func(reqID string) []string {
				return []string{
					listeningResp(reqID),
					actionTraceResp(reqID, 0, `{"inline_traces":[],"receiver":"eosioknights","act":{"account":"eosioknights","name":"transfer","authorization":[]}}}`),
					actionTraceResp(reqID, 2, `{"inline_traces":[],"receiver":"eosiobuddies","act":{"account":"eosiobuddies","name":"transfer","authorization":[]}}}`),
				}
			},
		},
		{
			name: "multi receivers, multi action_names",
			blocks: []archiveFiles{{"0000000000", acceptedBlockWithActions(t, "00000002a", statusExecuted,
				"eosioknights:eosiofriends:transfer",
				"eosiofriends:eosiofriends:transfer",
				"eosiobuddies:eosiofriends:transfer",
				"eosiobuddies:eosiofriends:rebirth",
			)}},
			listenData: `{"accounts":"eosiofriends","receivers":"eosioknights|eosiobuddies","action_names":"transfer"}`,
			expectedMsgFactory: func(reqID string) []string {
				return []string{
					listeningResp(reqID),
					actionTraceResp(reqID, 0, `{"inline_traces":[],"receiver":"eosioknights","act":{"account":"eosiofriends","name":"transfer","authorization":[]}}}`),
					actionTraceResp(reqID, 2, `{"inline_traces":[],"receiver":"eosiobuddies","act":{"account":"eosiofriends","name":"transfer","authorization":[]}}}`),
				}
			},
		},
		{
			name: "receivers inferred",
			blocks: []archiveFiles{{"0000000000", acceptedBlockWithActions(t, "00000002a", statusExecuted,
				"eosioknights:eosiofriends:transfer",
				"eosiofriends:eosiofriends:transfer",
				"eosiobuddies:eosiofriends:transfer",
				"eosiobuddies:eosiofriends:rebirth",
			)}},
			listenData: `{"accounts":"eosiofriends"}`,
			expectedMsgFactory: func(reqID string) []string {
				return []string{
					listeningResp(reqID),
					actionTraceResp(reqID, 1, `{"inline_traces":[],"receiver":"eosiofriends","act":{"account":"eosiofriends","name":"transfer","authorization":[]}}}`),
				}
			},
		},
		{
			name: "match all single",
			blocks: []archiveFiles{{"0000000000", acceptedBlockWithActions(t, "00000002a", statusExecuted,
				"eosioknights:eosiofriends:transfer",
				"eosiofriends:eosiofriends:transfer",
				"eosiofriends:eosiofriends:issue",
				"eosiobuddies:eosiofriends:transfer",
				"eosiobuddies:eosiofriends:rebirth",
			)}},
			listenData: `{"receivers":"eosiofriends","accounts":"eosiofriends","action_names":"transfer"}`,
			expectedMsgFactory: func(reqID string) []string {
				fmt.Printf("For test2 %q, reqID is %q\n", "match all single", reqID)

				return []string{
					listeningResp(reqID),
					actionTraceResp(reqID, 1, `{"inline_traces":[],"receiver":"eosiofriends","act":{"account":"eosiofriends","name":"transfer","authorization":[]}}}`),
				}
			},
		},
		{
			name: "stream only executed",
			blocks: []archiveFiles{
				{"0000000000", acceptedBlockWithActions(t, "00000002a", statusHardFail,
					"eosioknights:eosioforlife:transfer",
					"eosiofriends:eosiofriends:failure",
				)},
				{"0000000100", acceptedBlockWithActions(t, "00000003a", statusExecuted,
					"eosiofriends:eosiofriends:transfer",
					"eosioknights:eosioforlife:transfer",
				)}},
			listenData: `{"receivers":"eosiofriends","accounts":"eosiofriends","action_names":"transfer|failure"}`,
			expectedMsgFactory: func(reqID string) []string {
				return []string{
					listeningResp(reqID),
					actionTraceRespWithBlock("00000003a", reqID, 0, `{"inline_traces":[],"receiver":"eosiofriends","act":{"account":"eosiofriends","name":"transfer","authorization":[]}}}`),
				}
			},
		},
		{
			name: "match all multiple",
			blocks: []archiveFiles{{"0000000000", acceptedBlockWithActions(t, "00000002a", statusExecuted,
				"eosioknights:eosioforlife:transfer",
				"eosioforlife:eosioforlife:transfer",
				"eosioforlife:eosioforlife:issue",
				"eosio.system:eosio.system:transfer",
				"eosio.system:eosio.system:issue",
				"eosiobuddies:eosioforlife:transfer",
				"eosiobuddies:eosioforlife:rebirth",
			)}},
			listenData: `{"receivers":"eosioforlife|eosio.system","accounts":"eosioforlife|eosio.system","action_names":"transfer|issue"}`,
			expectedMsgFactory: func(reqID string) []string {
				fmt.Printf("For test1 %q, reqID is %q\n", "match all multiple", reqID)
				return []string{
					listeningResp(reqID),
					actionTraceResp(reqID, 1, `{"inline_traces":[],"receiver":"eosioforlife","act":{"account":"eosioforlife","name":"transfer","authorization":[]}}}`),
					actionTraceResp(reqID, 2, `{"inline_traces":[],"receiver":"eosioforlife","act":{"account":"eosioforlife","name":"issue","authorization":[]}}}`),
					actionTraceResp(reqID, 3, `{"inline_traces":[],"receiver":"eosio.system","act":{"account":"eosio.system","name":"transfer","authorization":[]}}}`),
					actionTraceResp(reqID, 4, `{"inline_traces":[],"receiver":"eosio.system","act":{"account":"eosio.system","name":"issue","authorization":[]}}}`),
				}
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			archiveStore := dstore.NewMockStore(nil)

			subscriptionHub := newTestSubscriptionHub(t, 0, archiveStore)
			fluxClient := fluxdb.NewTestFluxClient()
			handler := NewWebsocketHandler(
				nil,
				nil,
				nil,
				subscriptionHub,
				fluxClient,
				nil,
				nil,
				nil,
				NewTestIrreversibleFinder("00000001a", nil),
				0,
			)

			conn, closer := newTestConnection(t, handler, &testCredentials{startBlock: 2})
			// dauth.Credentials{
			// 	StandardClaims: jwt.StandardClaims{Id: "testID"},
			// 	Version:        1,
			// 	Tier:           "beta-v1",
			// 	StartBlock:     2,
			// }
			defer closer()

			for _, f := range test.blocks {
				archiveStore.SetFile(f.name, f.content)
			}

			reqID := strconv.Itoa(rand.Int())
			err := conn.WriteMessage(1, []byte(actionTracesMsg(reqID, test.listenData)))
			require.NoError(t, err)
			go subscriptionHub.Launch()

			validateOutput(t, reqID, test.expectedMsgFactory(reqID), conn)
		})
	}
}

type TestPipelineInitiator struct {
	pipeline bstream.Pipeline
}

func (m *TestPipelineInitiator) NewPipeline(
	ctx context.Context,
	startBlockID string,
	startBlockNum uint32,
	emissionStartBlock uint32,
	originMsg wsmsg.IncomingMessager,
	pipeline bstream.Pipeline,
	onClose func(),
) {
	m.pipeline = pipeline
}

func (m *TestPipelineInitiator) pushBlock(block *bstream.Block, obj *forkable.ForkableObject) {
	m.pipeline.ProcessBlock(block, obj)
}

type archiveFiles struct {
	name    string
	content []byte
}

type ABIAccountName struct {
	accountName eos.AccountName
	abiString   string
}

type traceIDTestOpion string

var defaultTraceID = "105445aa7843bc8bf206b12000100000"

var defaultTestCredentials = &testCredentials{startBlock: 0}

// dauth.Credentials{
// 	StandardClaims: jwt.StandardClaims{Id: "testID"},
// 	Version:        1,
// 	Tier:           "beta-v1",
// 	StartBlock:     0,
// }

func newTestConnection(t *testing.T, handler http.Handler, options ...interface{}) (*websocket.Conn, func()) {
	credentials := dauth.Credentials(defaultTestCredentials)
	traceID := defaultTraceID

	for _, option := range options {
		switch value := option.(type) {
		case dauth.Credentials:
			credentials = value
		case traceIDTestOpion:
			traceID = string(value)
		}
	}

	traceMiddleware := OpenCensusMiddleware

	// Adds the required authKey to request context
	testMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r.WithContext(dauth.WithCredentials(r.Context(), credentials)))
		})
	}

	srv := httptest.NewServer(traceMiddleware(testMiddleware(handler)))

	u, _ := url.Parse(srv.URL)
	u.Scheme = "ws"
	u.Path = "/v1/stream"
	u.RawQuery = "token="

	headers := http.Header{}
	headers.Add("Origin", "http://www.examples.com")
	headers.Add("X-Cloud-Trace-Context", traceID+"/123")
	conn, _, err := websocket.DefaultDialer.Dial(u.String(), headers)
	require.NoError(t, err)
	return conn, func() {
		conn.Close()
		srv.Close()
	}
}

func validateOutput(t *testing.T, reqID string, expectedOutput []string, conn *websocket.Conn) {
	for _, expected := range expectedOutput {
		output := nextMessage(t, reqID, conn)

		// Let's not check content of message that are ignored
		//fmt.Println("expected: ", expected)
		if expected != "_" {
			require.JSONEq(t, expected, output)
		}
	}
}

func nextMessage(t *testing.T, reqID string, conn *websocket.Conn) string {
	message := ""
	for {
		_, r, err := conn.NextReader()
		rawOutput, err := ioutil.ReadAll(r)
		require.NoError(t, err)

		output := string(rawOutput)
		if reqID == "" || strings.Contains(output, fmt.Sprintf(`"req_id":%q`, reqID)) {
			message = output
			break
		}
	}

	return message
}

func newTestSubscriptionHub(t *testing.T, startBlock uint32, archiveStore dstore.Store) *hub.SubscriptionHub {
	t.Helper()

	fileSourceFactory := bstream.SourceFromNumFactory(func(startBlockNum uint64, h bstream.Handler) bstream.Source {
		return bstream.NewFileSource(pbbstream.Protocol_EOS, archiveStore, 1, 1, nil, h)
	})

	liveSourceFactory := bstream.SourceFromNumFactory(func(startBlockNum uint64, h bstream.Handler) bstream.Source {
		return bstream.NewTestSource(h)
	})

	buf := bstream.NewBuffer("pubsubbuf")
	tailManager := bstream.NewSimpleTailManager(buf, 10)
	subscriptionHub, err := hub.NewSubscriptionHub(uint64(startBlock), buf, tailManager.TailLock, fileSourceFactory, liveSourceFactory)
	require.NoError(t, err)
	return subscriptionHub
}

func encode(t *testing.T, i interface{}) []byte {
	t.Helper()

	buf := new(bytes.Buffer)
	enc := eos.NewEncoder(buf)
	require.NoError(t, enc.Encode(i))

	return buf.Bytes()
}

func acceptedBlockWithActions(t *testing.T, blockID string, status pbeos.TransactionStatus, actionTriplets ...string) []byte {
	t.Helper()

	var actTraces []*pbeos.ActionTrace
	for _, actionTriplet := range actionTriplets {
		parts := strings.Split(actionTriplet, ":")
		actTraces = append(actTraces, &pbeos.ActionTrace{
			Receiver: parts[0],
			Action: &pbeos.Action{
				Account: parts[1],
				Name:    parts[2],
			},
		})
	}

	stamp, _ := ptypes.TimestampProto(time.Time{})

	ref := bstream.BlockRefFromID(blockID)
	blk := &pbeos.Block{
		Id:     blockID,
		Number: uint32(ref.Num()),
		Header: &pbeos.BlockHeader{
			Previous:  fmt.Sprintf("%08d", ref.Num()-1) + ref.ID()[8:],
			Timestamp: stamp,
		},
		TransactionTraces: []*pbeos.TransactionTrace{
			{
				Id: "trx.1",
				Receipt: &pbeos.TransactionReceiptHeader{
					Status: status,
				},
				ActionTraces: actTraces,
			},
		},
	}

	return pbeosBlockToFile(t, blk)
}

func pbeosBlockToFile(t *testing.T, in *pbeos.Block) []byte {
	t.Helper()

	var buf bytes.Buffer

	w, _ := deos.NewBlockWriter(&buf)
	blk, err := deos.BlockFromProto(in)
	require.NoError(t, err)

	err = w.Write(blk)
	require.NoError(t, err)

	return buf.Bytes()
}
