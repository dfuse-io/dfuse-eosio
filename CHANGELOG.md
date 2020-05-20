# Change log

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [v0.1.0-beta3] 2020-05-13
* Added command `kv` to `tools` with sub command `get`, `scan`, `prefix`, `account`, `blk`, `blkirr`, `trx`, `trxtrace` to retrieve data from trxdb 
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
* [Breaking] To improve dfuse instrumented `nodeos` binary processing speed, we had to make incompatible changes to data exchange format going out of `nodeos`. This requires you to upgrade your dfuse instrumented `nodeos` binary to latest version (https://github.com/dfuse-io/eos/releases/tag/v2.0.5-dm-12.0). Follow instructions in at https://github.com/dfuse-io/dfuse-eosio/blob/develop/DEPENDENCIES.md#dfuse-instrumented-eosio-prebuilt-binaries to install the latest version for your platform.
* [Breaking] `--mindreader-working-dir` default value is now `{dfuse-data-dir}/mindreader/work` instead of `{dfuse-data-dir}/mindreader` this is to prevent mindreader from walking files into the working dir and trying to upload and delete nodes system files like `fork_db.dat`
* Added `--eosq-environment` environment where eosq will run (local, dev, production)
* Added `--apiproxy-autocert-domains`, `--apiproxy-autocert-cache-dir` and `--apiproxy-https-listen-addr` to serve SSL directly from proxy.

### Removed

* Removed the `--merger-store-timeout` flag.  Not needed anymore, as some sensible timeouts have been put here and there, using the latest `dstore@v0.1.0` that is context-aware.

### Changed

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
