#!/usr/bin/env bash

ROOT="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

function usage {
    echo "./reproc_shard_dev.sh <startBlock> <stopBlock> [fluxdbExtraArgs...]"
    exit 1
}

function main {
    if [[ $1 == "" || $1 == "-h" || $1 == "--help" || $2 == "" ]]; then
        usage
    fi

    # Ensure we are in $ROOT directory while executing
    pushd "$ROOT" &> /dev/null

    echo "Building 'fluxdb' CLI..."
    set -e
    go build ./cmd/fluxdb

    startBlock="$1"; shift
    stopBlock="$1"; shift

    echo "About to start sharding"
    ./fluxdb reproc shard --start-block="$startBlock" --stop-block="$stopBlock" --shard-count 4 $@
}

main $@

