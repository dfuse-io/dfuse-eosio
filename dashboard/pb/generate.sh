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


ROOT="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

current_dir="`pwd`"
trap "cd \"$current_dir\"" EXIT
pushd "$ROOT" &> /dev/null

# generating .go proto files in the same folder as the .proto definition file
protoc -I . dashboard.proto --go_out=plugins=grpc:.

# generating .ts proto files in the same folder as the .proto definition file
protoc --plugin="protoc-gen-ts=../client/node_modules/.bin/protoc-gen-ts" --js_out="import_style=commonjs,binary:../client/src/pb" --ts_out="service=grpc-web:../client/src/pb" ./dashboard.proto

# add /* eslint-disable */ atop all .js and .ts files generated above
grep -q '^/\* eslint-disable \*/$' ../client/src/pb/dashboard_pb.d.ts || echo '/* eslint-disable */' > temp && cat ../client/src/pb/dashboard_pb.d.ts >> temp && mv -f temp ../client/src/pb/dashboard_pb.d.ts
grep -q '^/\* eslint-disable \*/$' ../client/src/pb/dashboard_pb.js || echo '/* eslint-disable */' > temp && cat ../client/src/pb/dashboard_pb.js >> temp && mv -f temp ../client/src/pb/dashboard_pb.js
grep -q '^/\* eslint-disable \*/$' ../client/src/pb/dashboard_pb_service.d.ts || echo '/* eslint-disable */' > temp && cat ../client/src/pb/dashboard_pb_service.d.ts >> temp && mv -f temp ../client/src/pb/dashboard_pb_service.d.ts
grep -q '^/\* eslint-disable \*/$' ../client/src/pb/dashboard_pb_service.js || echo '/* eslint-disable */' > temp && cat ../client/src/pb/dashboard_pb_service.js >> temp && mv -f temp ../client/src/pb/dashboard_pb_service.js
