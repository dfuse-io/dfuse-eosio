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
	"fmt"
	"strings"
)

type nodes []*node

type node struct {
	kind        string
	actionIndex int
	children    []*node
}

func computeCreationTree(ops []*creationOp) (nodes, error) {
	if len(ops) <= 0 {
		return nil, nil
	}

	actionIndex := -1
	opsMap := creationOpsToMap(ops)

	var roots []*node
	opKinds, ok := opsMap[actionIndex+1]

	for ok {
		if opKinds[0] != "ROOT" {
			return nil, fmt.Errorf("first exec op kind of execution start should be ROOT, got %s", opKinds[0])
		}

		root := &node{"ROOT", -1, nil}
		roots = append(roots, root)

		executeAction(&actionIndex, root, opsMap)

		opKinds, ok = opsMap[actionIndex+1]

		// TODO: We should check for gaps in action indices here. Assume an exec ops
		//       list of `[{ROOT, 0}, {NOTIFY, 1}, {ROOT, 2}]`. In this list, we would
		//       create a ROOT #0, skip NOTIFY then try to execute ROOT #2. This is incorrect
		//       and there is a gap, i.e. there is an action index lower than next 2 that is
		//       not part of previous tree. How exactly we would do it is unsure, but that would
		//       add a validation step that everything is kosher.
	}

	return roots, nil
}

func executeAction(
	actionIndex *int,
	root *node,
	opsMap map[int][]string,
) {
	*actionIndex++
	root.actionIndex = *actionIndex

	notifies, cfas, inlines := recordChildCreationOp(root, opsMap[root.actionIndex])

	for i := 0; i < len(notifies); i++ {
		nestedNotifies, nestedCfas, nestedInlines := executeNotify(actionIndex, notifies[i], opsMap)

		notifies = append(notifies, nestedNotifies...)
		cfas = append(cfas, nestedCfas...)
		inlines = append(inlines, nestedInlines...)
	}

	for _, cfa := range cfas {
		executeAction(actionIndex, cfa, opsMap)
	}

	for _, inline := range inlines {
		executeAction(actionIndex, inline, opsMap)
	}
}

func executeNotify(
	actionIndex *int,
	root *node,
	opsMap map[int][]string,
) (notifies []*node, cfas []*node, inlines []*node) {
	*actionIndex++
	root.actionIndex = *actionIndex

	return recordChildCreationOp(root, opsMap[root.actionIndex])
}

func recordChildCreationOp(root *node, opKinds []string) (notifies []*node, cfas []*node, inlines []*node) {
	for _, opKind := range opKinds {
		if opKind == "ROOT" {
			continue
		}

		child := &node{opKind, -1, nil}
		switch opKind {
		case "NOTIFY":
			notifies = append(notifies, child)
		case "CFA_INLINE":
			cfas = append(cfas, child)
		case "INLINE":
			inlines = append(inlines, child)
		}

		root.children = append(root.children, child)
	}

	return
}

func creationOpsToMap(ops []*creationOp) map[int][]string {
	mapping := map[int][]string{}
	for _, op := range ops {
		mapping[op.actionIndex] = append(mapping[op.actionIndex], op.kind)
	}

	return mapping
}

func toFlatTree(roots ...*node) CreationFlatTree {
	var tree CreationFlatTree

	walkIndex := -1
	for _, root := range roots {
		walkIndex++
		tree = append(tree, _toFlatTree(root, -1, &walkIndex)...)
	}

	return tree
}

func _toFlatTree(root *node, parentIndex int, walkIndex *int) (tree CreationFlatTree) {
	tree = append(tree, [3]int{*walkIndex, parentIndex, root.actionIndex})
	childRootIndex := *walkIndex

	for _, child := range root.children {
		*walkIndex++
		tree = append(tree, _toFlatTree(child, childRootIndex, walkIndex)...)
	}

	return
}

func (nodes nodes) Stringer() string {
	builder := &strings.Builder{}
	for _, node := range nodes {
		node.toString(builder, "")
	}

	return builder.String()
}

func (node *node) Stringer() string {
	builder := &strings.Builder{}
	node.toString(builder, "")

	return builder.String()
}

func (node *node) toString(builder *strings.Builder, spacing string) {
	fmt.Fprintf(builder, "%s (%d, %s)\n", spacing, node.actionIndex, node.kind)
	for _, child := range node.children {
		child.toString(builder, spacing+"  ")
	}
}
