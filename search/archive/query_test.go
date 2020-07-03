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

package archive

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/blevesearch/bleve/document"
	"github.com/blevesearch/bleve/index"
	"github.com/blevesearch/bleve/index/scorch"
	eosioSearch "github.com/dfuse-io/dfuse-eosio/search"
	"github.com/dfuse-io/dmesh"
	pb "github.com/dfuse-io/pbgo/dfuse/search/v1"
	"github.com/dfuse-io/search"
	searchArchive "github.com/dfuse-io/search/archive"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/metadata"
)

func TestRunQuery(t *testing.T) {
	tmp, err := ioutil.TempDir("", "")
	require.NoError(t, err)
	defer os.RemoveAll(tmp)

	type query struct {
		name string
		in   string
		low  uint64
		high uint64

		sortDesc            bool
		blockInChain        func(blockID string) (bool, error)
		expectedError       error
		expectLastBlockRead uint64
	}
	tests := []struct {
		poolSetup             func(t *testing.T, pool *searchArchive.IndexPool)
		queries               []query
		poolLiveHeadBlockTime time.Time
	}{
		{poolSetup: func(t *testing.T, pool *searchArchive.IndexPool) {
			appendTestIndex(t, tmp, "readonly", pool, 0, 9, "readonly-0", `
trx1:0000 {"block_num": 4, "input": true}
trx2:0000 {"block_num": 7, "data": {"auth": "hello"}}
`)
			appendTestIndex(t, tmp, "readonly", pool, 10, 19, "readonly-1", `
trx3:0000 {"block_num": 14, "input": true}
trx4:0000 {"block_num": 18, "data": {"auth": "hello"}}
`)
			pool.SearchPeer.IrrBlock = 39
			pool.SearchPeer.IrrBlockID = "39a"

		}, queries: []query{
			{
				name: "negative only",
				in:   `-input:true`,
				low:  0,
				high: 19,
			},
			{
				name: "negative and positive",
				in:   `-input:true data.auth:hello`,
				low:  0,
				high: 19,
			},
			{
				name: "negative and positive noresults",
				in:   `-input:true data.auth:nothello`,
				low:  0,
				high: 19,
			},
			{
				name: "two negatives",
				in:   `-input:true -data.auth:nothello`,
				low:  0,
				high: 19,
			},
			{
				name: "nested negatives",
				in:   `(input:nottrue OR -input:true) -data.auth:nothello`,
				low:  0,
				high: 19,
			},
			{
				name: "input true only",
				in:   `input:true`,
				low:  4,
				high: 4,
			},
			{
				name:     "reverse input true",
				in:       `input:true`,
				sortDesc: true,
				low:      4,
				high:     4,
			},
			{
				name: "quoted true expression",
				in:   `input:"true"`,
				low:  1,
				high: 1,
			},
			{
				name: "quoted true expression",
				in:   `input:"true"`,
				low:  1,
				high: 1,
			},
			{
				name: "open limit, but reached block_count",
				in:   `data.auth:hello`,
				low:  17,
				high: 18,
			},
		},
			poolLiveHeadBlockTime: time.Date(2019, 1, 29, 12, 23, 2, 500000000, time.UTC),
		},

		// Negations (FIXME: was status checks)
		{
			poolSetup: func(t *testing.T, pool *searchArchive.IndexPool) {
				appendTestIndex(t, tmp, "readonly", pool, 0, 9, "readonly-0", `
trx1:0000 {"block_num": 4, "trx_idx": 1, "input": true}
trx4:0000 {"block_num": 5, "trx_idx": 4, "action":"transfer", "scheduled": true}
trx9:0000 {"block_num": 7, "trx_idx": 9, "action":"data"}
`)
			},
			queries: []query{
				{
					name: "negated with an OR",
					in:   `-(action:data OR input:true)`,
					low:  0,
					high: 9,
				},
				{
					name: "negate with an AND",
					in:   `-action:data scheduled:true`,
					low:  0,
					high: 9,
				},
			},
		},

		{
			poolSetup: func(t *testing.T, pool *searchArchive.IndexPool) {
				appendTestIndex(t, tmp, "readonly", pool, 0, 9, "readonly-0", `
trx2:0000 {"block_num": 5, "trx_idx": 2, "data": {"auth.keys": [{"key": "EOS6j4hqTnuXdmpcePV9AHr2Av4fxrf3kFiRKJpEbTYbP6ZwJi62h"}]}}
trx3:0000 {"block_num": 6, "trx_idx": 3, "data": {"auth.keys": [{"key": "EOS7j4hqTnuXdmpcePV9AHr2Av4sdaf3kFiRKJpEbTYbP6ZwJi72h"}]}}
`)
			},
			queries: []query{
				{
					name: "nested query on auth.keys.key works",
					in:   `data.auth.keys.key:EOS6j4hqTnuXdmpcePV9AHr2Av4fxrf3kFiRKJpEbTYbP6ZwJi62h`,
					low:  0,
					high: 9,
				},
			},
		},
	}

	// FIXME: use a real path here:
	basePath := "/tmp/tests-run-query"

	for idx, test := range tests {
		t.Run(fmt.Sprintf("index %d", idx+1), func(t *testing.T) {
			os.RemoveAll(basePath)
			os.MkdirAll(basePath, 0755)

			pool := &searchArchive.IndexPool{
				IndexesPath:     filepath.Join(basePath, "indexes"),
				ShardSize:       10,
				PerQueryThreads: 1,
				SearchPeer: &dmesh.SearchPeer{
					BlockRangeData: dmesh.BlockRangeData{
						TailBlock:   0,
						TailBlockID: "00000000a",
						HeadBlockData: dmesh.HeadBlockData{
							IrrBlock:    0,
							IrrBlockID:  "00000000a",
							HeadBlock:   0,
							HeadBlockID: "00000000a",
						},
					},
				},
			}

			test.poolSetup(t, pool)

			client, cleanup := searchArchive.TestNewClient(t, &searchArchive.ArchiveBackend{
				Pool:            pool,
				MaxQueryThreads: 2,
				SearchPeer:      pool.SearchPeer,
			})
			defer cleanup()

			for _, q := range test.queries {
				t.Run(q.name, func(t *testing.T) {
					resp, err := client.StreamMatches(context.Background(), &pb.BackendRequest{
						Query:        q.in,
						LowBlockNum:  q.low,
						HighBlockNum: q.high,
						Descending:   q.sortDesc,
					})

					require.NoError(t, err)

					var lastTrailer metadata.MD
					if q.expectedError != nil {
						_, err := resp.Recv()
						assert.Error(t, err)
						assert.Equal(t, q.expectedError, err)
					} else {
						require.NoError(t, err)

						var out []interface{}
						for {
							el, err := resp.Recv()
							lastTrailer = resp.Trailer()
							if err == io.EOF {
								break
							}
							require.NoError(t, err)
							out = append(out, el)
						}

						actualRaw, _ := json.MarshalIndent(out, "", "  ")
						actual := string(actualRaw)

						goldenFile := queryTestNameToGoldenFile(q.name)
						if os.Getenv("GOLDEN_UPDATE") != "" {
							require.NoError(t, ioutil.WriteFile(goldenFile, actualRaw, os.ModePerm))
						}

						expected := fromFixture(t, goldenFile)

						assert.JSONEqf(t, expected, actual, "Expected:\n%s\n\nActual:\n%s\n", expected, actual)

						var lastBlockRead uint64

						lastBlockReadArray := lastTrailer.Get("last-block-read")
						if len(lastBlockReadArray) > 0 {
							lastBlockRead, _ = strconv.ParseUint(lastBlockReadArray[0], 10, 64)
						}
						if q.expectLastBlockRead != 0 {
							assert.Equal(t, q.expectLastBlockRead, lastBlockRead, "got invalid lastBlockRead")
						}

					}

				})
			}

			pool.CloseIndexes()
		})
	}
}

