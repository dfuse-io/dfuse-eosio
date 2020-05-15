# dfuse Connector to SQL

This application is used to synchronize EOSIO tables to a SQL database.

Features:

* Bootstrap from any block height (from FluxDB, and eventually from a portable state snapshot).
* Sync modes:
    * Only irreversible state
    * Real-time, handling reorgnaizations/micro-forks.
* Consistent SQL querying, given each block's changes is committed as a single SQL transaction.
* Mapping of on-chain fields to different fields in SQL
    * Denormalization of nested structs on-chain, into their own fields in SQL


## Usage

Simplest way to start is:

    dfuseeos start sqlsync

But you will want to connect it somewhere:

    dfuseeos --config-file= --skip-checks \
             --common-blockmeta-addr=localhost:9000 \
             --common-blockstream-addr=localhost:9001 \
             --common-blocks-store-url=gs://dfuseio-global-blocks-us/wax-mainnet/v3  \
             --sqlsync-fluxdb-addr=http://localhost:9002  \
             --sqlsync-sql-dsn postgres://postgres:secret@localhost:5432/postgres  \
             start sqlsync -vvv

with port-forwarding for `blockmeta` on port 9000, and a `relayer` or
`mindreader` on port 9001, in addition to port-forward to `fluxdb`
listening on 9002 (to its http port).  Try something like:

    kc port-forward svc/blockmeta-v2 9000 & kc port-forward svc/relayer-v2 9001:9000 & kc port-forward svc/fluxdb-server-v2 9002:80 &


### Starting postgres

Start a local postgres:

    docker run --rm -ti --name psql -p 5432:5432 -e POSTGRES_PASSWORD=secret postgres

Go inside with:

    docker exec -ti -e POSTGRES_PASSWORD=secret psql psql -U postgres


### Deploy Hasura for GraphQL querying

You can deploy Hasura on K8s using:

    https://github.com/hasura/graphql-engine/blob/stable/install-manifests/kubernetes/deployment.yaml
