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

package dgraphql

import (
	"fmt"

	types "github.com/dfuse-io/dfuse-eosio/dgraphql/types"
	"github.com/dfuse-io/dgraphql"
	"github.com/gogo/protobuf/proto"
)

type Paginator struct {
	beforeKey       string
	afterKey        string
	first           uint32
	last            uint32
	HasPreviousPage bool
	HasNextPage     bool
}

func NewPaginator(firstReq, lastReq *types.Uint32, before, after *string, limit uint32, cursorFactory func() proto.Message) (*Paginator, error) {

	first := uint32(0)
	if firstReq != nil {
		first = uint32(*firstReq)
	}

	last := uint32(0)
	if lastReq != nil {
		last = uint32(*lastReq)
	}

	beforeKey := ""
	afterKey := ""

	if (first > 0) && (last > 0) {
		return nil, fmt.Errorf("cannot used first and last arguments together")
	} else if (limit > 0) && ((first > limit) || (last > limit)) {
		return nil, fmt.Errorf("invalid first or last argument for this query: max %d", limit)
	}

	if first == 0 && last == 0 && limit > 0 {
		first = limit
	}

	if before != nil {
		beforeCursor := cursorFactory()
		err := dgraphql.UnmarshalCursorProto(*before, beforeCursor)
		if err != nil {
			return nil, fmt.Errorf("unable to process before cursor: %s", err)
		}
		beforeKey = beforeCursor.(Keyable).Key()
	}

	if after != nil {
		afterCursor := cursorFactory()
		err := dgraphql.UnmarshalCursorProto(*after, afterCursor)
		if err != nil {
			return nil, fmt.Errorf("unable to process after cursor: %s", err)
		}
		afterKey = afterCursor.(Keyable).Key()
	}

	return &Paginator{
		beforeKey: beforeKey,
		afterKey:  afterKey,
		first:     first,
		last:      last,
	}, nil
}

func (p *Paginator) Paginate(results Pagineable) Pagineable {
	validBeforeKey := false
	validAfterKey := false
	beforeKeyPassed := false
	afterKeyPassed := false

	indexes := []int{}
	for i := 0; i < results.Length(); i++ {
		if p.beforeKey != "" {
			// there is a before key
			if !beforeKeyPassed {
				// before key has not been seen
				if results.IsEqual(i, p.beforeKey) {
					// we are at the before key
					beforeKeyPassed = true
					validBeforeKey = false
				} else {
					// we are not yet at the before key
					validBeforeKey = true
				}
			} else {
				// before key has passed
				validBeforeKey = false
			}
		} else {
			// no before key so always valid
			validBeforeKey = true
		}

		if p.afterKey != "" {
			// there is an after key
			if afterKeyPassed {
				// after key has passed
				validAfterKey = true
			} else {
				// before key has not been seen
				if results.IsEqual(i, p.afterKey) {
					// we are at the after key
					afterKeyPassed = true
					validAfterKey = false
				} else {
					// we are not yet at the after key
					validAfterKey = false
				}
			}
		} else {
			// no after key so always valid
			validAfterKey = true
		}

		if validBeforeKey && validAfterKey {
			indexes = append(indexes, i)
		}

	}

	if (p.first > 0) && (int(p.first) < len(indexes)) {
		indexes = indexes[0:p.first]
	} else if (p.last > 0) && (int(p.last) < len(indexes)) {
		indexes = indexes[(len(indexes) - int(p.last)):len(indexes)]
	}

	if len(indexes) == 0 {
		if results.Length() > 0 {
			p.HasNextPage = true
			p.HasPreviousPage = true
		}
	} else {
		if indexes[len(indexes)-1] != (results.Length() - 1) {
			p.HasNextPage = true
		}

		if indexes[0] != 0 {
			p.HasPreviousPage = true
		}
	}

	var elements Pagineable
	for _, i := range indexes {
		elements = results.Append(elements, i)
	}
	return elements
}

type Keyable interface {
	Key() string
}

type Pagineable interface {
	Length() int
	IsEqual(index int, key string) bool
	Append(slice Pagineable, index int) Pagineable
}

type PagineableStrings []string

func (p PagineableStrings) Length() int {
	return len(p)
}

func (p PagineableStrings) IsEqual(index int, key string) bool {
	return p[index] == key
}

func (p PagineableStrings) Append(slice Pagineable, index int) Pagineable {
	if slice == nil {
		return Pagineable(PagineableStrings([]string{p[index]}))
	} else {
		return Pagineable(append(slice.(PagineableStrings), p[index]))
	}
}
