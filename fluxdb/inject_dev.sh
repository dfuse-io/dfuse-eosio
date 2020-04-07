#!/usr/bin/env bash

ROOT="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

function usage {
    echo "./inject_dev.sh [fluxdbExtraArgs...]"
    exit 1
}

function main {
    if [[ $1 == "-h" || $1 == "--help" ]]; then
        usage
    fi

#     # Ensures we are using the local dev environment
#     $(gcloud beta emulators bigtable env-init)
#
#     IFS=':' read -r -a bt_emulator_host_port <<< "$BIGTABLE_EMULATOR_HOST"
#
#     bt_emulator_host=${bt_emulator_host_port[0]}
#     bt_emulator_port=${bt_emulator_host_port[1]}
#
#     check_open="nc -z -w3 "$bt_emulator_host" "$bt_emulator_port""
#     _=`$check_open`
#     if [[ $? != 0 ]]; then
#         echo "GCloud BigTable emulator is not listening, start it with 'gcloud beta emulators bigtable start'"
#         echo "Command '$check_open' failed"
#         exit 1
#     fi

    # Ensure we are in $ROOT directory while executing
    pushd "$ROOT" &> /dev/null

    echo "Building 'fluxdb' CLI..."
    set -e
    go build ./cmd/fluxdb

#   storeDsnArg="--store-dsn=bigtable://dev.dev/test-v0?createTables=true"
#    storeDsnArg="--store-dsn=bbolt://fluxdb.bbolt?createTables=true"
    storeDsnArg="--store-dsn=badger:///Users/julien/codebase/dfuse/fluxdb/badger.db"

    cmd="./fluxdb inject $storeDsnArg $@"

    echo "About to start indexing using '$cmd'"
    $cmd

}

main $@
