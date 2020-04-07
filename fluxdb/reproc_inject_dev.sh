#!/usr/bin/env bash

ROOT="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

function usage {
    echo "./reproc_inject_dev.sh [fluxdbExtraArgs...]"
    exit 1
}

function teardown() {
  for job in `jobs -p`; do
    kill -s TERM $job &> /dev/null || true
  done
}

function main {
    if [[ $1 == "-h" || $1 == "--help" ]]; then
        usage
    fi

    # Ensures we are using the local dev environment
    $(gcloud beta emulators bigtable env-init)

    IFS=':' read -r -a bt_emulator_host_port <<< "$BIGTABLE_EMULATOR_HOST"

    bt_emulator_host=${bt_emulator_host_port[0]}
    bt_emulator_port=${bt_emulator_host_port[1]}

    check_open="nc -z -w3 "$bt_emulator_host" "$bt_emulator_port""
    _=`$check_open`
    if [[ $? != 0 ]]; then
        echo "GCloud BigTable emulator is not listening, start it with 'gcloud beta emulators bigtable start'"
        echo "Command '$check_open' failed"
        exit 1
    fi

    # Ensure we are in $ROOT directory while executing
    pushd "$ROOT" &> /dev/null

    echo "Building 'fluxdb' CLI..."
    set -e
    go build ./cmd/fluxdb

    trap teardown EXIT

    echo "About to start indexing (in BigTable Emulator)..."
    storeDsnArg="--store-dsn=bigtable://dev.dev/test-v0?createTables=true"

    ./fluxdb reproc inject $storeDsnArg --bigtable-create=true --shard-index="0" --shard-count 4 $@ &
    ./fluxdb reproc inject $storeDsnArg --bigtable-create=false --shard-index="1" --shard-count 4 $@ &
    ./fluxdb reproc inject $storeDsnArg --bigtable-create=false --shard-index="2" --shard-count 4 $@ &
    ./fluxdb reproc inject $storeDsnArg --bigtable-create=false --shard-index="3" --shard-count 4 $@ &

    echo ""
    echo "Waiting for all shard to finish injection, press Ctrl+C to terminate now"
    for job in `jobs -p`; do
        wait $job || true
    done
}

main $@
