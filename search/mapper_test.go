package search

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"github.com/andreyvit/diff"
	pbeos "github.com/dfuse-io/dfuse-eosio/pb/dfuse/codecs/eos"
	eos "github.com/eoscanada/eos-go"
	"github.com/dfuse-io/jsonpb"
	"github.com/golang/protobuf/ptypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestPreprocessTokenization_EOS(t *testing.T) {
	tests := []struct {
		name  string
		block *pbeos.Block
	}{
		{"standard-block", deosTestBlock(t, "00000001a", nil,
			`{"id":"a1","receipt":{"status":"TRANSACTIONSTATUS_EXECUTED"},"action_traces":[
				{"receipt":{"receiver":"battlefield1"},"action":{"name":"transfer","account":"eosio","json_data":"{\"to\":\"eosio\"}"}}
			]}`,
			`{"id":"a2","receipt":{"status":"TRANSACTIONSTATUS_EXECUTED"},"action_traces":[
				{"receipt":{"receiver":"other"},"action":{"name":"random","account":"account","json_data":"{\"proposer\":\"eosio\"}"}}
			]}`,
		)},
		{"auth-keys", deosTestBlock(t, "00000001a", nil,
			`{"id":"a1","receipt":{"status":"TRANSACTIONSTATUS_EXECUTED"},"action_traces":[
				{
					"receipt": {"receiver":"battlefield1"},
					"action": {
					  "name":"transfer",
					  "account":"eosio",
					  "json_data":"{\"auth\":{\"accounts\":[],\"keys\":[{\"key\":\"EOS6j4hqTnuXdmpcePV9AHr2Av4fxrf3kFiRKJpEbTYbP6ZwJi62h\",\"weight\":1}],\"threshold\":1,\"waits\":[]}}"
					}
				}
			]}`,
		)},
		{"on-blocks", deosTestBlock(t, "00000001a", nil,
			`{"id":"a1","receipt":{"status":"TRANSACTIONSTATUS_EXECUTED"},
				"action_traces":[{"receipt": {"receiver":"eosio"}, "action": {"name":"transfer","account":"eosio","json_data":""}}],
				"db_ops":[
					{"code": "eosio", "scope": "eosio", "table_name": "producers", "primary_key": "eoshuobipool"},
					{"code": "eosio", "scope": "eosio", "table_name": "namebids", "primary_key": "j"},
					{"code": "eosio", "scope": "eosio", "table_name": "global", "primary_key": "global"},
					{"code": "eosio", "scope": "eosio", "table_name": "global2", "primary_key": "global2"},
					{"code": "eosio", "scope": "eosio", "table_name": "global3", "primary_key": "global3"}
				]
			}`,
		)},
		{"dtrx-onerror-soft-fail", deosTestBlock(t, "00000001a", nil,
			`{"id":"a1","receipt":{"status":"TRANSACTIONSTATUS_SOFTFAIL"},
				"action_traces":[{"receipt": {"receiver":"any"}, "action": {"name":"onerror","account":"eosio","json_data":""}}],
				"db_ops":[
					{"code": "eosio", "scope": "eosio", "table_name": "producers", "primary_key": "eoshuobipool"}
				]
			}`,
			`{"id":"a2","receipt":{"status":"TRANSACTIONSTATUS_SOFTFAIL"},
				"action_traces":[{"receipt": {"receiver":"any"}, "action": {"name":"dtrexec","account":"any","json_data":"{\"to\":\"toaccount\"}"}}],
				"ram_ops":[
					{"namespace": "NAMESPACE_DEFERRED_TRX", "action": "ACTION_REMOVE"}
				]
			}`,
		)},
		{"dtrx-onerror-hard-fail", deosTestBlock(t, "00000001a", nil,
			`{"id":"a1","receipt":{"status":"TRANSACTIONSTATUS_HARDFAIL"},
				"action_traces":[{"receipt": {"receiver":"any"}, "action": {"name":"onerror","account":"eosio","json_data":""}}],
				"db_ops":[
					{"code": "eosio", "scope": "eosio", "table_name": "producers", "primary_key": "eoshuobipool"}
				]
			}`,
			`{"id":"a2","receipt":{"status":"TRANSACTIONSTATUS_SOFTFAIL"},
				"action_traces":[{"receipt": {"receiver":"any"}, "action": {"name":"dtrexec","account":"any","json_data":"{\"to\":\"toaccount\"}"}}],
				"ram_ops":[
					{"namespace": "NAMESPACE_DEFERRED_TRX", "action": "ACTION_REMOVE"}
				]
			}`,
		)},
		{"dtrx-soft-fail-wrong-onerror", deosTestBlock(t, "00000001a", nil,
			`{"id":"a1","receipt":{"status":"TRANSACTIONSTATUS_SOFTFAIL"},
				"action_traces":[{"receipt": {"receiver":"any"}, "action": {"name":"onerror","account":"any","json_data":"{\"to\":\"toaccount\"}"}}],
				"db_ops":[
					{"code": "eosio", "scope": "eosio", "table_name": "producers", "primary_key": "eoshuobipool"}
				]
			}`,
		)},
		{"dfuse-hooks-at-input-not-indexed", deosTestBlock(t, "00000001a", nil,
			`{"id":"a1","receipt":{"status":"TRANSACTIONSTATUS_EXECUTED"},
				"action_traces":[{"receipt": {"receiver":"dfuseiohooks"}, "action": {"name":"event","account":"dfuseiohooks","json_data":"{\"data\":\"key=value\"}"}}]
			}`,
		)},
		{"dfuse-hooks-inline-indexed-at-creator", deosTestBlock(t, "00000001a", nil,
			`{"id":"a1","receipt":{"status":"TRANSACTIONSTATUS_EXECUTED"},
				"action_traces":[
					{"receipt": {"receiver":"any"}, "action": {"name":"event","account":"eosio","json_data":"{}"}, "action_ordinal":1},
					{"receipt": {"receiver":"dfuseiohooks"}, "action": {"name":"event","account":"dfuseiohooks","json_data":"{\"data\":\"key=value\"}"},"action_ordinal":2,"creator_action_ordinal":1}
				]
			}`,
		)},
		{"dfuse-hooks-deep-inline-indexed-at-creator", deosTestBlock(t, "00000001a", nil,
			`{"id":"a1","receipt":{"status":"TRANSACTIONSTATUS_EXECUTED"},
				"action_traces":[
					{"receipt": {"receiver":"any"}, "action": {"name":"topevent","account":"eosio","json_data":"{}"}, "action_ordinal":1},
					{"receipt": {"receiver":"any"}, "action": {"name":"childevent","account":"eosio","json_data":"{}"}, "action_ordinal":2,"creator_action_ordinal":1},
					{"receipt": {"receiver":"dfuseiohooks"}, "action": {"name":"event","account":"dfuseiohooks","json_data":"{\"data\":\"key=value\"}"},"action_ordinal":3,"creator_action_ordinal":2}
				]
			}`,
		)},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			blockMapper, _ := NewEOSBlockMapper("dfuseiohooks:event", "")

			goldenFilePath := filepath.Join("testdata", test.name+".golden.json")

			coll := &eosDocCollection{}
			err := blockMapper.prepareBatchDocuments(test.block, coll.update)
			require.NoError(t, err)

			cnt, err := json.MarshalIndent(coll.docs, "", "  ")
			require.NoError(t, err)

			_, err = os.Stat(goldenFilePath)

			if os.IsNotExist(err) || os.Getenv("GOLDEN_UPDATE") != "" {
				ioutil.WriteFile(goldenFilePath, cnt, os.ModePerm)
			}

			actual := string(cnt)
			expected := fromFixture(t, goldenFilePath)

			assert.JSONEq(t, expected, actual, diff.LineDiff(expected, actual))
		})
	}
}

