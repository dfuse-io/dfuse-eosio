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
	"strings"
	"testing"

	"github.com/dfuse-io/dfuse-eosio/codec/eosio"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_computeCreationTree_empty(t *testing.T) {
	var ops []*creationOp
	var emptyNodes nodes

	nodes, err := computeCreationTree(ops)
	require.NoError(t, err)

	assert.Equal(t, emptyNodes, nodes)
}

func Test_computeCreationTree_singleRoot(t *testing.T) {
	ops := []*creationOp{
		{"ROOT", 0},
	}

	assertCreationTreeForOps(t, ops, `
		| (0, ROOT)
	`)
}

func Test_computeCreationTree_singleRoot_oneLevel(t *testing.T) {
	ops := []*creationOp{
		{"ROOT", 0},
		{"NOTIFY", 0},
		{"CFA_INLINE", 0},
		{"INLINE", 0},
		{"CFA_INLINE", 0},
		{"NOTIFY", 0},
		{"INLINE", 0},
	}

	assertCreationTreeForOps(t, ops, `
		| (0, ROOT)
		|   (1, NOTIFY)
		|   (3, CFA_INLINE)
		|   (5, INLINE)
		|   (4, CFA_INLINE)
		|   (2, NOTIFY)
		|   (6, INLINE)
	`)
}

func Test_computeCreationTree_singleRoot_multiLevel_inline(t *testing.T) {
	ops := []*creationOp{
		{"ROOT", 0},
		{"NOTIFY", 0},
		{"CFA_INLINE", 0},
		{"INLINE", 0},
		{"CFA_INLINE", 0},
		{"NOTIFY", 0},
		{"INLINE", 0},
		{"NOTIFY", 5},
		{"CFA_INLINE", 5},
		{"INLINE", 5},
	}

	assertCreationTreeForOps(t, ops, `
		| (0, ROOT)
		|   (1, NOTIFY)
		|   (3, CFA_INLINE)
		|   (5, INLINE)
		|     (6, NOTIFY)
		|     (7, CFA_INLINE)
		|     (8, INLINE)
		|   (4, CFA_INLINE)
		|   (2, NOTIFY)
		|   (9, INLINE)
	`)
}

func Test_computeCreationTree_singleRoot_multiLevel_notify(t *testing.T) {
	ops := []*creationOp{
		{"ROOT", 0},
		{"NOTIFY", 0},
		{"CFA_INLINE", 0},
		{"INLINE", 0},
		{"CFA_INLINE", 0},
		{"NOTIFY", 0},
		{"INLINE", 0},
		{"NOTIFY", 2},
		{"CFA_INLINE", 2},
		{"INLINE", 2},
	}

	assertCreationTreeForOps(t, ops, `
		| (0, ROOT)
		|   (1, NOTIFY)
		|   (4, CFA_INLINE)
		|   (7, INLINE)
		|   (5, CFA_INLINE)
		|   (2, NOTIFY)
		|     (3, NOTIFY)
		|     (6, CFA_INLINE)
		|     (9, INLINE)
		|   (8, INLINE)
	`)
}

func Test_computeCreationTree_singleRoot_multiLevel(t *testing.T) {
	ops := []*creationOp{
		{"ROOT", 0},
		{"NOTIFY", 0},
		{"NOTIFY", 0},
		{"NOTIFY", 0},
		{"CFA_INLINE", 0},
		{"INLINE", 0},
		{"INLINE", 2},
		{"CFA_INLINE", 2},
		{"NOTIFY", 2},
		{"NOTIFY", 4},
		{"NOTIFY", 8},
		{"NOTIFY", 8},
		{"INLINE", 8},
		{"CFA_INLINE", 8},
		{"NOTIFY", 13},
		{"CFA_INLINE", 13},
		{"NOTIFY", 13},
	}

	assertCreationTreeForOps(t, ops, `
		| (0, ROOT)
		|   (1, NOTIFY)
		|   (2, NOTIFY)
		|     (13, INLINE)
		|       (14, NOTIFY)
		|       (16, CFA_INLINE)
		|       (15, NOTIFY)
		|     (7, CFA_INLINE)
		|     (4, NOTIFY)
		|       (5, NOTIFY)
		|   (3, NOTIFY)
		|   (6, CFA_INLINE)
		|   (8, INLINE)
		|     (9, NOTIFY)
		|     (10, NOTIFY)
		|     (12, INLINE)
		|     (11, CFA_INLINE)
	`)
}

func Test_computeCreationTree_multiRoot_allEmpty(t *testing.T) {
	ops := []*creationOp{
		{"ROOT", 0},
		{"ROOT", 1},
	}

	assertCreationTreeForOps(t, ops, `
		| (0, ROOT)
		| (1, ROOT)
	`)
}

