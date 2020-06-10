# Partial synchronisation of Kylin testnet's head using dfuse for EOSIO

## Requirements

* A working `dfuseeos` command and nodeos installed with 'deep-mind' patch (see https://github.com/dfuse-io/dfuse-eosio/blob/develop/DEPENDENCIES.md#dfuse-instrumented-eosio-prebuilt-binaries)

## Get a clean workspace folder, fetch a Kylin snapshot (using EOS Nation snapshots as a source in this example)

```
mkdir workspace && cd workspace
curl -L -o kylin-snapshot.bin.zst https://snapshots.eosnation.io/kylin/latest
zstd -d kylin-snapshot.bin.zst
```

**Note** `zstd` CLI decompression tool can be downloaded from https://github.com/facebook/zstd/releases.

## Prepare {workspace}/kylin-phase1-blocks.yaml

### Required information
* `mindreader-stop-block-num: {kylin current head block number, rounded to 100}`
  * You can go to https://kylin.eosq.app/ or use this kind of shell command: `curl -s https://kylin.eos.dfuse.io/v1/chain/get_info | sed 's/.*head_block_num..\([0-9]*\),.*/\1/'`
* `mindreader-snapshot-store-url: file:///{Current working directory}`
  * Folder where you downloaded the snapshot (output of command `pwd` in your shell)

### Example kylin-phase1-blocks.yaml

```
start:
  args:
  - mindreader
  flags:
    config-file: ""
    log-to-file: false
    mindreader-log-to-zap: false
    mindreader-merge-and-store-directly: true
    mindreader-start-failure-handler: true
    mindreader-blocks-chan-capacity: 100000
    mindreader-restore-snapshot-name: snapshot.bin
    mindreader-discard-after-stop-num: false
    mindreader-snapshot-store-url: file:///home/johndoe/workspace
    mindreader-stop-block-num: 107367000
```

## Prepare mindreader nodeos config

```
# from {workspace}
mkdir mindreader
cat >mindreader/config.ini <<EOC
# Plugins
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

wasm-runtime = eos-vm-jit
eos-vm-oc-enable = true
eos-vm-oc-compile-threads = 4

## Peers (choose your favorite ones, those are from https://github.com/cryptokylin/CryptoKylin-Testnet/blob/master/config/peer-config.ini)
p2p-peer-address = peer.kylin.alohaeos.com:9876
p2p-peer-address = p2p.kylin.helloeos.com.cn:9876
p2p-peer-address = kylin-testnet.starteos.io:9876
p2p-peer-address = kylin-fn001.eossv.org:443
p2p-peer-address = testnet.zbeos.com:9876
p2p-peer-address = kylin.eosrio.io:39876
EOC

```

## Run 'phase1-blocks'

```
dfuseeos -c kylin-phase1-blocks.yaml start -v
```

* You can see the 'actual' progress of block files being written by running this command from another terminal: `ls -ltr dfuse-data/storage/merged-blocks/ |tail`
* From different terminal sessions, you can run the "search" and "trxdb" phases in parallel with this phase. They will wait for merged block files to be created. See next steps in this document.

KNOWN ISSUES:

* The mindreader writes merged block files SLOWER than the nodeos instance can catch up. This means that it will keep going for a while after the "stop block" appears in the logs. Do not worry and do not try to force kill the dfuseeos instance! Let it continue until it finishes.
* Since the nodeos will go further than the requested "stop block", all extra blocks will be written to the 'dfuse-data/storage/one-blocks` folder, so that the `merger` can pick them up on the next run.


## 'phase1-search' (can be done in parallel with phase1)

### Required information
* `search-indexer-start-block: {500 blocks higher than first merged block file, rounded to the next 'shard-size'}`
  * Take that bluck number `ls  ./dfuse-data/storage/merged-blocks/ |head -n 1` and add 500 to it
* `search-indexer-stop-block: {100 below the value that you put in for mindreader-stop-block-num earlier, rounded to the lower 'shard-size'}`

### Example kylin-phase1-search.yaml

```
start:
  args:
  - search-indexer
  flags:
    config-file: ""
    log-to-file: false
    search-indexer-enable-batch-mode: true
    search-indexer-start-block: 107305500
    search-indexer-stop-block: 107366500
    search-indexer-shard-size: 500
```

### Run 'phase1-search'

```
dfuseeos -c kylin-phase1-search.yaml start  -vv
```

NOTE: the 'actual' start block that you can use afterwards will most likely be the one that you set here in 'search-indexer'

## 'phase1-trxdb' (can be done in parallel with phase1)

### Required information
* `trxdb-loader-start-block-num: {500 blocks higher than first merged block file}`
  * Take that bluck number `ls  ./dfuse-data/storage/merged-blocks/ |head -n 1` and add 500 to it
* `trxdb-loader-stop-block-num: {100 below the value that you put in for mindreader-stop-block-num earlier}`
* `common-chain-id: 5fff1dae8dc8e2fc4d5b23b2c7665c97f9e9d8edf2b6485a86ba311c25639191`
  * change this if you are syncing another chain than kylin, could be scripted like this `curl -s https://kylin.eos.dfuse.io/v1/chain/get_info | sed 's/.*chain_id...\([a-f0-9]*\).*/\1/'` or with better tools like 'jq'

### Example kylin-phase1-trxdb.yaml

```
start:
  args:
  - trxdb-loader
  flags:
    config-file: ""
    log-to-file: false
    common-chain-id: 5fff1dae8dc8e2fc4d5b23b2c7665c97f9e9d8edf2b6485a86ba311c25639191
    trxdb-loader-start-block-num: 107305400
    trxdb-loader-stop-block-num: 107366900
    trxdb-loader-processing-type: batch
```

### Run 'phase1-trxdb'

```
dfuseeos -c kylin-phase1-trxdb.yaml start  -vv
```

KNOWN ISSUES:
* It's currently hard to follow the progress and not have too many logs

## Run kylin-phase2 (syncing up to head)

### Example kylin-phase2.yaml
```
start:
  args:
  - search-archive
  - search-router
  - search-indexer
  - search-live
  - dashboard
  - dgraphql
  - apiproxy
  - mindreader
  - merger
  - relayer
  - trxdb-loader
  - blockmeta
  flags:
    config-file: ""
    log-to-file: false
    mindreader-log-to-zap: true
    common-chain-id: 5fff1dae8dc8e2fc4d5b23b2c7665c97f9e9d8edf2b6485a86ba311c25639191
    search-indexer-shard-size: 500
    search-indexer-start-block: 107305500
    search-archive-shard-size: 500
    search-archive-start-block: 107305500
    blockmeta-eos-api-upstream-addr: https://kylin.eos.dfuse.io
```

### Known Issues

* You may see some warnings like this: `found a hole in a oneblock files`, sometimes they are false positive. Watch the progression of merged-blocks in the folder like this: `ls  ./dfuse-data/storage/merged-blocks/ |tail -n 1` to make sure that the merger keeps going correctly
* EOSQ will not work correctly without "eosws", "fluxdb", "abicodec", which do not support 'partial chain syncing' at the moment. You will need to sync the full chain to get that.

### Watch as the chain syncs its missing part up to the head

* dashboard: http://localhost:8081

### Play with search and block functions in the synced range

* graphiql: http://localhost:8080/graphiql

* examples queries:
  * query block: http://localhost:8080/graphiql/?query=cXVlcnkgKCRibG9ja051bTogVWludDMyKSB7CiAgYmxvY2sobnVtOiAkYmxvY2tOdW0pIHsKICAgIGlkCiAgICBudW0KICAgIGRwb3NMSUJOdW0KICAgIGV4ZWN1dGVkVHJhbnNhY3Rpb25Db3VudAogICAgaXJyZXZlcnNpYmxlCiAgICBoZWFkZXIgewogICAgICBpZAogICAgICBudW0KICAgICAgdGltZXN0YW1wCiAgICAgIHByb2R1Y2VyCiAgICAgIHByZXZpb3VzCiAgICB9CiAgICB0cmFuc2FjdGlvblRyYWNlcyhmaXJzdDogNSkgewogICAgICBwYWdlSW5mbyB7CiAgICAgICAgc3RhcnRDdXJzb3IKICAgICAgICBlbmRDdXJzb3IKICAgICAgfQogICAgICBlZGdlcyB7CiAgICAgICAgY3Vyc29yCiAgICAgICAgbm9kZSB7CiAgICAgICAgICBpZAogICAgICAgICAgc3RhdHVzCiAgICAgICAgICB0b3BMZXZlbEFjdGlvbnMgewogICAgICAgICAgICBhY2NvdW50CiAgICAgICAgICAgIG5hbWUKICAgICAgICAgICAgcmVjZWl2ZXIKICAgICAgICAgICAganNvbgogICAgICAgICAgfQogICAgICAgIH0KICAgICAgfQogICAgfQogIH0KfQo=&variables=ewogICJibG9ja051bSI6IDEwNzMwNTUwMAp9
  * stream search: show actions other than `onblock`: http://localhost:8080/graphiql/?query=c3Vic2NyaXB0aW9uICgkcXVlcnk6IFN0cmluZyEsICRjdXJzb3I6IFN0cmluZykgewogIHNlYXJjaFRyYW5zYWN0aW9uc0ZvcndhcmQocXVlcnk6ICRxdWVyeSwgY3Vyc29yOiAkY3Vyc29yKSB7CiAgICB1bmRvCiAgICBjdXJzb3IKICAgIHRyYWNlIHsKICAgICAgYmxvY2sgewogICAgICAgIG51bQogICAgICAgIGlkCiAgICAgICAgY29uZmlybWVkCiAgICAgICAgdGltZXN0YW1wCiAgICAgICAgcHJldmlvdXMKICAgICAgfQogICAgICBpZAogICAgICBtYXRjaGluZ0FjdGlvbnMgewogICAgICAgIGFjY291bnQKICAgICAgICBuYW1lCiAgICAgICAganNvbgogICAgICAgIHNlcQogICAgICAgIHJlY2VpdmVyCiAgICAgIH0KICAgIH0KICB9Cn0K&variables=ewogICJxdWVyeSI6ICItYWN0aW9uOm9uYmxvY2siLAogICJjdXJzb3IiOiAiIgp9



