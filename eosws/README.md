DEPRECATION NOTICE
==================

The eosws project will be shutdown after its features are migrated to different projects (dgraphql, etc.).
Do not expect features to be added and a high level of maintenance.


EOSIO websocket streaming service
=================================

This service provides REST endpoints for:
* transaction push guarantee
* websocket streaming services
* pass-through to reach FluxDB (historical state database). See https://github.com/dfuse-io/dfuse-eosio/fluxdb
* serves a few internal / undocumented services for `eosq` too
