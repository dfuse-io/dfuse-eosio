# FluxDB

An historical database for tabular data with loader for EOSIO state table & permissions.

**Important** While this is used in production currently at dfuse, this repository will
undergo a major refactoring in the upcoming months. We will revisit the project, its
design, its role and the overall architecture. It will most probably be split into
smaller pieces, a core module that does storage of historical data in a really agnostic
fashion, even blockchain agnostic and a second component that uses the core API to
implement the current set of features in EOSIO.

## Installation

Install through [dfuse for EOSIO](..)

## Usage

Assuming the following has been run already:

    export FLUXDB_BASE_URL=https://mainnet.eos.dfuse.io

Get currency balances for user's account `eoscanada` (specified as `scope`) on various currencies (specified as `accounts`).

    curl "$FLUXDB_BASE_URL/v0/state/tables/accounts?block_num=26415000&accounts=eosio.token|eosadddddddd|tokenbyeocat|ethsidechain|epraofficial|alibabapoole|hirevibeshvt|oo1122334455|irespotokens|publytoken11|parslseed123|trybenetwork|zkstokensr4u&scope=eoscanadacom&table=accounts&json=true&&token=$DFUSE" | jq .

## Features

FluxDB supports parallel ingestion by doing a first sharding pass on
the chain history, splitting the different tables being mutated in
different shards, then ingesting linearly each table.

This allows ingestion of the whole history in a few hours.


## Documentation

See the `/v0/state` endpoints under https://docs.dfuse.io/reference/eosio/rest/

## Contributing

*Issues and PR related to FluxDB belong to this repository*

See the dfuse-wide
[contribution guide](https://github.com/dfuse-io/dfuse#contributing)
if you wish to contribute to this code base.

## License

[Apache 2.0](LICENSE)
