#!/bin/bash

ROOT="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

current_dir="`pwd`"
trap "cd \"$current_dir\"" EXIT
pushd "$ROOT" &> /dev/null

protoc -I. -I$ROOT/../../../../service-definitions kvrows.proto --go_out=plugins=grpc:.
