# dfuse for EOSIO

All [dfuse](https://dfuse.io) services for EOSIO, running from your laptop, as a single binary.


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

Also, install our instrumented `nodeos` node in the section below.

### Build from source

```
git clone git@github.com:dfuse-io/dfuse-eosio
cd dfuse-eosio
go install -v ./cmd/dfuseeos
```

### Usage

Make sure you have our dfuse instrumented `nodeos` binary on your machine, a fork
of the standard `EOSIO` software. On **Mac OS X**, you can simply do:

    brew install dfuse-io/tap/eosio

For other platforms, check the [Prebuilt Binaries Instructions](#dfuse-Instrumented-EOSIO-Prebuilt-Binaries)
section for installation details.

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

### dfuse Instrumented EOSIO Prebuilt Binaries

dfuse uses a specialized EOSIO binary that contains instrumentation required
to extract the data needed to power all dfuse's features.

The current source code can be found on branch [release/2.0.x-dm](https://github.com/dfuse-io/eos/tree/release/2.0.x-dm)
under [github.com/dfuse-io/eos](https://github.com/dfuse-io/eos) fork of EOSIO software.

**Note** It is safe to use this forked version as a replacement for your current installation, all
special instrumentations are gated around a config option (i.e. `deep-mind = true`) that is off by
default.

#### Mac OS X:

##### Mac OS X Brew Install

```sh
brew tap dfuse-io/tap/eosio
```

##### Mac OS X Brew Uninstall

```sh
brew remove eosio
```

#### Ubuntu Linux:

##### Ubuntu 18.04 Package Install

```sh
wget https://github.com/dfuse-io/eos/releases/download/v2.0.3-dm/eosio_2.0.3-dm_ubuntu-18.04_amd64.deb
sudo apt install ./eosio_2.0.3-dm_ubuntu-18.04_amd64.deb
```

##### Ubuntu 16.04 Package Install

```sh
wget https://github.com/dfuse-io/eos/releases/download/v2.0.3-dm/eosio_2.0.3-dm_ubuntu-16.04_amd64.deb
sudo apt install ./eosio_2.0.3-dm_ubuntu-16.04_amd64.deb
```

##### Ubuntu Package Uninstall

```sh
sudo apt remove eosio
```

#### RPM-based (CentOS, Amazon Linux, etc.):

##### RPM Package Install

```sh
wget https://github.com/dfuse-io/eos/releases/download/v2.0.3-dm/eosio_2.0.3-dm.el7.x86_64.rpm
sudo yum install ./eosio_2.0.3-dm.el7.x86_64.rpm
```

##### RPM Package Uninstall

```sh
sudo yum remove eosio
```

## Built with

- Go
- Protobuf data models & gRPC services
- All those dfuse pieces being open-sourced

## Contributing

Please read [CONTRIBUTING.md] for details on our Code of Conduct, [CONVENTIONS.md] for coding conventions, and processes for submitting pull requests.

## License

[Apache 2.0](LICENSE)

## References

- [dfuse Docs](https://docs.dfuse.io)
- [dfuse on Telegram](https://t.me/dfuseAPI) - community & team support


## Debugging

It's possible to debug `dfuseeos` by providing multiple times the
verbosity flags, like `-vvv` which enables debugging verbosity level
3 (default is 0).

Here the default level per package(s) for the various verbosity level
as well as formatting rules in place depending on the verbosity level.

More higher is the verbosity level, more debugging statements and more
each log line has contextual information to help debugging.

#### Verbosity 0 (no flag)

Level:

- INFO `github.com/dfuse-io/dfuse-eosio`
- INFO `github.com/dfuse-io/dfuse-eosio/cmd/dfuseeos`
- <Hidden> All others

Formatting:

- No level displayed
- No logger name displayed
- Caller displayed when log entry level >= WARN
- Stacktrace displayed when log entry level >= ERROR (if present)

#### Verbosity 1 (-v)

Level:

- INFO `github.com/dfuse-io/dfuse-eosio`
- INFO `github.com/dfuse-io/dfuse-eosio/cmd/dfuseeos`
- WARN `github.com/dfuse-io/manageos.*`
- INFO All others

Formatting:

- Level always displayed
- Logger name always displayed
- Caller displayed when log entry level >= WARN
- Stacktrace displayed when log entry level >= ERROR (if present)

#### Verbosity 2 (-v)

Level:

- DEBUG `github.com/dfuse-io/dfuse-eosio`
- DEBUG `github.com/dfuse-io/dfuse-eosio/cmd/dfuseeos`
- INFO `github.com/dfuse-io/manageos.*`
- INFO All others

Formatting:

- Level always displayed
- Logger name always displayed
- Caller displayed when log entry level >= WARN
- Stacktrace always displayed (if present)

#### Verbosity 3 (-v)

Level:

- DEBUG All packages

Formatting:

- Level always displayed
- Logger name always displayed
- Caller always displayed, but package version trimmed out
- Stacktrace always displayed (if present)

#### Verbosity 4+ (-vvvv [and more])

Level:

- DEBUG All packages

Formatting:

- Level always displayed
- Logger name always displayed
- Caller always displayed, full path, with package version present
- Stacktrace always displayed (if present)

#### Environment variable `DEBUG="<regexp>"`

Overrides behavior of verbosity for packages matching the `<regexp>` value
as well as changing the formatting rules. For example, you can run:

```
DEBUG="github.com/dfuse-io/bstream.*" dfuseeos start
```

Which will keep the level behavior of verbosity 0 but will set all loggers
registered matching `github.com/dfuse-io/bstream.*` to `DEBUG` level.

Formatting:

- Level always displayed
- Logger name always displayed
- Caller always displayed, but package version trimmed out (unless -vvvv or more present)
- Stacktrace always displayed (if present)
