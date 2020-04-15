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

package bigt

import (
	"fmt"
	"strings"
)

type idToPrefix map[string]string

func (m idToPrefix) prefix(prefixes []string, id string) (prefix string, err error) {
	if cachedPrefix, found := m[id]; found {
		return cachedPrefix, nil
	}

	for _, prefix = range prefixes {
		if strings.HasPrefix(id, prefix) {
			m[id] = prefix
			return
		}
	}

	return "", fmt.Errorf("no prefix found that match %q out of %d prefixes", id, len(prefixes))
}
