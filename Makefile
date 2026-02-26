# SPDX-FileCopyrightText: 2020 Ethel Morgan
#
# SPDX-License-Identifier: MIT

VERSION := $(shell git describe --tags --abbrev=0)
COMMIT := $(shell git rev-parse --short=7 HEAD)
DATE := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

all: build

build:
	docker build -t helix-build . -f Dockerfile.build
	docker run --rm \
		-v "$(CURDIR):/app" \
		-e "VERSION=${VERSION}" \
		-e "COMMIT=${COMMIT}" \
		-e "DATE=${DATE}" \
		helix-build

clean:
	rm -rf bin

.PHONY: all build clean
