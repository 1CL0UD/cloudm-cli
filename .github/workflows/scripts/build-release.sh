#!/usr/bin/env bash
VERSION=${1:-$(git describe --tags --always)}
BUILD_DATE=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
GIT_COMMIT=$(git rev-parse --short HEAD)

LDFLAGS="-X main.Version=${VERSION} -X main.BuildDate=${BUILD_DATE} -X main.GitCommit=${GIT_COMMIT}"

mkdir -p dist

GOOS=linux GOARCH=amd64 go build -ldflags "$LDFLAGS" -o dist/cloudm-cli-linux-amd64 main.go
GOOS=linux GOARCH=arm64 go build -ldflags "$LDFLAGS" -o dist/cloudm-cli-linux-arm64 main.go
GOOS=darwin GOARCH=amd64 go build -ldflags "$LDFLAGS" -o dist/cloudm-cli-darwin-amd64 main.go
GOOS=darwin GOARCH=arm64 go build -ldflags "$LDFLAGS" -o dist/cloudm-cli-darwin-arm64 main.go

echo "Binaries built in dist/"
ls -lh dist/