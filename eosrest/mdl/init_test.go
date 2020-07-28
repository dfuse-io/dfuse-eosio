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

package mdl

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"go.uber.org/zap"
)

func init() {
	if os.Getenv("DEBUG") != "" {
		zlog, _ = zap.NewDevelopment()
	}
}

func unmarshalFromFixture(filename string, target interface{}) {
	err := json.Unmarshal(fromFixture(filename), target)
	if err != nil {
		panic(fmt.Errorf("unable to unmarshal fixture %s: %s", filename, err))
	}
}

func fromFixture(filename string) []byte {
	cnt, err := ioutil.ReadFile(filename)
	if err != nil {
		panic(fmt.Errorf("unable to read fixture %s: %s", filename, err))
	}

	return []byte(strings.TrimSpace(string(cnt)))
}
