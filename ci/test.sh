#!/bin/bash

set -e

export GO111MODULE="on"
go test -tags 's3 metadata' -v -race -coverprofile=coverage.out -covermode=atomic -coverpkg github.com/websummoner/websummoner,github.com/websummoner/websummoner/session,github.com/websummoner/websummoner/config,github.com/websummoner/websummoner/protect,github.com/websummoner/websummoner/service,github.com/websummoner/websummoner/upload,github.com/websummoner/websummoner/info,github.com/websummoner/websummoner/jsonerror
cp coverage.out coverage.txt

ci/govulncheck.sh
