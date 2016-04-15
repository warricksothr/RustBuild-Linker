#!/usr/bin/env bash

set -x
set -e

GOPATH="$PWD:$GOPATH"

go clean
go fmt
go get || true
go build -o "rbl_$(date +%Y-%m-%d)_$(git rev-parse --short HEAD)" -p $(($(nproc)-1)) -ldflags "-s" rustbuild-linker.go

mkdir -p builds
mv rbl_* builds
LATEST_BUILD=$(ls -t builds/rbl_* | head -1)
ln -sf LATEST_BUILD latest_build