func fromFixture(t *testing.T, path string) string {
	t.Helper()

	cnt, err := ioutil.ReadFile(path)
	require.NoError(t, err)

	return string(cnt)
}

var queryTestNameNormalizeRegexp = regexp.MustCompile("[^a-z0-9_]")

func queryTestNameToGoldenFile(name string) string {
	normalized := queryTestNameNormalizeRegexp.ReplaceAllString(name, "_")

	return filepath.Join("testdata", "query", normalized+".golden.json")
}

func appendTestIndex(t *testing.T, tmpDir string, typ string, pool *searchArchive.IndexPool, startBlock, endBlock uint64, indexName string, content string) *search.ShardIndex {
	t.Helper()

	var shard *search.ShardIndex
	var err error

	switch typ {
	case "readonly", "merge", "writable":
		// here we only open live index, faster, and it's not the goal to distinguish the
		// types of indexes here... we just want to query them and that'll work with
		// any index state or insertion config.
		path := filepath.Join(tmpDir, fmt.Sprintf("%s.bleve", indexName))
		shard, err = openTestIndex(t, startBlock, endBlock, path)
	default:
		t.Errorf("invalid index type %q", typ)
		t.Fail()
	}
	require.NoError(t, err)

	m, _ := eosioSearch.NewBlockMapper("", false, "*")

	// Analyze `content`, split in blocks, and FEED into the index in the SIMPLEST way possible.
	// Make a batch with those documents, with an `id`.
	// Write to shard
	batch := index.NewBatch()
	for _, line := range strings.Split(content, "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}

		chunks := strings.SplitN(line, " ", 2)

		var data map[string]interface{} // unwrap each line into a JSON doc here
		require.NoError(t, json.Unmarshal([]byte(chunks[1]), &data))

		bNumHex := fmt.Sprintf("%08x", uint64(data["block_num"].(float64)))
		metaDoc := document.NewDocument(bNumHex + ":" + chunks[0])

		// Defaults to `0` for testing
		if _, found := data["trx_idx"]; !found {
			data["trx_idx"] = 0
		}
		if _, found := data["act_idx"]; !found {
			data["act_idx"] = 0
		}

		require.NoError(t, m.MapDocument(metaDoc, data))
		batch.Update(metaDoc)
	}

	require.NoError(t, shard.Batch(batch))

	switch typ {
	case "readonly":
		if len(pool.ReadPool) == 0 {
			pool.LowestServeableBlockNum = startBlock
		}
		pool.ReadPool = append(pool.ReadPool, shard)
	}

	return shard
}

