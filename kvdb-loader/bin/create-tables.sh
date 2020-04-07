#!/bin/bash
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

set -e

ROOT="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && cd .. && pwd )"

function main() {
    current_dir="`pwd`"
    trap "cd \"$current_dir\"" EXIT
    pushd "$ROOT" &> /dev/null

    if [[ $1 != "" ]]; then
        platform=${1}; shift
    fi

    if [[ $platform != "eos" && $platform != "eth" ]]; then
        echo "Please set a valid platform: 'eos' or 'eth'"
        read -r -p "Enter valid platform: " platform
        echo ""

        # FIXME: Validate entered name again and loop read again if wrong
    fi

    if [[ $1 != "" ]]; then
        table_prefix=${1}; shift
    fi

    if ! echo ${table_prefix} | grep -qE "[a-z0-9]+-v[0-9]+"; then
        echo "Please set a table prefix like 'aca3-v4' or '5fff-v3' or whatever"
        read -r -p "Enter table prefix (gives '${platform}-<prefix>-*'): " table_prefix
        echo ""

        # FIXME: Validate entered name again and loop read again if wrong
    fi

    project=${BT_PROJECT:-"dev"}
    instance=${BT_INSTANCE:-"dev"}
    bt_flags="-bigtable-instance ${instance} -bigtable-project ${project} -create-tables -exit-after-create-tables -table-prefix ${table_prefix}"

    echo "About to create '${platform}-${table_prefix}-*' tables (on ${project}:${instance}) ... 5s delay"
    sleep 1

    go build ./cmd/kvdb-loader

    cli="./kvdb-loader --protocol="$(printf $platform | tr a-z A-Z)" ${bt_flags}"
    if [ -x "$(command -v zap-pretty)" ]; then
        $cli | zap-pretty
    else
        $cli
    fi
}

main $@
