# dfuse for EOSIO

All [dfuse.io](https://dfuse.io) services for EOSIO, running from your
laptop or from a container, released as a single statically linked
binary.

See the general [dfuse repository](https://github.com/dfuse-io/dfuse)
for other blockchain protocols implementations.


## Getting started

### Prerequisites

If it's the first time you boot a `nodeos` node, please review
https://developers.eos.io/welcome/latest/tutorials/bios-boot-sequence
and make sure you get a grasp of what this blockchain node is capable.

You might want to have the [eosio.cdt tools](https://github.com/EOSIO/eosio.cdt)
installed, as well as `cleos` (from [EOSIO/eos](https://github.com/EOSIO/eos)) or
[eosc](https://github.com/eoscanada/eosc/releases).

### Install

Get a release from the _Releases_ tab in GitHub. Install the binary in your `PATH`.

See [INSTALL.md](INSTALL.md) to install the dependencies (like an instrumented `nodeos`).


#### Build from source

```
git clone git@github.com:dfuse-io/dfuse-eosio
cd dfuse-eosio
go install -v ./cmd/dfuseeos
```


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

Please read [CONTRIBUTING.md] for details on our Code of Conduct, [CONVENTIONS.md] for coding conventions, and processes for submitting pull requests.

## License

[Apache 2.0](LICENSE)

## References

- [dfuse Docs](https://docs.dfuse.io)
- [dfuse on Telegram](https://t.me/dfuseAPI) - community & team support
