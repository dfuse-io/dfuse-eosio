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

package types

import (
	"encoding/json"
	"errors"
)

type JSON []byte

func (t JSON) ImplementsGraphQLType(name string) bool {
	return name == "JSON"
}

func (t *JSON) UnmarshalGraphQL(input interface{}) error {
	var err error
	switch input := input.(type) {
	case []byte:
		*t = input
	case json.RawMessage:
		*t = []byte(input)
	case string:
		*t = []byte(input)
	default:
		err = errors.New("wrong type")
	}
	return err
}

func (t JSON) MarshalJSON() ([]byte, error) {
	return []byte(t), nil
}
