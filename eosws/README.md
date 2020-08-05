eosws - EOSIO websocket and REST service
========================================

EOSIO-specific websocket interface, REST interface, Push guarantee
instrumented `/v1/chain/push_transaction` endpoint, and pass-through
to [statedb](../statedb).

## *DEPRECATION NOTICE*

The features herein are scheduled to be migrated to separate REST
service, push-guarantee service, some are to be moved to a better
unified GraphQL interface.  The Websocket interface is to be carried
over to [the GraphQL subscriptions](../dgraphql) eventually.

New things are not to be built on this project.

## Usage

You can view rendered documentation for the REST and Websocket endpoints here:

* Websocket messages: https://docs.dfuse.io/reference/eosio/websocket/
* See _REST API_ under https://docs.dfuse.io/reference/eosio/rest/

## Overview

This service provides REST endpoints for:
* transaction push guarantee
* paginated search
* websocket streaming services
* pass-through to `nodeos` nodes
* pass-through to reach [StateDB](../statedb/) (historical state database)
