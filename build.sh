#!/usr/bin/env bash

set -x
set -e

go clean
go fmt
go get || true
go build -o "rbl_$(git rev-parse --short HEAD)_$(date +%Y-%m-%d)" -p $(($(nproc)-1)) -ldflags "-s" rustbuild-linker.go
