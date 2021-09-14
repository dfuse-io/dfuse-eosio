# Changelog

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased] - develop

### Deprecated
- This whole project is marked as deprecated, we plan to migrate its features, mostly to dgraphql

### Added
- WARN: Configurability of the `--dipp-secret` and `--healthz-secret` in the `eosws` binary. These should be changed now that the code goes into the open.

### Changed
- Startup now based on new blockmeta features. It does not fetch startblock from EOS API nor depend on the kvdb to be ready to start processing blocks.
- Flag `--filesource-ratelimit-ms=2` should be **replaced** by `--filesource-ratelimit=2ms`
- Removed `--bigtable-db`, replaced by a kvdb DSN string as `--kvdb-dsn`.  See https://github.com/streamingfast/kvdb for details.
- Removed `--redis-address`, `--billing-pubsub-project`, `--with-cut-off`, `--backlist-update-interval`, `--jwt-auth-kms-keypath`, which are all *replaced by*  `--dauth-plugin` and `--dmetering-plugin` flags.  See https://github.com/streamingfast/dauth and https://github.com/streamingfast/dmetering for more config string examples.
- Response from `/v0/simple_search` for a matching `block` used to return a huge payload (straight up protobuf in json), but now only the `id` is returned. It is the only field that was consumed by `eosq`, the rest was undocumented and weird.

### Removed
- Removed `--db-type`, use `--kvdb-dsn` now.
- Removed deprecated `/v1/healthz` endpoint handler (`/healthz` is the way to go now).
- Removed `www/healthz.html`.
    - **Migrate**: Update builds/deploys to avoid its copy now.
- Removed `--debug` flag (was unused)
