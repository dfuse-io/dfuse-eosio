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

package cli

var localGenesisJSON = `{
	"initial_timestamp": "2018-07-23T17:14:45",
	"initial_key":       "EOS6MRyAjQq8ud7hVNYcfnVPJqcVpscN5So8BhtHuGYqET5GDW5CV"
}
`

var producerLocalConfigIni = `# Plugins
plugin = eosio::producer_plugin
plugin = eosio::producer_api_plugin
plugin = eosio::chain_plugin
plugin = eosio::chain_api_plugin
plugin = eosio::http_plugin
plugin = eosio::db_size_api_plugin
plugin = eosio::net_api_plugin

# Chain
chain-state-db-size-mb = 4096
reversible-blocks-db-size-mb = 512
max-transaction-time = 5000

# P2P
agent-name = dfuse for EOSIO (producer)
p2p-server-address = 127.0.0.1:9876
p2p-listen-endpoint = 127.0.0.1:9876
p2p-max-nodes-per-host = 5
connection-cleanup-period = 15

# HTTP
http-server-address = 127.0.0.1:8888
http-max-response-time-ms = 1000
http-validate-host = 0
verbose-http-errors = true

# We want to produce the block logs, no deep-mind instrumentation here.
producer-name = eosio
enable-stale-production = true
signature-provider = EOS6MRyAjQq8ud7hVNYcfnVPJqcVpscN5So8BhtHuGYqET5GDW5CV=KEY:5KQwrPbwdL6PhXujxW37FSSQZ1JiwsST4cqQzDeyXtP79zkvFD3
`

var mindreaderLocalConfigIni = `# Plugins
plugin = eosio::producer_plugin      # for state snapshots
plugin = eosio::producer_api_plugin  # for state snapshots
plugin = eosio::chain_plugin
plugin = eosio::chain_api_plugin
plugin = eosio::http_plugin
plugin = eosio::db_size_api_plugin
plugin = eosio::net_api_plugin

# Chain
chain-state-db-size-mb = 4096
reversible-blocks-db-size-mb = 512
max-transaction-time = 5000

read-mode = head
p2p-accept-transactions = false
api-accept-transactions = false

# P2P
agent-name = dfuse for EOSIO (mindreader)
p2p-server-address = 127.0.0.1:9877
p2p-listen-endpoint = 127.0.0.1:9877
p2p-max-nodes-per-host = 2
connection-cleanup-period = 60

# HTTP
access-control-allow-origin = *
http-server-address = 127.0.0.1:9888
http-max-response-time-ms = 1000
http-validate-host = false
verbose-http-errors = true

# Enable deep mind
deep-mind = true
contracts-console = true

## Peers
p2p-peer-address = 127.0.0.1:9876
`

var mindreaderRemoteConfigIniFormat = `# Plugins
plugin = eosio::producer_plugin      # for state snapshots
plugin = eosio::producer_api_plugin  # for state snapshots
plugin = eosio::chain_plugin
plugin = eosio::chain_api_plugin
plugin = eosio::http_plugin
plugin = eosio::db_size_api_plugin
plugin = eosio::net_api_plugin

# Chain
chain-state-db-size-mb = 64000
reversible-blocks-db-size-mb = 2048
max-transaction-time = 5000

read-mode = head
p2p-accept-transactions = false
api-accept-transactions = false

# P2P
agent-name = dfuse for EOSIO (mindreader)
p2p-server-address = 127.0.0.1:9877
p2p-listen-endpoint = 127.0.0.1:9877
p2p-max-nodes-per-host = 2
connection-cleanup-period = 60

# HTTP
access-control-allow-origin = *
http-server-address = 127.0.0.1:9888
http-max-response-time-ms = 1000
http-validate-host = false
verbose-http-errors = true

# Enable deep mind
deep-mind = true
contracts-console = true

## Peers
%s
`