func toData(value string) []byte {
	data, err := hex.DecodeString(value)
	if err != nil {
		panic(err)
	}

	return data
}

type eosDocCollection struct {
	docs []*eosParsedDoc
}

func (c *eosDocCollection) update(trxID string, idx int, data map[string]interface{}) error {
	c.docs = append(c.docs, &eosParsedDoc{
		TrxID: trxID,
		Index: idx,
		Data:  data,
	})
	sort.Slice(c.docs, func(i, j int) bool {
		return c.docs[i].Index < c.docs[j].Index
	})

	return nil
}

type eosParsedDoc struct {
	TrxID string                 `json:"trx_id"`
	Index int                    `json:"-"` // hrm.. would need to fix tests
	Data  map[string]interface{} `json:"data"`
}

func fromFixture(t *testing.T, path string) string {
	t.Helper()

	cnt, err := ioutil.ReadFile(path)
	require.NoError(t, err)

	return string(cnt)
}

func deosTestBlock(t *testing.T, id string, blockCustomizer func(block *pbeos.Block), trxTraceJSONs ...string) *pbeos.Block {
	trxTraces := make([]*pbeos.TransactionTrace, len(trxTraceJSONs))
	for i, trxTraceJSON := range trxTraceJSONs {
		trxTrace := new(pbeos.TransactionTrace)
		require.NoError(t, jsonpb.UnmarshalString(trxTraceJSON, trxTrace))

		trxTraces[i] = trxTrace
	}

	pbblock := &pbeos.Block{
		Id:                id,
		Number:            eos.BlockNum(id),
		TransactionTraces: trxTraces,
	}

	blockTime, err := time.Parse(time.RFC3339, "2006-01-02T15:04:05.5Z")
	require.NoError(t, err)

	blockTimestamp, err := ptypes.TimestampProto(blockTime)
	require.NoError(t, err)

	pbblock.DposIrreversibleBlocknum = pbblock.Number - 1
	pbblock.Header = &pbeos.BlockHeader{
		Previous:  fmt.Sprintf("%08d%s", pbblock.Number-1, id[8:]),
		Producer:  "tester",
		Timestamp: blockTimestamp,
	}

	if blockCustomizer != nil {
		blockCustomizer(pbblock)
	}

	if os.Getenv("DEBUG") != "" {
		marshaler := &jsonpb.Marshaler{}
		out, err := marshaler.MarshalToString(pbblock)
		require.NoError(t, err)

		// We re-normalize to a plain map[string]interface{} so it's printed as JSON and not a proto default String implementation
		normalizedOut := map[string]interface{}{}
		require.NoError(t, json.Unmarshal([]byte(out), &normalizedOut))

		zlog.Debug("created test block", zap.Any("block", normalizedOut))
	}

	return pbblock
}

