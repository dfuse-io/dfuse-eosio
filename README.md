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

Build requirements:
* `Go` 1.13 or higher
* `Node.js` 12 or higher, `yarn`
* [rice](https://github.com/GeertJohan/go.rice) Go static assets embedder (see installation instructions below)

```
# Install `rice` CLI tool if you don't have it already
go get github.com/GeertJohan/go.rice/rice

git clone https://github.com/dfuse-io/dfuse-eosio
cd dfuse-eosio

pushd dashboard/client
  yarn install && yarn build
popd

pushd eosq
  yarn install && yarn build
popd

go generate ./dashboard
go generate ./eosq/app/eosq

go install -v ./cmd/dfuseeos
```

This will install the binary in your `$GOPATH/bin` folder (normally
`$HOME/go/bin`). Make sure this folder is in your `PATH` env variable.

### Usage (creating a new local chain)

1. Initialize a few configuration files in your working directory (`dfuse.yaml`, `mindreader/config.ini`, ...)

```
dfuseeos init
```

2. Boot your instance with:

```
dfuseeos start
```

3. A terminal prompt will list the graphical interfaces with their relevant links:

```
Dashboard: http://localhost:8081
Explorer & APIs:  http://localhost:8080
GraphiQL:         http://localhost:8080/graphiql
```

  * If dfuse is starting a new chain, two nodeos instances will now be running on your machine, a block producer node and a mindreader node, and the dfuse services should be ready in a matter of seconds.
  * If you chose to sync to an existing chain, only the mindreader node will launch. It may take a while for the initial sync depending on the size of the chain and the services may generate various error logs until it catches up. (More options for quickly syncing with an existing chain will be proposed in coming releases.)

4. If you chose to have dfuse create a new chain for you, see [bootstrapping](./bootstrapping) for info on creating the initial accounts and interacting with the chain

### Usage (syncing existing chain)

* See [Syncing a chain partially](./PARTIAL_SYNC.md)
* See the following issue about the complexity of [syncing a large chain](https://github.com/dfuse-io/dfuse-eosio/issues/26)

### Logging

See [Logging](./Logging.md)

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

## Troubleshooting

See [Troubleshooting](./TROUBLESHOOTING.md) section

## Contributing

Please read [CONTRIBUTING.md](CONTRIBUTING.md) for details on our Code of Conduct, [CONVENTIONS.md](CONVENTIONS.md) for coding conventions, and processes for submitting pull requests.

## License

[Apache 2.0](LICENSE)

## References

- [dfuse Docs](https://docs.dfuse.io)
- [dfuse on Telegram](https://t.me/dfuseAPI) - community & team support
