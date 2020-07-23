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

package pbtokenmeta

import fmt "fmt"

func (e *AccountBalance) Key() string {
	return fmt.Sprintf("%s:%s:%s", e.TokenContract, e.Symbol, e.Account)
}

func (e *Token) Key() string {
	return fmt.Sprintf("%s:%s", e.Contract, e.Symbol)
}

func (c *TokenCursor) Key() string {
	return fmt.Sprintf("%s:%s", c.Contract, c.Symbol)
}

func (c *AccountBalanceCursor) Key() string {
	return fmt.Sprintf("%s:%s:%s", c.Contract, c.Symbol, c.Account)

}
