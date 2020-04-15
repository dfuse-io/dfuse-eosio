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
	"testing"
)

func TestBlockLinkableInterface(t *testing.T) {
	t.Skip("TODO")
	//cnt := `{"block":{"id":"00000bd4455cf6af570b9d244320c71489779199a2527cc4ed8550f01e7d8c7b","block_num":3028,"header":{"timestamp":"2018-06-09T12:23:27.500","producer":"eosio","confirmed":0,"previous":"00000bd31ac5b2defa70706c5032ea9b69897e8c906bbb65b0d1773c7f5fed7e","transaction_mroot":"0000000000000000000000000000000000000000000000000000000000000000","action_mroot":"9faa201236f16fc14b84a8a454fb9475259c2c08058e091b4abbbb2630bffebb","schedule_version":0,"header_extensions":[],"producer_signature":"SIG_K1_KivwpSye7WCcai7XvyUXLkUWLUwVPhTcBkQV6FDEPDTHi4PrQYHgSaJEov26cSxD8ceQJR9R2xdmNKMa5tAGAj48VhXE6X"},"dpos_proposed_irreversible_blocknum":3028,"dpos_irreversible_blocknum":3027,"bft_irreversible_blocknum":0,"pending_schedule_lib_num":0,"pending_schedule_hash":"ef8306df0f8b307111e9b2f4d25ed68076229da7d94080d420ec791e7181f510","pending_schedule":{"version":0,"producers":[]},"active_schedule":{"version":0,"producers":[{"producer_name":"eosio","block_signing_key":"EOS7EarnUhcyYqmdnPon8rm7mBCTnBoot6o7fE2WzjvEX2TdggbL3"}]},"blockroot_merkle":{"_active_nodes":["00000bd31ac5b2defa70706c5032ea9b69897e8c906bbb65b0d1773c7f5fed7e","686f4a24499851225883fa25b1536b16801c39e70ac05ac43f9934a8155fb136","101ac58ab8cc115fda513ab37395b54654913bc47b23c3233887fd6377e80694","5e0110805a86855a0df673f357b1c394c550b4976e0069eb76e9fd680e9bad93","30842838627629719a9cc58cf714efe4a3972e460defa2a659fbf7e022827e4a","7e13af20c73138992e019fbbf24b3a9861494450582dfd1fd51ab7abe106c961","430ad8cac716f1d37d7344755f07b81e1819fe9ce8da3ee28c6eaa3163855b1e","f70f047609de8073ba020b278a8cf815cf80ea035c07fb2c78844edcb19b1a58","7d2d7f1ea3a18b511ec964256010b513ec828b2fbc214804cc0f029bd96cdd63"],"_node_count":3027},"producer_to_last_produced":[["eosio",3028]],"producer_to_last_implied_irb":[["eosio",3027]],"block_signing_key":"EOS7EarnUhcyYqmdnPon8rm7mBCTnBoot6o7fE2WzjvEX2TdggbL3","confirm_count":[],"confirmations":[],"block":{"timestamp":"2018-06-09T12:23:27.500","producer":"eosio","confirmed":0,"previous":"00000bd31ac5b2defa70706c5032ea9b69897e8c906bbb65b0d1773c7f5fed7e","transaction_mroot":"0000000000000000000000000000000000000000000000000000000000000000","action_mroot":"9faa201236f16fc14b84a8a454fb9475259c2c08058e091b4abbbb2630bffebb","schedule_version":0,"header_extensions":[],"producer_signature":"SIG_K1_KivwpSye7WCcai7XvyUXLkUWLUwVPhTcBkQV6FDEPDTHi4PrQYHgSaJEov26cSxD8ceQJR9R2xdmNKMa5tAGAj48VhXE6X","transactions":[],"block_extensions":[]},"validated":false,"in_current_chain":true},"transaction_traces":[{"dbops":null,"id":"edc080fda1ca7e8e7836e3588c05fcb211d6c046f8b6df441df4e456dea13d22","receipt":{"status":"executed","cpu_usage_us":100,"net_usage_words":0},"elapsed":300,"net_usage":0,"scheduled":false,"action_traces":[{"receipt":{"receiver":"eosio","act_digest":"ce14ae4e7ef5686f84051a2e0f5806584341cef62c3a2bd03d636d234c924c9c","global_sequence":725766,"recv_sequence":340316,"auth_sequence":[["eosio",725759]],"code_sequence":2,"abi_sequence":2},"act":{"account":"eosio","name":"onblock","authorization":[{"actor":"eosio","permission":"active"}],"data":"7e065d450000000000ea3055000000000bd2345b507ad9bcfd9d621d2c2c835413bfd702a0f854075d1d9d939cd0c625c14cb97a138e557317d677916424ac6e706c340e94796d6915222810b7a1a312682190fd0f0ddec4dc48fcb67001c03ad2ec57a1f6a6c964b226ec24396b000000000000"},"elapsed":273,"cpu_usage":0,"console":"","total_cpu_usage":0,"trx_id":"edc080fda1ca7e8e7836e3588c05fcb211d6c046f8b6df441df4e456dea13d22","inline_traces":[]}],"except":null}]}`
	//
	//var blk *Block
	//err := json.Unmarshal([]byte(cnt), &blk)
	//assert.NoError(t, err)
	//blk.Parse()
	//
	//assert.Equal(t, "00000bd4455cf6af570b9d244320c71489779199a2527cc4ed8550f01e7d8c7b", blk.BlockID())
	//assert.Equal(t, uint64(3028), blk.BlockNum())
	//assert.Equal(t, "00000bd31ac5b2defa70706c5032ea9b69897e8c906bbb65b0d1773c7f5fed7e", blk.PreviousID())
	//assert.Equal(t, []string{"edc080fda1ca7e8e7836e3588c05fcb211d6c046f8b6df441df4e456dea13d22"}, blk.TransactionIDs())
}

//func TestRLimitOp_Account(t *testing.T) {
//	tests := []struct {
//		in       eos.RLimitOp
//		expected string
//	}{
//		{eos.RLimitOp{"CONFIG", "UPD", json.RawMessage(`{"owner":"eosio"}`), ""}, ""},
//		{eos.RLimitOp{"STATE", "UPD", json.RawMessage(`{"owner":"eosio"}`), ""}, ""},
//		{eos.RLimitOp{"ACCOUNT_LIMITS", "UPD", json.RawMessage(`{"owner":"eosio"}`), ""}, "eosio"},
//		{eos.RLimitOp{"ACCOUNT_USAGE", "UPD", json.RawMessage(`{"owner":"eosio"}`), ""}, "eosio"},
//		{eos.RLimitOp{"ACCOUNT_LIMITS", "UPD", json.RawMessage(`{"owner":"eosio"}`), "precached"}, "precached"},
//		{eos.RLimitOp{"ACCOUNT_USAGE", "UPD", json.RawMessage(`{"owner":"eosio"}`), "precached"}, "precached"},
//		{eos.RLimitOp{"ACCOUNT_LIMITS", "UPD", json.RawMessage(`{"no":"eosio"}`), ""}, ""},
//		{eos.RLimitOp{"ACCOUNT_USAGE", "UPD", json.RawMessage(`{"no":"eosio"}`), ""}, ""},
//	}
//
//	for i, test := range tests {
//		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
//			actual := test.in.Account()
//			assert.Equal(t, test.expected, actual)
//		})
//	}
//}
