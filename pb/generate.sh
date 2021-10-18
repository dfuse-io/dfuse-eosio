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
  checks

  set -e

  current_dir="`pwd`"
  trap "cd \"$current_dir\"" EXIT
  pushd "$ROOT/pb" &> /dev/null

  # generate "dfuse/eosio/abicodec/v1/abicodec.proto"
  generate "dfuse/eosio/codec/v1/codec.proto"
  # generate "dfuse/eosio/statedb/v1/" "statedb.proto" "tablet.proto" "singlet.proto"
  # generate "dfuse/eosio/trxdb/v1/trxdb.proto"
  # generate "dfuse/eosio/funnel/v1/funnel.proto"
  # generate "dfuse/eosio/search/v1/search.proto"
  # generate "dfuse/eosio/tokenmeta/v1/tokenmeta.proto"
  # generate "dfuse/eosio/accounthist/v1/accounthist.proto"

  echo "generate.sh - `date` - `whoami`" > $ROOT/pb/last_generate.txt
  echo "streamingfast/proto revision: `GIT_DIR=$PROTO/.git git rev-parse HEAD`" >> $ROOT/pb/last_generate.txt
  echo "streamingfast/proto-eosio revision: `GIT_DIR=$PROTO_EOSIO/.git git rev-parse HEAD`" >> $ROOT/pb/last_generate.txt
}

# usage:
# - generate <protoPath>
# - generate <protoBasePath/> [<file.proto> ...]
function generate() {
    base=""
    if [[ "$#" -gt 1 ]]; then
      base="$1"; shift
    fi

    for file in "$@"; do
      protoc -I$PROTO -I$PROTO_EOSIO $base$file --go_out=plugins=grpc,paths=source_relative:.
    done
}

function checks() {
  # The old `protoc-gen-go` did not accept any flags. Just using `protoc-gen-go --version` in this
  # version waits forever. So we pipe some wrong input to make it exit fast. This in the new version
  # which supports `--version` correctly print the version anyway and discard the standard input
  # so it's good with both version.
  result_1_3_5_and_older=`printf "" | protoc-gen-go --version 2>&1 | grep -Eo v[0-9\.]+`
  result_1_4_0_and_later=`printf "" | protoc-gen-go --version 2>&1 | grep -Eo 'unknown argument'`

  if [[ "$result_1_3_5_and_older" != "" || $result_1_4_0_and_later == "unknown argument" ]]; then
    echo "Your version of 'protoc-gen-go' is **too** recent!"
    echo ""
    echo "This repository requires a strict gRPC version not higher than v1.29.1 however"
    echo "the newer protoc-gen-go versions generates code compatible with v1.32 at the minimum."
    echo ""
    echo "To keep the compatibility until the transitive dependency TiKV is updated (through streamingfast/kvdb)"
    echo "you must ue the older package which is hosted at 'github.com/golang/protobuf/protoc-gen-go' (you most"
    echo "probably have 'google.golang.org/protobuf/cmd/protoc-gen-go')."
    echo ""
    echo "To fix your problem, perform those commands:"
    echo ""
    echo "  pushd /tmp"
    echo "    go install github.com/golang/protobuf/protoc-gen-go@v1.3.5"
    echo "  popd"
    echo ""
    echo "If everything is working as expected, the command:"
    echo ""
    echo "  protoc-gen-go --version"
    echo ""
    echo "Should hang indefinitely (as it expects standard input to come)"
    exit 1
  fi
}

main "$@"
