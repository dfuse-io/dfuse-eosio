#!/bin/bash

pushd "$(dirname "$0")"
yarn install
yarn build
popd
