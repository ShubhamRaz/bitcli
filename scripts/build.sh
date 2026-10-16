#!/usr/bin/env sh
# Build the BitCLI binary for the current platform.
set -eu
mkdir -p bin
go build -ldflags "-s -w" -o bin/bitcli ./cmd/bitcli

