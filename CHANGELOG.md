# Change log

The format is based on
[Keep a Changelog](https://keepachangelog.com/en/1.0.0/), and this
project adheres to
[Semantic Versioning](https://semver.org/spec/v2.0.0.html). See
[MAINTAINERS.md](./MAINTAINERS.md) for instructions to keep up to
date.

# Unreleased

## System Administration Changes

### Added

* Added support for environment variable `EOSWS_PUSH_V1_OUTPUT=true` that forces push-transaction (guarantee:in-block) to output the same content format as nodeos 2.0.x (with Inlines)
* Added support for environment variable `DSTORE_S3_BUFFERED_READ=true` that forces reading S3 files (ex: blocks) ahead of processing, useful when S3 provider has trouble keeping long connections alive.
* Added support for looking up irreversible blocks on blockmeta (when the LIB was stuck for a while) from tokenmeta and trxdb-loader. They now use '--common-blockmeta-addr' flag if available
* Added `--common-chain-core-symbol` flag to define actual chain core symbol in the form `<precision>,<symbol code>` defaults to `4,EOS` by default.
* Added `--tokenmeta-readiness-max-latency` with default=5m, now tokenmeta will show as "NotServing" through grpc healthcheck if last processed block (HEAD) is older than this. Value of 0 disables that feature.
* Added `--relayer-source-request-burst` with default=90 to allow a relayer connecting to another relayer to request a 'burst'
* Added `--statedb-disable-indexing` to disable indexing of tablet and injecting data into storage engine **developer option, don't use that in production**.
* Added `--eosws-nodeos-rpc-push-extra-addresses` to allow providing a list of backup EOS addresses when push-transaction does not succeed in getting the transaction inside a block (with push_guarantee)
* Added `--eosws-max-stream-per-connection` to allow changing how many stream can be active at the same time for a given WebSocket connection, defaults to `12` which was the hard-coded value.
+ Added `--eosws-statedb-proxy-retries`, Number of time to retry proxying a request to statedb before failing (default 2)
+ Added `--eosws-nodeos-rpc-proxy-retries`, Number of time to retry proxying a request to statedb before failing (default 2)
* Added `--mindreader-max-console-length-in-bytes` which is the limit in bytes that we allow action trace's console output to be before truncating them.
* Environment variable `MINDREADER_MAX_TOKEN_SIZE` can now be set to override `bufio.Scanner()` max token size (default `52428800`, i.e. `50Mb`) for EOSIO chains with huge transactions
* Flag `--accounthist-mode` to specific the accounthist mode of operation
* Added `tools check accounthist-shards` to
* Flag `--common-include-filter-expr`, `--common-exclude-filter-expr`, `--common-system-actions-include-filter-expr` can optionally specify multiple values, separated by `;;;` and prefixed by `#123;` where 123 is a block number at which we stat applying that filter
* Added `accounthist` tools allows you to scan and read accounts `dfuseeos tools accounthist read ...` `dfuseeos tools accounthist scan ...`
* Flag `--search-router-truncation-low-block-num` to make the router aware of lower-block-truncation and serve requests accordingly
* Flag `--mindreader-oneblock-suffix` that mindreaders can each write their own file per block without competing for writes. https://github.com/dfuse-io/dfuse-eosio/issues/140
* Flag `--eosws-disabled-messages` a comma separated list of ws messages to disable.
* Flag `--common-system-shutdown-signal-delay`, a delay that will be applied between receiving SIGTERM signal and shutting down the apps. Health-check for `eosws` and `dgraphql` will respond 'not healthy' during that period.

### Removed

* **Breaking Change** Removed `--eosq-price-ticker-name` flag, if you were using this flag, please use `--common-chain-core-symbol` instead to define it.
* Removed `dgraphql-graceful-shutdown-delay`, it was a left-over, unused. Must use `--common-system-shutdown-signal-delay` now
* Removed `relayer-max-drift` (now dependent on a new condition of presence of a "block hole", and no new block sent for 5 seconds)
* Removed `relayer-init-time` (no need for it with this new condition ^)

### Changed

* The `--eosq-available-networks` `logo` field each network now has a maximum height of `70px`.
* The `--eosq-available-networks` config of each network now accepts a `logo_text` that when present, is displayed alongside the `logo` field. This field is taken into consideration only when `logo` is defined. In this mode, the logo is fixed to `48px x 48px`. If the `logo_text` value is `eosq`, this is rendered like the standard `eosq` logo.
* **Breaking Change** Changes to `--eosq-available-networks` config might be required around the `logo` field each network. You must now remove the `logo` field if it's not pointing to an existing image otherwise, the logo will not be rendered correctly.
* Applying a block filter over previously-filtered-blocks does not panic anymore, it applies the new filter on top of it, only if that specific filter has never been applied before. Applied filters definitions are concatenated in the block metadata, separated by `;;;`
* Default `trxdb-loader-batch-size` changed to 100, Safe to do so because it does not batch when close to head.
* Improved relayer mechanics: replaced "max drift" detection by "block hole" detection and recovery action is now to restart the joining source (instead of shutting down the process)
* Improved `dfuseeos tools check statedb-reproc-injector` output by showing all shard statistics (and not just most highest block).
* **Breaking Change** Changed `--statedb-enable-pipeline` flag to `--statedb-disable-pipeline` to make it clearer that it should not be disable, if you were using the flag, change the name and invert the logical value (i.e. `--state-enable-pipeline=false` becomes `--state-disable-pipeline=true`)

### Fixed
* Fixed validation of transaction ID passed to WebSocket `get_transaction` API, the prior validation was too permissive.
* Fixed a bug making search-forkresolver useless, because ignored by search-router.
* Fixed a bug on StateDB server not accepting symbol and symbol code as `scope` parameter value.
* Fixed shutdown on dgraphql (grpc/http) so it closes the active connections a little bit more gracefully.
* Fixed a bug in `TiKV` store implementation preventing it to delete keys correctly.
* Fixed a bug in `eosws` WebSocket `get_transaction_lifecycle` where a transaction not yet in the database would never stream back any message to the client.
* Fixed a bug with `--mindreader-no-blocks-log` option actually not being picked up (always false)
* Fixed a bug with `/state/table/row` not correctly reading row when it was in the table index.
* Fixed a bug with `/state/tables/scopes` where the actual block num used to query the data was incorrect leading to invalid response results.
* Fixed a bug with gRPC `dfuse.eosio.statedb.v1/State#StreamMultiScopesTableRows` where the actual block num used to query the data was incorrect leading to invalid response results.
* Fixed issue when reading ABI from StateDB where speculative writes were not handled correctly.
* Fixed issue when reading Table Row from StateDB where speculative writes were not handled correctly.
* Fixed a potential crash when reading ABI from StateDB and it does not exist in database.

# [v0.1.0-beta8] 2020-08-08
* fix **experimental** netkv implementation for statedb

# [v0.1.0-beta7] 2020-12-07
* fix **experimental** netkv implementation for trxdb

# [v0.1.0-beta6] 2020-08-27
* When using filtering capabilities, only absolutely required system actions will be indexed/processed.
* Added missing `updateauth` and `deleteauth` as require system actions in flag `common-system-actions-include-filter-expr`.

# [v0.1.0-beta5] 2020-08-24

## PUBLIC API Changes

### Added
* Added `tokens`, `accountBalances`, `tokenBalances` calls to dgraphql (based on tokenmeta)

## System Administration Changes

### Changed
* **Breaking Change** FluxDB has been extracted to a dedicated library (github.com/dfuse-io/fluxdb) with complete re-architecture design.
* **Breaking Change** FluxDB has been renamed to StateDB and is incompatible with previous written data. See [FluxDB Migration](#fluxdb-to-statedb-migration) section below for more details on how to migrate.
* `merger` startblock behavior changed, now relies on state-file, see https://github.com/streamingfast/merger/issues/1
* Changed `merger-seen-blocks-file` flag to `merger-state-file` to reflect this change.
* `merger` now properly handles storage backend errors when looking for where to start
* `mindreader` now automatically produces "merged blocks" instead of "one-block-files" when catching up (based on blocktime or if a blockmeta is reachable at `--common-blockmeta-addr`)
* `mindreader` now sets optimal EOS VM settings automatically if the platform supports it them when doing `dfuseeos init`.
* Changed `--abicodec-export-cache-url` flag to `abicodec-export-abis-base-url` and will contain only the URL of the where to export the ABIs in JSON.
* Changed `--abicodec-export-cache` flag to `abicodec-export-abis-enabled`.

### Added
* Added `tokenmeta` app, with its flags
* Added support for **filtered** *blocks*, *search indices* and *trxdb*, with `--common-include-filter-expr` and `--common-include-filter-expr`.
* Added `merged-filter` app (not running by default), that generates filtered merged blocks files from regular merged blocks files.
* Added truncation handling to `trxdb-loader`, which will only keep a _moving window of data_ in `trxdb`, and delete preceding transactions. Enabling that feature requires reprocessing trxdb.
  * `--trxdb-loader-truncation-enabled`
  * `--trxdb-loader-truncation-window`
  * `--trxdb-loader-truncation-purge-interval`
* Added `--statedb-reproc-shard-scratch-directory` to run StateDB reprocessing sharder using scratch directory to reduce RAM usage
* Added `--merger-one-block-deletion-threads` (default:10) to allow control over one-block-files deletion parallelism
+ Added `--merger-max-one-block-operations-batch-size` to allow control over one-block-files batches that are looked up on storage,
* Added `--eosws-with-completion` (default: true) to allow control over that feature
* Added `--mindreader-merge-threshold-block-age` when processing blocks with a blocktime older than this threshold, they will be automatically merged: (default 12h)
* Added `--mindreader-batch-mode` to force always merging blocks (like --mindreader-merge-and-store-directly did) AND overwriting existing files in destination.
* Added `--mindreader-wait-upload-complete-on-shutdown` flag to control how mindreader waits on upload completion when shutting down (previously waited indefinitely)
* Added `--search-live-hub-channel-size` flag to specific the size of the search live hub channel capacity
* Added `--search-live-preprocessor-concurrent-threads`: number of thread used to run file source preprocessor function
* Added `--abicodec-export-abis-file-name`, contains the URL where to export the ABIs in JSON
* Added `--metrics-listen-addr` to control on which address to server the metrics API (Prometheus), setting this value to an empty string disable metrics serving.
* Added `--dashboard-metrics-api-addr` to specify a different API address where to retrieve metrics for the dashboard.
* Added **Experimental** support for kvdb backend `netkv://`, an extremely simple network layer over `badger` to allow running dfuse components in separate instances.

### Removed
* The `--merger-delete-blocks-before` flag is now removed and is the only behavior for merger.
* The `--mindreader-merge-and-store-directly` flag was removed. That behavior is now activated by default when encountering 'old blocks'. Also see new flag mindreader-batch-mode.
* The `--mindreader-discard-after-stop-num` flag was removed, its implementation was too complex and it had no case where it was really useful.
* The `--mindreader-producer-hostname` flag was removed, this option made no sense in the context of `mindreader` app.
* The `--eosq-disable-tokenmeta` flag was removed, token meta is now included, so this flag is now obsolete.
* The `--eosq-on-demand` flag was removed, this was unused in the codebase.

### Fixed
* Fixed issue where `blockmeta` was not serving on GRPC at all because it couldn't figure out where to start on the stream
* Fixed issue with `merger` with a possible panic when reading a one-block-file that is empty, for example on a non-atomic storage backend
* Fixed issue with `mindreader` not stopping correctly (and showing any error) if the bootstrap phase (ex: restore-from-snapshot) failed.
* Fixed issue with `pitreos` not taking a backup at all when sparse-file extents checks failed.
* Fixed issue with `dfuseeos tools check merged-blocks` (start/end block, false valid ranges when the first segment is not 0, etc.)
* Improved performance by using value for `bstream.BlockRef` instead of pointers and ensuring we use the cached version.
* `mindreader` and `node-manager` improved `nodeos` log handling

#### FluxDB to StateDB Migration

FluxDB required an architecture re-design to fit with our vision about the tool and make it chain agnostic (so
it is easier to re-use on our other supported chain).

The code that was previously found here has been extracted to its own library (https://github.com/dfuse-io/fluxdb).
There is now a new app named StateDB (`statedb` is the app identifier) in dfuse for EOSIO that uses FluxDB to
support all previous API endpoints served by the FluxDB app as well as now offering a gRPC interface.

While doing this, we had to change how keys and data were written to the underlying engine. This means that all
your previous data stored cannot be read anymore by the new StateDB and that new data written by StateDB will
not be compatible on a previous instance.

What that means exactly is that StateDB will require to re-index all the current merged blocks up to live blocks
before being able to serve requests. This is the main reason why we decided to rename the app, so you are forced
to peform this step.

Here the steps required to migrate to the new `statedb` app:

1. We strongly recommend that you take a full backup of your data directory (while the app is shut down)
2. Launch a stand-alone stateDB instance in 'inject-mode' that reads from your block files and writes to a new location (see '--statedb-store-dsn')
3. Let it complete the "catch up" until it is very close to the HEAD of your network, then stop that instance.
4. Stop your previous instance (that uses fluxdb),
5. Copy the content of your statedb database to a location accessible from there (that you will define in '--statedb-store-dsn')
6. Launch the new version of the code, with the modified flags, over your previous data, including the new statedb database content (see below for the necessary flag and config modifications)

Here are the flags and config modifications required for switching to `fluxdb` to `statedb`, once the new data has been generated.

- In your `dfuse.yaml` config, under replace the `fluxdb` app by `statedb` and all flags prefixed with
  `fluxdb-` must now be prefixed with `statedb-`.

  From:

  ```yaml
  start:
  args:
  - ...
  - fluxdb
  - ...
  flags:
    ...
    fluxdb-http-listen-addr: :9090
    ...
  ```

  To:

  ```yaml
  start:
  args:
  - ...
  - statedb
  - ...
  flags:
    ...
    statedb-http-listen-addr: :9090
    ...
  ```

- If you had a customization for `fluxdb-statedb-dsn`, you must first renamed it `statedb-store-dsn`
  to `statedb-store-dsn`. **Important** You must use a new fresh database, so update your argument
  so it points to a new database, ensuring we don't overwrite data over old now incompatible data.
  If you did not customize the flag, continue reading, the default value has changed to point to a
  fresh storage folder.

  If you had a customization for `eosws-flux-addr`, rename to `eosws-statedb-grpc-addr` and ensure it
  points to the StateDB GRPC address (and not the HTTP address), its value must be the same as the flag
  `statedb-grpc-listen-addr`.

- If you have a custom `fluxdb-max-threads` flag, removed it, customizing this value is not supported
  anymore.

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
