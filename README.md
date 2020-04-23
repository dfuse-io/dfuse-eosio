# dfuse for EOSIO
[![reference](https://img.shields.io/badge/godoc-reference-5272B4.svg?style=flat-square)](https://pkg.go.dev/github.com/dfuse-io/dfuse-eosio)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)

All **[dfuse.io services](https://dfuse.io/technology)** for EOSIO,
running from your laptop or from a container, released as a single
statically linked binary: `dfuseeos`.

See the general [dfuse repository](https://github.com/dfuse-io/dfuse)
for other blockchain protocols implementations.


## Getting started

If it's the first time you boot a `nodeos` node, please review
https://developers.eos.io/welcome/latest/tutorials/bios-boot-sequence
and make sure you get a grasp of what this blockchain node is capable.

The default settings of `dfuseeos` allow you to quickly bootstrap a working
development chain by also managing the block producing node for you.

### Requirements

#### Operating System
* This software runs on Linux or Mac OS X

#### dfuse Instrumented nodeos (deep-mind)
* See [DEPENDENCIES.md](DEPENDENCIES.md) for instructions on how to get an instrumented `nodeos` binary.

#### Recommended tools
* [eosio.cdt tools](https://github.com/EOSIO/eosio.cdt)
* `cleos` (from [EOSIO/eos](https://github.com/EOSIO/eos)) or
* [eosc](https://github.com/eoscanada/eosc/releases).

### Installing

#### From a pre-built release

* Download a tarball from the [GitHub Releases Tab](https://github.com/dfuse-io/dfuse-eosio/releases).
* Put the binary `dfuseeos` in your `PATH`.

#### From source

```
# The minimum required Go version is 1.13
git clone https://github.com/dfuse-io/dfuse-eosio
cd dfuse-eosio
go install -v ./cmd/dfuseeos
```

This will install the binary in your `$GOPATH/bin` folder (normally
`$HOME/go/bin`). Make sure this folder is in your `PATH` env variable.

### Usage

Initialize a new `dfuse.yaml` config file (answer 'y' for a quick start) with:

    dfuseeos init

The created file will contain the private and public keys generated
for the booted chain.

Boot your instance with:

    dfuseeos start

If you answered 'y', this will boot a producer node, a reader node,
both communicating together, boot all dfuse services and expose a
dashboard and all the APIs.

A terminal prompt will list the graphical interfaces with their relevant links:

    Dashboard: http://localhost:8080
    GraphiQL: http://localhost:13019
    Eosq: http://localhost:8081

The **Dashboard** is a diagnostic tool to monitor the status of each
component of dfuse for EOSIO. It also provides a graph to visualize how far
their head block has drifted from the current block.

To run the dashboard in dev mode:

    cd dashboard/client
    yarn install
    yarn start

Head to the **GraphiQL** interface, and do a streaming search of
what's happening on chain! All dfuse GraphQL queries are available here.

**Eosq**, our high precision block explorer is also integrated in box.
Use it to test out search queries and view organized information
such as accounts, transaction data, and inline actions.
The built javascript folder is committed with the code for convenience.
See [eosq README](eosq/README.md) for build instructions


## Overview

Here's a quick map of this repository:

The glue:
* The [dfuseeos](./cmd/dfuseeos) binary.
* The [launcher](./launcher) which starts all the internal services

The EOSIO-specific services:
* [abicodec](./abicodec): ABI encoding and decoding service
* [fluxdb](./fluxdb): the **dfuse State** database for EOSIO, with all tables at any block height
* [kvdb-loader](./kvdb-loader): service that loads data into the `kvdb` storage
* [dashboard](./dashboard): server and UI for the **dfuse for EOSIO** dashboard.
* [eosq](./eosq): the famous https://eosq.app block explorer
* [eosws](./eosws): the REST, Websocket service, push guarantee, chain pass-through service.

dfuse Products's EOSIO-specific hooks and plugins:
* [search plugin](./search), object mappers, EOSIO-specific indexer, results mapper (along with the [search client](./search-client).
* [dgraphql resolvers](./dgraphql), with all data schemas for EOSIO
* [blockmeta plugin](./blockmeta), for EOS-specific `kvdb` bridge.


## Contributing

Please read [CONTRIBUTING.md](CONTRIBUTING.md) for details on our Code of Conduct, [CONVENTIONS.md](CONVENTIONS.md) for coding conventions, and processes for submitting pull requests.

## License

[Apache 2.0](LICENSE)

## References

- [dfuse Docs](https://docs.dfuse.io)
- [dfuse on Telegram](https://t.me/dfuseAPI) - community & team support