func Test_computeCreationTree_multiRoot_allSingle(t *testing.T) {
	ops := []*creationOp{
		{"ROOT", 0},
		{"CFA_INLINE", 0},
		{"NOTIFY", 0},
		{"INLINE", 0},
		{"ROOT", 4},
		{"INLINE", 4},
		{"ROOT", 6},
		{"NOTIFY", 6},
		{"CFA_INLINE", 6},
		{"INLINE", 6},
	}

	assertCreationTreeForOps(t, ops, `
		| (0, ROOT)
		|   (2, CFA_INLINE)
		|   (1, NOTIFY)
		|   (3, INLINE)
		| (4, ROOT)
		|   (5, INLINE)
		| (6, ROOT)
		|   (7, NOTIFY)
		|   (8, CFA_INLINE)
		|   (9, INLINE)
	`)
}
func Test_computeCreationTree_multiRoot_allMulti(t *testing.T) {
	ops := []*creationOp{
		{"ROOT", 0},
		{"NOTIFY", 0},
		{"NOTIFY", 0},
		{"NOTIFY", 0},
		{"CFA_INLINE", 0},
		{"INLINE", 0},
		{"INLINE", 2},
		{"CFA_INLINE", 2},
		{"NOTIFY", 2},
		{"NOTIFY", 4},
		{"NOTIFY", 8},
		{"NOTIFY", 8},
		{"INLINE", 8},
		{"CFA_INLINE", 8},
		{"NOTIFY", 13},
		{"CFA_INLINE", 13},
		{"NOTIFY", 13},
		{"ROOT", 17},
		{"NOTIFY", 17},
		{"NOTIFY", 18},
		{"INLINE", 19},
		{"CFA_INLINE", 19},
		{"CFA_INLINE", 21},
		{"NOTIFY", 21},
		{"CFA_INLINE", 21},
	}

	assertCreationTreeForOps(t, ops, `
		| (0, ROOT)
		|   (1, NOTIFY)
		|   (2, NOTIFY)
		|     (13, INLINE)
		|       (14, NOTIFY)
		|       (16, CFA_INLINE)
		|       (15, NOTIFY)
		|     (7, CFA_INLINE)
		|     (4, NOTIFY)
		|       (5, NOTIFY)
		|   (3, NOTIFY)
		|   (6, CFA_INLINE)
		|   (8, INLINE)
		|     (9, NOTIFY)
		|     (10, NOTIFY)
		|     (12, INLINE)
		|     (11, CFA_INLINE)
		| (17, ROOT)
		|   (18, NOTIFY)
		|     (19, NOTIFY)
		|       (21, INLINE)
		|         (23, CFA_INLINE)
		|         (22, NOTIFY)
		|         (24, CFA_INLINE)
		|       (20, CFA_INLINE)
	`)
}

func Test_toFlatTree(t *testing.T) {
	root1 := &node{"ROOT", 0, []*node{
		{"NOTIFY", 1, nil},
		{"NOTIFY", 2, []*node{
			{"NOTIFY", 4, nil},
			{"INLINE", 6, []*node{
				{"NOTIFY", 7, nil},
				{"CFA_INLINE", 8, nil},
			}},
			{"CFA_INLINE", 5, nil},
		}},
		{"NOTIFY", 3, nil},
	}}

	root2 := &node{"ROOT", 9, []*node{
		{"NOTIFY", 10, nil},
		{"NOTIFY", 11, []*node{
			{"NOTIFY", 13, nil},
			{"CFA_INLINE", 14, nil},
		}},
		{"NOTIFY", 12, nil},
	}}

	assert.Equal(t, eosio.CreationFlatTree{
		eosio.CreationFlatNode{0, -1, 0},
		eosio.CreationFlatNode{1, 0, 1},
		eosio.CreationFlatNode{2, 0, 2},
		eosio.CreationFlatNode{3, 2, 4},
		eosio.CreationFlatNode{4, 2, 6},
		eosio.CreationFlatNode{5, 4, 7},
		eosio.CreationFlatNode{6, 4, 8},
		eosio.CreationFlatNode{7, 2, 5},
		eosio.CreationFlatNode{8, 0, 3},
		eosio.CreationFlatNode{9, -1, 9},
		eosio.CreationFlatNode{10, 9, 10},
		eosio.CreationFlatNode{11, 9, 11},
		eosio.CreationFlatNode{12, 11, 13},
		eosio.CreationFlatNode{13, 11, 14},
		eosio.CreationFlatNode{14, 9, 12},
	}, toFlatTree(root1, root2))
}

func assertCreationTreeForOps(t *testing.T, ops []*creationOp, expected string) {
	t.Helper()

	nodes, err := computeCreationTree(ops)
	require.NoError(t, err)

	assert.Equal(t, trimMargin(expected, "|"), nodes.Stringer())
}

func trimMargin(s string, delimiter string) string {
	lines := strings.Split(strings.TrimLeft(s, "\n"), "\n")
	mappedLines := make([]string, len(lines))
	for i, line := range lines {
		mappedLines[i] = strings.TrimPrefix(strings.TrimLeft(line, " \t"), delimiter)
	}

	return strings.Join(mappedLines, "\n")
}
