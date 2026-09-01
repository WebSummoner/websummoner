#!/bin/bash

set -e

export GO111MODULE="on"
LDFLAGS="-X main.buildStamp=$(date -u '+%Y-%m-%d_%I:%M:%S%p') -X main.gitRevision=$(git describe --tags || git rev-parse HEAD) -s -w"

build() {
    local goos=$1 goarch=$2 ext=""
    if [ "$goos" = "windows" ]; then
        ext=".exe"
    fi
    GOOS=$goos GOARCH=$goarch CGO_ENABLED=0 go build -ldflags "$LDFLAGS" -o "dist/websummoner_${goos}_${goarch}${ext}" .
}

build linux amd64
build darwin amd64
build darwin arm64
build windows amd64
build windows arm64
build windows 386
build linux arm64
