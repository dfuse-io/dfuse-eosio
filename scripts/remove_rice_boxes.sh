#!/usr/bin/env bash

ROOT="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && cd .. && pwd )"

rm $ROOT/dashboard/rice-box.go &> /dev/null || true
rm $ROOT/eosq/app/eosq/rice-box.go &> /dev/null || true
