#!/bin/bash

go generate
CGO_ENABLED=1 GOOS=linux go build -o dfusebox-v0.1.1-linux-amd64  ./cmd/dfusebox
CGO_ENABLED=1 GOOS=darwin go build -o dfusebox-v0.1.1-darwin-amd64  ./cmd/dfusebox
rm ./dashboard/rice-box.go
