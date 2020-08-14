# Change log

The format is based on
[Keep a Changelog](https://keepachangelog.com/en/1.0.0/), and this
project adheres to
[Semantic Versioning](https://semver.org/spec/v2.0.0.html). See
[MAINTAINERS.md](./MAINTAINERS.md) for instructions to keep up to
date.

# [Unreleased]

### Added
* Added `--merger-one-block-deletion-threads` (default:10) to allow control over one-block-files deletion parallelism
+ Added `--merger-max-one-block-operations-batch-size` to allow control over one-block-files batches that are looked up on storage,
* Added `--eosws-with-completion` (default: true) to allow control over that feature
* Added `--mindreader-merge-threshold-block-age` when processing blocks with a blocktime older than this threshold, they will be automatically merged: (default 12h)
* Added `--mindreader-batch-mode` to force always merging blocks (like --mindreader-merge-and-store-directly did) AND overwriting existing files in destination.
* Added `search-live-hub-channel-size` flag to specific the size of the search live hub channel capacity
* Added `--mindreader-wait-upload-complete-on-shutdown` flag to control how mindreader waits on upload completion when shutting down (previously waited indefinitely)
* Added `merged-filter` application (not running by default), that takes merged blocks files (100-blocks files), filters them according to the `--common-include-filter-expr` and `--common-include-filter-expr`.
* Added `tokenmeta` application, with its flags
* Add `--search-live-preprocessor-concurrent-threads` Number of thread used to run file source preprocessor function
* flag `abicodec-export-abis-file-name` will contain only the URL of the where to export the ABIs in JSON
* Added `--metrics-listen-addr` to control on which address to server the metrics API (Prometheus), setting this value to an empty string disable metrics serving.
* Added `--dashboard-metrics-api-addr` to specify a different API address where to retrieve metrics for the dashboard.
* Experimental support for `netkv://127.0.0.1:1234` as a possible `kvdb` database backend, which allows decoupling of single pods deployment into using an extremely simple networked k/v store, using the same badger backend and database as when you boot with default parameters.
* Truncation handling to `trxdb-loader`, which will only keep a _moving window of data_ in `trxdb`, based on a window of blocks. Uses flags:
  * `--trxdb-loader-truncation-enabled`
  * `--trxdb-loader-truncation-window`
  * `--trxdb-loader-truncation-purge-interval`

### Removed
* The `--mindreader-merge-and-store-directly` flag was removed. That behavior is now activated by default when encountering 'old blocks'. Also see new flag mindreader-batch-mode.
* The `--mindreader-discard-after-stop-num` flag was removed, its implementation was too complex and it had no case where it was really useful.
* The `--eosq-disable-tokenmeta` flag was removed, token meta is now included, so this flag is now obsolete.
* The `--eosq-on-demand` flag was removed, this was unused in the codebase.
* The `--mindreader-producer-hostname` flag was removed, this option made no sense in the context of `mindreader` app.

### Changed
* **Breaking Change** FluxDB has been extracted to a dedicated library (github.com/dfuse-io/fluxdb) with complete re-architecture design.
* **Breaking Change** FluxDB has been renamed to StateDB and is incompatible with previous written data. See [FluxDB Migration](#fluxdb-to-statedb-migration) section below for more details on how to migrate.
* Merger now deletes blocks that have been processed and are in his "seenBlocks" cache, so a mindreader restarting from recent snapshot will not clog your one-block-file storage
* Merger now properly handles storage backend errors when looking for where to start
* Mindreader now automatically start producing "merged blocks" instead of "one-block-files" the blocks are passed a certain age threshold (based on blocktime) OR if a "blockmeta" service can be reached and validates that the blocks numbers are lower than the Last Irreversible Block (LIB). When those conditions cease to be true - close to HEAD -,  it changes to producing one-block-files again (until restarted)
* Mindreader can now use flag `--common-blockmeta-addr` so it can determine the network LIB and start producing "merged blocks"
* Improved performance by using value for `bstream.BlockRef` instead of pointers and ensuring we use the cached version.
* EOS VM settings on mindreader are now automatically added if the platform supports it them when doing `dfuseeos init`.
* Fixed a bunch of small issues with `dfuseeos tools check merged-blocks` command, like inverted start/end block in detected holes and false valid ranges when the first segment is not 0. Fixed also issue where a leading `./` was not working as expected.
* Improved `nodeos` log interceptions (when using `(mindreader|node-manager)-log-to-zap` flag) by adjusting log level for specific lines, that should improve the overall experience and better notice what is really an important error. More tweaking on the adjustment will continue as an iterative process, don't hesitate to report log line that should adjusted.
* Flag `abicodec-export-cache-url` changed to `abicodec-export-abis-base-url` and will contain only the URL of the where to export the ABIs in JSON.
* Flag `abicodec-export-cache` changed to `abicodec-export-abis-enabled`.

### Fixed
* Fixed issue with `merger` with a possible panic when reading a one-block-file that is empty, for example on a non-atomic storage backend
* Fixed issue with `mindreader` not stopping correctly (and showing any error) if the bootstrap phase (ex: restore-from-snapshot) failed.
* Fixed issue with `pitreos` not taking a backup at all when sparse-file extents checks failed.

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

- From an operator standpoint, what we suggest is to craft a `dfuse.yaml` config that starts StateDB
  only in inject mode only. You let this instance run until StateDB reaches the live block of your
  network.

  Once you have reached this point, you can now perform a switch to the new StateDB database. Stop
  the injecting instance. Stop your production instance, renaming old app id `fluxdb` to `statedb`
  (and all flags) then reconfigure it so the `statedb-store-dsn` points to the database populated
  by the injecting instance. At this point, you can restart your production node and continue
  normally using the new `statedb` app.

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
