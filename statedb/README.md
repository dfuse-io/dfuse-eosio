# StateDB

An historical database for tabular data with loader for EOSIO state table & permissions.

## Installation

Install through [dfuse for EOSIO](..)

## Features

StateDB supports parallel ingestion by doing a first sharding pass on
the chain history, splitting the different tables being mutated in
different shards, then ingesting linearly each table.

This allows ingestion of the whole history in a few hours.

## Documentation

See the `/v0/state` endpoints under https://docs.dfuse.io/reference/eosio/rest/

## Contributing

*Issues and PR related to FluxDB belong to this repository*

See the dfuse-wide [contribution guide](https://github.com/dfuse-io/dfuse#contributing)
if you wish to contribute to this code base.
