#!/bin/bash

pushd "$(dirname "$0")"
yarn build
yarn install
popd
