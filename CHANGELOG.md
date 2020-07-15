# Change log

The format is based on
[Keep a Changelog](https://keepachangelog.com/en/1.0.0/), and this
project adheres to
[Semantic Versioning](https://semver.org/spec/v2.0.0.html). See
[MAINTAINERS.md](./MAINTAINERS.md) for instructions to keep up to
date.

# [Unreleased]

### Added
* Added `tokenmeta` application, with its flags    
* flag `abicodec-export-abis-file-name` will contain only the URL of the where to export the ABIs in JSON
* Added `--metrics-listen-addr` to control on which address to server the metrics API (Prometheus), setting this value to an empty string disable metrics serving.
* Added `--dashboard-metrics-api-addr` to specify a different API address where to retrieve metrics for the dashboard.
* Experimental support for `netkv://127.0.0.1:1234` as a possible `kvdb` database backend, which allows decoupling of single pods deployment into using an extremely simple networked k/v store, using the same badger backend and database as when you boot with default parameters.

### Removed
The `--mindreader-producer-hostname` flag was removed, this option made no sense in the context of `mindreader` app.

### Changed
* flag `abicodec-export-cache-url` changed to `abicodec-export-abis-base-url` and will contain only the URL of the where to export the ABIs in JSON
* flag `abicodec-export-cache` change to `abicodec-export-abis-enabled`
* Experimental support for `netkv://127.0.0.1:1234` as a possible `kvdb` database backend, which allows decoupling of single pods deployment into using an extremely simple networked k/v store, using the same badger backend and database as when you boot with default parameters.

# [v0.1.0-beta4] 2020-06-23

See [release notes](https://github.com/dfuse-io/dfuse-eosio/releases/tag/v0.1.0-beta4).

## [v0.1.0-beta3] 2020-05-13

See [release notes](https://github.com/dfuse-io/dfuse-eosio/releases/tag/v0.1.0-beta3).

## [v0.1.0-beta2] 2020-04-27

### Added
* Added `apiproxy` application, with its flags
* Added `--log-format` option for JSON output and `log-to-file` bool (default to true, same behavior as before)
* Filtering (whitelist and blacklist) of what is indexed in Search, based on Google's Common Expression Language.  See [details here](./search/README.md). Added `--search-common-action-filter-on-expr` and `--search-common-action-filter-out-expr`.
    * NOTE: This doesn't affect what is extracted from the chain, allowing you to re-index selectively without a chain replay.

### Changed
* CLI: dfuseeos init now writes dfuse.yaml with the `start` command's flags, also the array of components to start
* CLI: new `{dfuse-data-dir}` replacement string in config flags, also changed default flag values
 * `--node-manager-config-dir` now `./producer` (was `manager/config`)
 * `--node-manager-data-dir` now `{dfuse-data-dir}/node-manager/data` (was `managernode/data`)
 * `--mindreader-config-dir` now `./mindreader` (was `mindreadernode/config`)
 * `--mindreader-data-dir` now `{dfuse-data-dir}/mindreader/data` (was `mindreadernode/data`)
* CLI: regrouped some flags:
 * `--search-indexer-dfuse-hooks-action-name`, `--search-live-dfuse-hooks-action-name`, `--search-forkresolver-dfuse-hooks-action-name` fused into new `--search-common-dfuse-hooks-action-name`.
 * `--search-...-mesh-publish-polling-duration` fused into new `--search-common-mesh-publish-polling-duration`.
 * all of the `--search-mesh-...` options were renamed to `--search-common-mesh-...` (previously `--search-mesh-service-version`, `--search-mesh-namespace`, `--search-mesh-store-addr`)
* `dashboard`: now separate metrics for mindreader vs producer node
* `dashboard` doesn't act as a reverse proxy anymore (`apiproxy` does)
* `dashboard`'s default port is now `:8081`
* `eosq`'s port is now proxied through `:8080`, so use that.
* App `manager` renamed to `node-manager`. All of its flags were changed from `--manager-...` to `--node-manager-...`

### Removed
* The `--search-...-indexing-restrictions-json`.  This was replaced by the filtering listed above.

## [v0.1.0-beta1] 2020-04-17

### Added
* Added `--nodeos-path` to control which `nodeos` executable is used.

### Removed
* Removed `--chain-name`, replaced by `--config-file` (or `-c`)
* Removed `init --reset` option, `dfusebox purge` does it all now.

### Changed
* Renamed `dfusebox` to `dfuse for EOSIO`
  * `dfusebox.yaml` to `dfuse.yaml`
  * `dfusebox-data` to `dfuse-data`
* `--data-dir` now defaults to `./dfusebox-data`, and is separate from a chain name or the config file location.
* `dfusebox init` now only generates a `dfusebox.yaml` config file, which can be booted with `dfusebox start`.
* `dfusebox init` now only has the interactive method. We can later add more programmatic method to boot chains.  With a `dfusebox.yaml` config now, however, we can reuse initializations multiple times.
* License changed to Apache 2.0
* Added GitHub workflow for PR checks
