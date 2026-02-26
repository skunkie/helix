#!/bin/sh
#
# SPDX-FileCopyrightText: 2020 Ethel Morgan
#
# SPDX-License-Identifier: MIT

set -e

mkdir -p bin

# These variables are passed in from the Makefile via `docker run -e`
LDFLAGS="-s -w -X main.version=${VERSION} -X main.commit=${COMMIT} -X main.date=${DATE}"

build() {
    echo "Building for $1/$2..."
    GOOS=$1 GOARCH=$2 go build -buildvcs=false -ldflags "${LDFLAGS}" -trimpath -o "bin/helix-$1-$2$3" ./cmd/helix-directory
}

build_arm() {
    echo "Building for linux/armv$1..."
    GOOS=linux GOARCH=arm GOARM=$1 go build -buildvcs=false -ldflags "${LDFLAGS}" -trimpath -o "bin/helix-linux-arm-v$1" ./cmd/helix-directory
}

build linux amd64
build_arm 6
build_arm 7
build linux arm64
build darwin amd64
build darwin arm64
build windows amd64 .exe
build windows arm64 .exe
