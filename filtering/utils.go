// Copyright 2020 dfuse Platform Inc.
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

package filtering

import (
	"sort"
	"strconv"
)

func toList(in map[string]bool) (out []string) {
	for k := range in {
		out = append(out, k)
	}
	sort.Strings(out)
	return
}

func fromHexUint16(input string) (uint16, error) {
	val, err := strconv.ParseUint(input, 16, 16)
	if err != nil {
		return 0, err
	}
	return uint16(val), nil
}

func fromHexUint32(input string) (uint32, error) {
	val, err := strconv.ParseUint(input, 16, 32)
	if err != nil {
		return 0, err
	}
	return uint32(val), nil
}
