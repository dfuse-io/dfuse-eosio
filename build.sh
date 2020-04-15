#!/bin/bash

go generate
CGO_ENABLED=1 GOOS=linux go build -o dfuseeos-v0.1.1-linux-amd64  ./cmd/dfuseeos
CGO_ENABLED=1 GOOS=darwin go build -o dfuseeos-v0.1.1-darwin-amd64  ./cmd/dfuseeos
rm ./dashboard/rice-box.go
