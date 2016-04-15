#!/usr/bin/env bash

#set -x
set -e

GOPATH="$PWD:$GOPATH"

echo "Cleaning project"
go clean
echo "Enforcing Code Formatting"
go fmt
echo "Updating libraries"
go get || true
echo "Building RustBuild-Linker"
go build -o "rbl_$(date +%Y-%m-%d)_$(git rev-parse --short HEAD)" -p $(($(nproc)-1)) -ldflags "-s" rustbuild-linker.go

echo "Moving binary to target directory [./builds]"
mkdir -p builds
mv rbl_* builds
LATEST_BUILD=$(ls -t builds/rbl_* | head -1)
LATEST_BUILD_NAME=$(basename $LATEST_BUILD)
echo "Linking latest build"
ln -sf $LATEST_BUILD latest_build
echo "Build [$LATEST_BUILD_NAME] Complete!"
