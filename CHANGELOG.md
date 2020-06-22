# Change log

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## Unreleased

### Networked APIs Changed

* **BREAKING**: `eosws` transaction lifecycle field `creation_irreversible` changed to `dtrx_creation_irreversible`
* **BREAKING**: `eosws` transaction lifecycle field `cancelation_irreversible` changed to `dtrx_cancelation_irreversible`

### Changed
* In general: `eosdb` was renamed to `trxdb`, which shouldn't change much externally.
  * Specifically: the `healthz` endpoint's `eosdb` field is now
    `trxdb`, so you might need to adjust your monitoring.
* **BREAKING**: `search` flag `--search-common-dfuse-hooks-action-name` changed to `--search-common-dfuse-events-action-name`
* **BREAKING**: `abicodec` app default value for `abicodec-cache-base-url` and `abicodec-export-cache-url` flags was changed to `{dfuse-data-dir}/storage/abicache` (fixing a typo in `abicahe`). To remain compatible, simply do a rename manually on disk before starting the update version (`mv dfuse-data/storage/abicahe {dfuse-data-dir}/storage/abicache`).
* **BREAKING**: `fluxdb` Removed `fluxdb-enable-dev-mode` flag, use `fluxdb-enable-live-pipeline=false` to get the same behavior as before.
* `mindreader` ContinuityChecker is not enabled by default anymore
* `dfuseeos tools check blocks` was renamed to `dfuseeos tools check merged-blocks`
* `search` roarCache now based on a normalized version of the query string (ex: `a:foo b:bar` is now equivalent to `b:bar a:foo`, etc.)
* Various startup speed improvements for `blockmeta`, `bstream`, `search-indexer`

### Removed
* Removed `search-indexer-num-blocks-before-start` flag from `search-indexer`, search-indexer automatically resolved its start block

