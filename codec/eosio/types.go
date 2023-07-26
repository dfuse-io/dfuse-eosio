package eosio

import (
	"unicode/utf8"

	pbcodec "github.com/dfuse-io/dfuse-eosio/pb/dfuse/eosio/codec/v1"
	"github.com/eoscanada/eos-go"
)

// CreationFlatTree represents the creation order tree
// in a flatten manners. The flat list is built by doing
// a deep-first walk of the creational tree, outputting
// at each traversal the `CreationNode` triplet
// `(index, creatorParentIndex, executionIndex)` where a parent of
// `-1` represents a root node.
//
// For example, assuming a `CreationFlatTree` of the form:
//
// [
//
//	[0, -1, 0],
//	[1, 0, 1],
//	[2, 0, 2],
//	[3, 2, 3],
//
// ]
//
// Represents the following creational tree:
//
// ```
//
//	0
//	├── 1
//	└── 2
//	    └── 3
//
// ```
//
// The tree can be reconstructed using the following quick Python.
type CreationFlatTree = []CreationFlatNode

// CreationFlatNode represents a flat node in a flat tree.
// It's a triplet slice where elements reprensents the following
// values, assuming `(<depthFirstWalkIndex>, <parentDepthFirstWalkIndex>, <executionActionIndex>)`:
//
// The first value of the node is it's id, derived by doing a depth-first walk
// of the creation tree and incrementing an index at each node visited.
//
// The second value is the parent index of the current node, the index is the
// index of the initial element of the `CreationFlatNode` slice.
//
// The third value is the execution action index to get the actual execution traces
// from the actual execution tree (deep-first walking index in the execution
// tree).
type CreationFlatNode = [3]int

type ConversionOption interface{}

type ActionConversionOption interface {
	Apply(actionTrace *pbcodec.ActionTrace)
}

type actionConversionOptionFunc func(actionTrace *pbcodec.ActionTrace)

func (f actionConversionOptionFunc) Apply(actionTrace *pbcodec.ActionTrace) {
	f(actionTrace)
}

func LimitConsoleLengthConversionOption(maxByteCount int) ConversionOption {
	return actionConversionOptionFunc(func(in *pbcodec.ActionTrace) {
		if maxByteCount == 0 {
			return
		}

		if len(in.Console) > maxByteCount {
			in.Console = in.Console[:maxByteCount]

			// Prior truncation, the string had only valid UTF-8 charaters, so at worst, we will need
			// 3 bytes (`utf8.UTFMax - 1`) to reach a valid UTF-8 sequence.
			for i := 0; i < utf8.UTFMax-1; i++ {
				lastRune, size := utf8.DecodeLastRuneInString(in.Console)
				if lastRune != utf8.RuneError {
					// Last element is a valid utf8 character, nothing more to do here
					return
				}

				// We have an invalid UTF-8 sequence, size 0 means empty string, size 1 means invalid character
				if size == 0 {
					// The actual string was empty, nothing more to do here
					return
				}

				in.Console = in.Console[:len(in.Console)-1]
			}
		}
	})
}

// Best effort to extract public keys from a signed transaction
func GetPublicKeysFromSignedTransaction(chainID eos.Checksum256, signedTransaction *eos.SignedTransaction) []string {
	eccPublicKeys, err := signedTransaction.SignedByKeys(chainID)
	if err != nil {
		// We discard any errors and simply return an empty array
		return nil
	}

	publicKeys := make([]string, len(eccPublicKeys))
	for i, eccPublicKey := range eccPublicKeys {
		publicKeys[i] = eccPublicKey.String()
	}

	return publicKeys
}