func baseFile(baseBlockNum uint64, suffix string) string {
	panic("you need to implement this")
	return ""
}

func openTestIndex(t *testing.T, start, end uint64, path string) (*search.ShardIndex, error) {
	_ = os.RemoveAll(path)

	idxer, err := scorch.NewScorch("eos", map[string]interface{}{
		"path":         path,
		"unsafe_batch": true,
	}, index.NewAnalysisQueue(2))
	if err != nil {
		return nil, fmt.Errorf("creating ramdisk-based scorch index: %s", err)
	}

	err = idxer.Open()
	if err != nil {
		return nil, fmt.Errorf("opening ramdisk-based scorch index: %s", err)
	}

	_, err = search.NewShardIndexWithAnalysisQueue(start, end-start+1, idxer, baseFile, nil)
	require.Error(t, err, "should not read index without boundaries")

	addBoundaryDocs(t, idxer, start, end)

	idx, err := search.NewShardIndexWithAnalysisQueue(start, end-start+1, idxer, baseFile, nil)
	require.NoError(t, err)

	return idx, nil
}

func addBoundaryDocs(t *testing.T, idx index.Index, start, end uint64) {

	batch := index.NewBatch()
	for _, id := range []string{
		fmt.Sprintf("meta:boundary:end_num:%d", end),
		"meta:boundary:end_id:0000000a",
		fmt.Sprintf("meta:boundary:end_time:%s", time.Now().UTC().Format(search.TimeFormatBleveID)),
		fmt.Sprintf("meta:boundary:start_num:%d", start),
		"meta:boundary:start_id:00000014",
		fmt.Sprintf("meta:boundary:start_time:%s", time.Now().UTC().Format(search.TimeFormatBleveID)),
	} {
		batch.Update(document.NewDocument(id))

	}

	err := idx.Batch(batch)
	require.NoError(t, err)
}
