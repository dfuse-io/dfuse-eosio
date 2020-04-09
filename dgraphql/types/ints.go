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
	"errors"
	"strconv"

	eos "github.com/eoscanada/eos-go"
)

///
/// Int64
///

type Int64 int64

// ToInt64 does EOS-style decoding of an int64, and returns a Int64 from this package.
func ToInt64(rawJSON string) Int64 {
	var res eos.Int64
	_ = res.UnmarshalJSON([]byte(rawJSON))
	return Int64(res)

}

func (u Int64) ImplementsGraphQLType(name string) bool {
	return name == "Int64"
}

func (u *Int64) Native() int64 {
	if u == nil {
		return 0
	}
	return int64(*u)
}

func (u *Int64) UnmarshalGraphQL(input interface{}) error {
	var err error
	switch input := input.(type) {
	case string:
		res, err2 := strconv.ParseInt(input, 10, 64)
		if err2 != nil {
			err = err2
			break
		}
		*u = Int64(res)
	case float64:
		// FIXME: do bound checks, ensure it fits within the Int64
		// boundaries before truncating silently.
		*u = Int64(input)
	case float32:
		*u = Int64(input)
	case int64:
		*u = Int64(input)
	case uint64:
		*u = Int64(input)
	case uint32:
		*u = Int64(input)
	case int32:
		*u = Int64(input)
	default:
		err = errors.New("wrong type")
	}
	return err
}

func (u Int64) MarshalJSON() ([]byte, error) {
	return []byte(`"` + strconv.FormatInt(u.Native(), 10) + `"`), nil
}

///
/// Uint64
///

type Uint64 uint64

// ToUint64 does EOS-style decoding of an uint64, and returns a Uint64 from this package.
func ToUint64(rawJSON string) Uint64 {
	var res eos.Uint64
	_ = res.UnmarshalJSON([]byte(rawJSON))
	return Uint64(res)

}

func (u Uint64) ImplementsGraphQLType(name string) bool {
	return name == "Uint64"
}

func (u *Uint64) Native() uint64 {
	if u == nil {
		return 0
	}
	return uint64(*u)
}

func (u *Uint64) UnmarshalGraphQL(input interface{}) error {
	var err error
	switch input := input.(type) {
	case string:
		res, err2 := strconv.ParseUint(input, 10, 64)
		if err2 != nil {
			err = err2
			break
		}
		*u = Uint64(res)
	case float64:
		// FIXME: do bound checks, ensure it fits within the Uint64
		// boundaries before truncating silently.
		*u = Uint64(input)
	case float32:
		*u = Uint64(input)
	case int64:
		*u = Uint64(input)
	case uint64:
		*u = Uint64(input)
	case uint32:
		*u = Uint64(input)
	case int32:
		*u = Uint64(input)
	default:
		err = errors.New("wrong type")
	}
	return err
}

func (u Uint64) MarshalJSON() ([]byte, error) {
	return []byte(`"` + strconv.FormatUint(u.Native(), 10) + `"`), nil
}