func TestParseRestrictionsJSON(t *testing.T) {
	// very shallow test, but we dont want to test actual golang JSON unmarshalling,
	// just the general format of our restrictions
	emptyRests, err := parseRestrictionsJSON("")
	assert.NoError(t, err)
	require.Len(t, emptyRests, 0)

	rests, err := parseRestrictionsJSON(`[{"account":"eidosonecoin"},{"receiver":"eidosonecoin"},{"account":"eosio.token","data.to":"eidosonecoin"},{"account":"eosio.token","data.from":"eidosonecoin"}]`)
	require.NoError(t, err)
	assert.Len(t, rests, 4)
}

func TestRestrictions(t *testing.T) {
	tests := []struct {
		name         string
		restriction  *restriction
		message      map[string]interface{}
		expectedPass bool
	}{
		{
			"let pass eosio transfer",
			&restriction{
				"account":   "eosio.token",
				"data.from": "badguy",
			},
			map[string]interface{}{
				"account": "eosio.token",
				"data": map[string]interface{}{
					"from": "goodguy",
					"to":   "badguy",
				},
			},
			true,
		},
		{
			"prevent eosio transfer to eidosonecoin",
			&restriction{
				"account": "eosio.token",
				"data.to": "eidosonecoin",
			},
			map[string]interface{}{
				"account": "eosio.token",
				"data": map[string]interface{}{
					"from": "goodguy",
					"to":   "eidosonecoin",
				},
			},
			false,
		},
		{
			"block receiver",
			&restriction{
				"receiver": "badguy",
			},
			map[string]interface{}{
				"receiver": "badguy",
			},
			false,
		},
		{
			"pass with data restriction on no data",
			&restriction{
				"account":   "badacct",
				"data.from": "badguy",
			},
			map[string]interface{}{
				"account":  "badacct",
				"receiver": "badrecv",
			},
			true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.expectedPass, test.restriction.Pass(test.message))
		})
	}

}
