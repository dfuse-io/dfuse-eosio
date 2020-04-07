#!/usr/bin/env bash
# Copyright 2019 dfuse Platform Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.


ROOT="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

# Ensures we are using the local dev environment
$(gcloud beta emulators bigtable env-init)

IFS=':' read -r -a bt_emulator_host_port <<< "$BIGTABLE_EMULATOR_HOST"

bt_emulator_host=${bt_emulator_host_port[0]}
bt_emulator_port=${bt_emulator_host_port[1]}

check_open="nc -z -w3 "$bt_emulator_host" "$bt_emulator_port""
_=`$check_open`
if [[ $? != 0 ]]; then
    echo "GCloud BigTable emulator is not listening, start it in another terminal with 'gcloud beta emulators bigtable start'"
    echo "Command '$check_open' failed"
    exit 1
fi

# Ensure we are in $ROOT directory while executing
pushd "$ROOT"

# Trap exit signal and pop directory
trap popd EXIT

echo "Building 'eosdb-loader' CLI..."
set -e
go build ./cmd/kvdb-loader

blocks_store_url=gs://example/blocks

echo "About to start indexing (in BigTable Emulator)..."
./kvdb-loader \
    -protocol=EOS \
    -processing-type=live \
    -block-version=2 \
    -chain-id=68c4335171ad518f7ebf8930b8f1740ed9d2638e4a6898a18472f4e360994a8f \
    -source-store=${blocks_store_url} \
    -batch-size=100 \
    -bigtable-instance=dev \
    -bigtable-project=dev \
    -table-prefix=test-v1 \
    -create-tables=true \
    -allow-live-on-empty-table=true \
    ${@}
