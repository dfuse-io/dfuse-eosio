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

package deos

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/stretchr/testify/assert"
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
		&creationOp{"ROOT", 0},
	}

	assertCreationTreeForOps(t, ops, `
		| (0, ROOT)
	`)
}

func Test_computeCreationTree_singleRoot_oneLevel(t *testing.T) {
	ops := []*creationOp{
		&creationOp{"ROOT", 0},
		&creationOp{"NOTIFY", 0},
		&creationOp{"CFA_INLINE", 0},
		&creationOp{"INLINE", 0},
		&creationOp{"CFA_INLINE", 0},
		&creationOp{"NOTIFY", 0},
		&creationOp{"INLINE", 0},
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
		&creationOp{"ROOT", 0},
		&creationOp{"NOTIFY", 0},
		&creationOp{"CFA_INLINE", 0},
		&creationOp{"INLINE", 0},
		&creationOp{"CFA_INLINE", 0},
		&creationOp{"NOTIFY", 0},
		&creationOp{"INLINE", 0},
		&creationOp{"NOTIFY", 5},
		&creationOp{"CFA_INLINE", 5},
		&creationOp{"INLINE", 5},
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
		&creationOp{"ROOT", 0},
		&creationOp{"NOTIFY", 0},
		&creationOp{"CFA_INLINE", 0},
		&creationOp{"INLINE", 0},
		&creationOp{"CFA_INLINE", 0},
		&creationOp{"NOTIFY", 0},
		&creationOp{"INLINE", 0},
		&creationOp{"NOTIFY", 2},
		&creationOp{"CFA_INLINE", 2},
		&creationOp{"INLINE", 2},
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
		&creationOp{"ROOT", 0},
		&creationOp{"NOTIFY", 0},
		&creationOp{"NOTIFY", 0},
		&creationOp{"NOTIFY", 0},
		&creationOp{"CFA_INLINE", 0},
		&creationOp{"INLINE", 0},
		&creationOp{"INLINE", 2},
		&creationOp{"CFA_INLINE", 2},
		&creationOp{"NOTIFY", 2},
		&creationOp{"NOTIFY", 4},
		&creationOp{"NOTIFY", 8},
		&creationOp{"NOTIFY", 8},
		&creationOp{"INLINE", 8},
		&creationOp{"CFA_INLINE", 8},
		&creationOp{"NOTIFY", 13},
		&creationOp{"CFA_INLINE", 13},
		&creationOp{"NOTIFY", 13},
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
		&creationOp{"ROOT", 0},
		&creationOp{"ROOT", 1},
	}

	assertCreationTreeForOps(t, ops, `
		| (0, ROOT)
		| (1, ROOT)
	`)
}

func Test_computeCreationTree_multiRoot_allSingle(t *testing.T) {
	ops := []*creationOp{
		&creationOp{"ROOT", 0},
		&creationOp{"CFA_INLINE", 0},
		&creationOp{"NOTIFY", 0},
		&creationOp{"INLINE", 0},
		&creationOp{"ROOT", 4},
		&creationOp{"INLINE", 4},
		&creationOp{"ROOT", 6},
		&creationOp{"NOTIFY", 6},
		&creationOp{"CFA_INLINE", 6},
		&creationOp{"INLINE", 6},
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
		&creationOp{"ROOT", 0},
		&creationOp{"NOTIFY", 0},
		&creationOp{"NOTIFY", 0},
		&creationOp{"NOTIFY", 0},
		&creationOp{"CFA_INLINE", 0},
		&creationOp{"INLINE", 0},
		&creationOp{"INLINE", 2},
		&creationOp{"CFA_INLINE", 2},
		&creationOp{"NOTIFY", 2},
		&creationOp{"NOTIFY", 4},
		&creationOp{"NOTIFY", 8},
		&creationOp{"NOTIFY", 8},
		&creationOp{"INLINE", 8},
		&creationOp{"CFA_INLINE", 8},
		&creationOp{"NOTIFY", 13},
		&creationOp{"CFA_INLINE", 13},
		&creationOp{"NOTIFY", 13},
		&creationOp{"ROOT", 17},
		&creationOp{"NOTIFY", 17},
		&creationOp{"NOTIFY", 18},
		&creationOp{"INLINE", 19},
		&creationOp{"CFA_INLINE", 19},
		&creationOp{"CFA_INLINE", 21},
		&creationOp{"NOTIFY", 21},
		&creationOp{"CFA_INLINE", 21},
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
		&node{"NOTIFY", 1, nil},
		&node{"NOTIFY", 2, []*node{
			&node{"NOTIFY", 4, nil},
			&node{"INLINE", 6, []*node{
				&node{"NOTIFY", 7, nil},
				&node{"CFA_INLINE", 8, nil},
			}},
			&node{"CFA_INLINE", 5, nil},
		}},
		&node{"NOTIFY", 3, nil},
	}}

	root2 := &node{"ROOT", 9, []*node{
		&node{"NOTIFY", 10, nil},
		&node{"NOTIFY", 11, []*node{
			&node{"NOTIFY", 13, nil},
			&node{"CFA_INLINE", 14, nil},
		}},
		&node{"NOTIFY", 12, nil},
	}}

	assert.Equal(t, CreationFlatTree{
		CreationFlatNode{0, -1, 0},
		CreationFlatNode{1, 0, 1},
		CreationFlatNode{2, 0, 2},
		CreationFlatNode{3, 2, 4},
		CreationFlatNode{4, 2, 6},
		CreationFlatNode{5, 4, 7},
		CreationFlatNode{6, 4, 8},
		CreationFlatNode{7, 2, 5},
		CreationFlatNode{8, 0, 3},
		CreationFlatNode{9, -1, 9},
		CreationFlatNode{10, 9, 10},
		CreationFlatNode{11, 9, 11},
		CreationFlatNode{12, 11, 13},
		CreationFlatNode{13, 11, 14},
		CreationFlatNode{14, 9, 12},
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
