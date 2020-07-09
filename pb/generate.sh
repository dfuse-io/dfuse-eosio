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

ROOT="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && cd .. && pwd )"

# Protobuf definitions
PROTO=${1:-"$ROOT/../proto"}
PROTO_EOSIO=${2:-"$ROOT/../proto-eosio"}

function main() {
  set -e

  current_dir="`pwd`"
  trap "cd \"$current_dir\"" EXIT
  pushd "$ROOT/pb" &> /dev/null

  generate "dfuse/eosio/abicodec/v1/abicodec.proto"
  generate "dfuse/eosio/codec/v1/codec.proto"
  generate "dfuse/eosio/fluxdb/v1/fluxdb.proto"
  generate "dfuse/eosio/trxdb/v1/trxdb.proto"
  generate "dfuse/eosio/funnel/v1/funnel.proto"
  generate "dfuse/eosio/search/v1/search.proto"
  generate "dfuse/eosio/fluxdb/v1/fluxdb.proto"

  echo "generate.sh - `date` - `whoami`" > $ROOT/pb/last_generate.txt
  echo "dfuse-io/proto revision: `GIT_DIR=$PROTO/.git git rev-parse HEAD`" >> $ROOT/pb/last_generate.txt
  echo "dfuse-io/proto-eosio revision: `GIT_DIR=$PROTO_EOSIO/.git git rev-parse HEAD`" >> $ROOT/pb/last_generate.txt
}

function generate() {
    protoc -I$PROTO -I$PROTO_EOSIO $1 --go_out=plugins=grpc,paths=source_relative:.
}

main "$@"
