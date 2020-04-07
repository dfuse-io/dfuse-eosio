## FluxDB

An historical database for tabular data with loader for EOSIO state table & permissions.

**Important** While this is used in production currently at dfuse, this repository will
undergo a major refactoring in the upcoming months. We will revisit the project, its
design, its role and the overall architecture. It will most probably be split into
smaller pieces, a core module that does storage of historical data in a really agnostic
fashion, even blockchain agnostic and a second component that uses the core API to
implement the current set of features in EOSIO.

### Getting Started

To develop easily within `fluxdb` project, you will need the following:

* `gcloud`

If you never developed within `fluxdb`, run these commands once:

* `gcloud components install cbt`
* `go get github.com/cespare/reflex`

Spawn a side terminal and launch GCloud BigTable emulator in this side terminal:

```
gcloud beta emulators bigtable start
```

Back to your previous document, simply start a serving server in live reload mode
with:

```
reflex -r '\.go$' -s -- sh -c 'go install ./fluxdb && bash serve_dev.sh'
```

#### Indexing

The previous section was all about launching a serving instance, one that does not
index anything. If you need to start an indexing instance, do the following instead:

    ./index_dev.sh

#### Traces

This service is instrumented with Opencensus to collect traces and metrics. While
developing, we highly recommend using [zipkin](https://zipkin.io/) trace viewer
as your default exporter.

To use it, first start a Docker container running Zipkin system exposed:

```
docker pull openzipkin/zipkin
docker run -d -p 9411:9411 openzipkin/zipkin
```

Once Zipkin is running, simply export and extra environment variable before
starting the service. The variable that should be exported is `FLUXDB_ZIPKIN_EXPORTER`
and the value should be the `host:port` string where `Zipkin` system is running,
`localhost:9411` in this example:

```
export FLUXDB_ZIPKIN_EXPORTER="http://localhost:9411/api/v2/spans"
```

Launch the service as usual, perform some requests and then navigate to `locahost:9411`
from your favorite browser. Then `Find Traces` to see all traces (or query your exact one).

### Sample Queries

Assuming the following has been run already:

    export FLUXDB_BASE_URL=https://mainnet.eos.dfuse.io

#### Get Currency Balances

Get currency balances for user's account `eoscanada` (specified as `scope`) on various currencies (specified as `accounts`).

   curl "$FLUXDB_BASE_URL/v0/state/tables/accounts?block_num=26415000&accounts=eosio.token|eosadddddddd|tokenbyeocat|ethsidechain|epraofficial|alibabapoole|hirevibeshvt|oo1122334455|irespotokens|publytoken11|parslseed123|trybenetwork|zkstokensr4u&scope=eoscanadacom&table=accounts&json=true&&token=$DFUSE" | jq .

#### Auth

##### GET /v0/linked_permissions?account=eoscanadacom&block_num=123123

on veut:

```
{"block_num": 12312312321,
 "block_id": "alsdkjfldsakjflsakjfladkjfs",
 "account": "eoscanadacom",
 "links": [
   {"account": "eosio.token", "action_name": "transfer", "permission": "day2day"},
   {"account": "eosio.token", "action_name": "transfer", "permission": "active"},
  ]

 "links":  {
   "day2day": [{"account": "eosio.token", "action_name": "transfer"}]
  }

 "links": {
   "eosio.token:transfer": "day2day",
   "eosio.token:mogule": "day2day",
   "eosio.token:bobobob": "day2day",
 }
}
```

##### GET /v0/permissions?account=eoscanadacom&block_num=123123

## Massive reprocessing sharding steps design

Parallelly read all the blocks logs, open them and collect ONLY the
table changes, like FluxDB would, in order.  Purge any other
information in transactions, blocks, headers, except: `trx.TableOps`,
`trx.ActionTraces`, `trx.PermOps`, trx.ActionTrace and its data for
`eosio:eosio:linkauth+unlinkauth+setabi`, `trx.DbOps`

Parallelly, `fluxdb shard 1/100`, `2/100`. Process blo

Take all the things that would produce KEYS, and SHARD them according
to a simple hash + modulo algorithm.  For each shard, write it to a
file like:

* `001-000123123123-000234234234.dbin.zstd`

where `001` is the shard number, `000123123123` is the start-block,
and the rest is the end block.

Then spin out each shard ingestor in parallel. Once each shard
ingestor is done, it checks that all other shards are done, and writes
the FINAL line in the markers table to indicate that everything is
sync'd.

Now start the real-time ingestor, and go live.
