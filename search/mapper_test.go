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
	pbcodec "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/codec/v1"
	"github.com/dfuse-io/jsonpb"
	eos "github.com/eoscanada/eos-go"
	"github.com/golang/protobuf/ptypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestPreprocessTokenization_EOS(t *testing.T) {
	tests := []struct {
		name  string
		block *pbcodec.Block
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
		{"dfuse-events-at-input-not-indexed", deosTestBlock(t, "00000001a", nil,
			`{"id":"a1","receipt":{"status":"TRANSACTIONSTATUS_EXECUTED"},
				"action_traces":[{"receipt": {"receiver":"dfuseiohooks"}, "action": {"name":"event","account":"dfuseiohooks","json_data":"{\"data\":\"key=value\"}"}}]
			}`,
		)},
		{"dfuse-events-inline-indexed-at-creator", deosTestBlock(t, "00000001a", nil,
			`{"id":"a1","receipt":{"status":"TRANSACTIONSTATUS_EXECUTED"},
				"action_traces":[
					{"receipt": {"receiver":"any"}, "action": {"name":"event","account":"eosio","json_data":"{}"}, "action_ordinal":1},
					{"receipt": {"receiver":"dfuseiohooks"}, "action": {"name":"event","account":"dfuseiohooks","json_data":"{\"data\":\"key=value\"}"},"action_ordinal":2,"creator_action_ordinal":1}
				]
			}`,
		)},
		{"dfuse-events-deep-inline-indexed-at-creator", deosTestBlock(t, "00000001a", nil,
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
			blockMapper, _ := NewEOSBlockMapper("dfuseiohooks:event", false, "", "")

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

func deosTestBlock(t *testing.T, id string, blockCustomizer func(block *pbcodec.Block), trxTraceJSONs ...string) *pbcodec.Block {
	trxTraces := make([]*pbcodec.TransactionTrace, len(trxTraceJSONs))
	for i, trxTraceJSON := range trxTraceJSONs {
		trxTrace := new(pbcodec.TransactionTrace)
		require.NoError(t, jsonpb.UnmarshalString(trxTraceJSON, trxTrace))

		trxTraces[i] = trxTrace
	}

	pbblock := &pbcodec.Block{
		Id:                id,
		Number:            eos.BlockNum(id),
		TransactionTraces: trxTraces,
	}

	blockTime, err := time.Parse(time.RFC3339, "2006-01-02T15:04:05.5Z")
	require.NoError(t, err)

	blockTimestamp, err := ptypes.TimestampProto(blockTime)
	require.NoError(t, err)

	pbblock.DposIrreversibleBlocknum = pbblock.Number - 1
	pbblock.Header = &pbcodec.BlockHeader{
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

func TestFilterOut(t *testing.T) {
	tests := []struct {
		name         string
		filterOn     string
		filterOut    string
		message      map[string]interface{}
		expectedPass bool
	}{
		{
			"filter nothing",
			"",
			"",
			map[string]interface{}{"account": "whatever"},
			true,
		},
		{
			"filter nothing, with default programs",
			"true",
			"false",
			map[string]interface{}{
				"account": "whatever",
			},
			true,
		},
		{
			"blacklist things FROM badguy",
			`true`,
			`account == "eosio.token" && data.from == "badguy"`,
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
			"blacklist things TO badguy",
			`true`,
			"account == 'eosio.token' && data.to == 'badguy'",
			map[string]interface{}{
				"account": "eosio.token",
				"data": map[string]interface{}{
					"from": "goodguy",
					"to":   "badguy",
				},
			},
			false,
		},
		{
			"blacklist transfers to eidosonecoin",
			"",
			`account == 'eidosonecoin' || receiver == 'eidosonecoin' || (account == 'eosio.token' && (data.to == 'eidosonecoin' || data.from == 'eidosonecoin'))`,
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
			"non-matching identifier in filter-out program doesn't blacklist",
			"",
			`account == 'eosio.token' && data.from == 'broken'`,
			map[string]interface{}{
				"account": "eosio.token",
				"action":  "issue",
				"data": map[string]interface{}{
					"to": "winner",
				},
			},
			true,
		},
		{
			"non-matching identifier in filter-on program still matches",
			`account == 'eosio.token' && data.bob == 'broken'`,
			``,
			map[string]interface{}{
				"account": "eosio.token",
				"action":  "issue",
				"data": map[string]interface{}{
					"to": "winner",
				},
			},
			false,
		},
		{
			"both whitelist and blacklist fail",
			`data.bob == 'broken'`,
			`data.rita == 'rebroken'`,
			map[string]interface{}{
				"data": map[string]interface{}{
					"denise": "winner",
				},
			},
			false,
		},
		{
			"whitelisted but blacklist cleans out",
			`data.bob == '1'`,
			`data.rita == '2'`,
			map[string]interface{}{
				"data": map[string]interface{}{
					"bob":  "1",
					"rita": "2",
				},
			},
			false,
		},
		{
			"whitelisted but blacklist broken so doesn't clean out",
			`data.bob == '1'`,
			`data.broken == 'really'`,
			map[string]interface{}{
				"data": map[string]interface{}{
					"bob": "1",
				},
			},
			true,
		},

		{
			"block receiver",
			"",
			`receiver == "badguy"`,
			map[string]interface{}{
				"receiver": "badguy",
			},
			false,
		},
		{
			"prevent a failure on evaluation, so matches because blacklist fails",
			"",
			`account == "badacct" && has(data.from) && data.from != "badguy"`,
			map[string]interface{}{
				"account":  "badacct",
				"receiver": "badrecv",
				"data":     map[string]interface{}{},
			},
			true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mapper, err := NewEOSBlockMapper("", false, test.filterOn, test.filterOut)
			assert.NoError(t, err)

			assert.Equal(t, test.expectedPass, mapper.shouldIndexAction(test.message))
		})
	}
}

func TestCompileCELPrograms(t *testing.T) {
	_, err := NewEOSBlockMapper("", false, "bro = '", "")
	require.Error(t, err)

	_, err = NewEOSBlockMapper("", false, "", "ken")
	require.Error(t, err)
}
