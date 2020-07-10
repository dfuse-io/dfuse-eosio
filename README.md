# dfuse for EOSIO
[![reference](https://img.shields.io/badge/godoc-reference-5272B4.svg?style=flat-square)](https://pkg.go.dev/github.com/dfuse-io/dfuse-eosio)
[![License](https://img.shields.io/badxoge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)

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

#### Recommended auxiliary tools

* [eosio.cdt tools](https://github.com/EOSIO/eosio.cdt)
* `cleos` (from [EOSIO/eos](https://github.com/EOSIO/eos)) or
* [eosc](https://github.com/eoscanada/eosc/releases).

### Installing

#### From a pre-built release

* Download a tarball from the [GitHub Releases Tab](https://github.com/dfuse-io/dfuse-eosio/releases).
* Put the binary `dfuseeos` in your `PATH`.

See https://docs.dfuse.io/eosio/admin-guide/installation for more information.


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

# Build the javascript apps
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


### Usage

See the [System Admin Guide](https://docs.dfuse.io/eosio/admin-guide) of the documentation for instructions on how run `dfuseeos` on your laptop, synchronize large chains, do partial sync'ing, and the different deployment options available.



### Logging

See [Logging](./LOGGING.md)

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