### Added
* Added app called mindreader-stdin, which simply produces one-block-files (or merged-blocks-files) based on stdin, without trying to manage nodeos.
* Command `dmesh` to `tools` with flags `dsn` & `service-version` to inspect dmesh search peers. It currently only supports etcd.
* Added `booter` application with its flags.
* Flag: `--node-manager-auto-backup-hostname-match` If non-empty, auto-backups will only trigger if os.Hostname() return this value
* Flag: `--node-manager-auto-snapshot-hostname-match` If non-empty, auto-backups will only trigger if os.Hostname() return this value
* Flags `--mindreader-auto-backup-hostname-match` and `--node-manager-auto-snapshot-hostname-match` (identical to node-manager flags above)
* Flag: `--mindreader-fail-on-non-contiguous-block` (default:false) to enable the ContinuityChecker
* Flag: `--log-level-switcher-listen-addr` (default:1065) to change log level on a running instance (see DEBUG.md)
* Flag: `--common-ratelimiter-plugin` (default: null://) to enable a rate limiter plugin
* Flag: `--pprof-listen-addr` (default: 6060)
* Flag: `--search-common-dfuse-events-unrestricted` to lift all restrictions for search dfuse Events (max field count, max key length, max value length)
* Command `kv` to `tools` with sub command `get`, `scan`, `prefix`, `account`, `blk`, `blkirr`, `trx`, `trxtrace` to retrieve data from trxdb
* Command `db` to `tools` with sub command `blk`, `trx` to retrieve data from trxdb
* Command `check trxdb-blocks` to `tools`, to ensure linearity of irreversible blocks in storage.  This is useful to know if you've missed some block ranges when doing parallel insertions into your `trxdb` storage.
* `trxdb` deduper now reduces storage by removing identical action data and calls the "reduper" to add this data back.

### Fixed
* `search-indexer` no longer overflows on negative startblocks on new chains, it fails fast instead.
* `search-archive` relative-start-block truncation now works
* `search-forkresolver` no longer throws a nil pointer (app was previously broken)


## [v0.1.0-beta3] 2020-05-13

### Added
* Added `--eosq-available-networks` json string to configure the network section of eosq.
``` [
     {
       "id": "id.1",
       "name": "Network Name",
       "is_test": false,
       "logo": "/images/network-logo.png",
       "url": "https://www.example.com/"
     },
   ]
```
* Added `--eosq-default-network` string to configure the default network eosq
* Added `--eosq-disable-analytics` bool to configure eosq analytics
* Added `--eosq-display-price` bool to configure if eosq displays prices
* Added `--eosq-price-ticker` string to configure if eosq price ticker
* Added `--eosq-on-demand` bool to configure if eosq serves an on-demand network
* Added `--eosq-disable-tokenmeta` bool to configure if eosq disables tokenmenta

* **BREAKING**: To improve dfuse instrumented `nodeos` binary processing speed, we had to make incompatible changes to data exchange format going out of `nodeos`. This requires you to upgrade your dfuse instrumented `nodeos` binary to latest version (https://github.com/dfuse-io/eos/releases/tag/v2.0.5-dm-12.0). Follow instructions in at https://github.com/dfuse-io/dfuse-eosio/blob/develop/DEPENDENCIES.md#dfuse-instrumented-eosio-prebuilt-binaries to install the latest version for your platform.
* **BREAKING**: `--mindreader-working-dir` default value is now `{dfuse-data-dir}/mindreader/work` instead of `{dfuse-data-dir}/mindreader` this is to prevent mindreader from walking files into the working dir and trying to upload and delete nodes system files like `fork_db.dat`
* Added `--eosq-environment` environment where eosq will run (local, dev, production)
* Added `--apiproxy-autocert-domains`, `--apiproxy-autocert-cache-dir` and `--apiproxy-https-listen-addr` to serve SSL directly from proxy.
* Added `--node-manager-number-of-snapshots-to-keep` and `--mindreader-number-of-snapshots-to-keep` to allow keeping a few (default:5) snapshots only in the store

### Removed
* Removed the `--merger-store-timeout` flag.  Not needed anymore, as some sensible timeouts have been put here and there, using the latest `dstore@v0.1.0` that is context-aware.

### Changed

* **BREAKING**: flag `--node-manager-auto-restore` (bool) replaced with `--node-manager-auto-restore-source` (string)
* **BREAKING**: flag `--mindreader-auto-restore` (bool) replaced with `--mindreader-auto-restore-source` (string)
* Mindreader now has "producer" plugin enabled to allow taking snapshots
* Mindreader now runs with "NoBlocksLog" option (deleting blocks.log on restart)
* Node-manager and Mindreader now make dfuseeos shutdown when nodeos crashes.
* Node-manager and Mindreader now try to restore from snapshot if they crash within 10 seconds of starting (ex: dirty state)
* fixes dmetrics duplicate registration error (race condition)
* We improve by 4x times the performance of dfuse instrumented `nodeos` binary on heavy EOS Mainnet blocks. This required changes to `nodeos` data exchange format, so you will need to upgrade it, see the `Breaking` change entry at top of this section.
* Fixed behavior of `--eosq-api-endpoint-url` to allow specifying protocol (ex: https://api.mydomain.com)
* The `kvdb-loader` application was renamed `trxdb-loader`. In general what was (confusingly) named `kvdb` is now `trxdb`, so that `kvdb` can now take on its full meaning of a lean key-value storage abstraction (which is also used by FluxDB).
   * All `--kvdb-loader` flags have been renamed to `--trxdb-loader`.
   * Metrics ID for `kvdb-loader` has been changed to `trxdb-loader` (check your dashboards)
* The `--mindreader-merge-and-upload-directly` was renamed to `--mindreader-merge-and-store-directly`.
* `--common-blocks-store-url` now replaces all of these flags:
    * `--mindreader-merged-blocks-store-url`, `--relayer-blocks-store`, `--fluxdb-blocks-store`, `--kvdb-loader-blocks-store`, `--blockmeta-blocks-store`, `--search-indexer-blocks-store`, `--search-live-blocks-store`, `--search-forkresolver-blocks-store`, `--eosws-blocks-store`
* `--common-oneblock-store-url` now replaces these flags:
    *  `--mindreader-oneblock-store-url`, `--merger-one-block-path`
* `--common-backup-store-url` now replaces these flags:
    * `--node-manager-backup-store-url`, `--mindreader-backup-store-url`
* `--search-common-indices-store-url` now replaces these flags:
    * `--search-indexer-indices-store`, `--search-archive-indices-store`
* `--common-blockstream-addr` now replaces these flags:
    * `--fluxdb-block-stream-addr`, `--kvdb-loader-block-stream-addr`, `--blockmeta-block-stream-addr`, `--search-indexer-block-stream-addr`, `--search-live-block-stream-addr`, `--eosws-block-stream-addr`
* `--common-blockmeta-addr` now replaces these flags:
    * `--search-indexer-blockmeta-addr`, `--search-router-blockmeta-addr`, `--search-live-blockmeta-addr`, `--eosws-block-meta-addr`, `--dgraphql-block-meta-addr`
* `--common-network-id` now replaces this flag: `--dgraphql-network-id`
* `--common-auth-plugin` now replaces these flags:
    * `--dgraphql-auth-plugin`, `--eosws-auth-plugin`
* `--fluxdb-statedb-dsn` replaces `--fluxdb-kvdb-store-dsn` (to avoid confusion between what's actually stored in `kvdb` and how FluxDB is using it (as a simple kv store).
* `--trxdb-loader-parallel-file-download-count` replaces `--kvdb-parallel-file-download-count`
* `--common-trxdb-dsn` replaces these flags:
    * `--blockmeta-kvdb-dsn`, `--abicodec-kvdb-dsn`, `--eosws-kvdb-dsn`, `--dgraphql-kvdb-dsn`, `--kvdb-loader-kvdb-dsn`

* Changed default value for storage URL for `fluxdb` and `kvbd` (now named `trxdb`)

### Fixed

* `--kvdb-loader-chain-id` not being taken into account. This affected the decoding of public keys during the `kvdb` loading process.


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
