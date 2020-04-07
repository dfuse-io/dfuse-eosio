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

function main() {
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

    read -r -p "Are you sure you want to delete '${platform}-${table_prefix}-*' tables (on ${project}:${instance}), this is irreversible? [y/N] " response
    if [[ "$response" =~ ^([yY][eE][sS]|[yY])+$ ]]; then
        bt_flags="-project ${project} -instance ${instance}"

        if [[ $platform == "eos" ]]; then
            delete_eos_tables "$bt_flags" "$table_prefix"
        elif [[ $platform == "eth" ]]; then
            delete_eth_tables "$bt_flags" "$table_prefix"
        fi
    else
        echo "Aborting deletion of tables"
        exit 1
    fi
}

function delete_eos_tables() {
    bt_flags=$1
    table_prefix$2

    echo "Deleting tables 'eos-${table_prefix}-*' ..."
    set -x
    cbt $bt_flags deletetable eos-${table_prefix}-accounts
    cbt $bt_flags deletetable eos-${table_prefix}-blocks
    cbt $bt_flags deletetable eos-${table_prefix}-timeline
    cbt $bt_flags deletetable eos-${table_prefix}-trxs
    set +x
}

function delete_eth_tables() {
    bt_flags="$1"
    table_prefix="$2"

    echo "Deleting tables 'eth-${table_prefix}-*' ..."
    set -x
    cbt $bt_flags deletetable eth-${table_prefix}-blocks
    cbt $bt_flags deletetable eth-${table_prefix}-trxs
    cbt $bt_flags deletetable eth-${table_prefix}-timeline
    set +x
}

main $@
