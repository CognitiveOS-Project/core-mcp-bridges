SHELL := /bin/sh
.SHELLFLAGS := -eu -c
.ONESHELL:
.DELETE_ON_ERROR:

BUILD_DIR := build
BIN_DIR := $(BUILD_DIR)/bin
GO := go

.PHONY: build test lint clean pack

build:
	@mkdir -p $(BIN_DIR)/bridges
	@for dir in */; do
		bridge=$$(basename "$$dir")
		[ "$$bridge" = "internal" ] || [ "$$bridge" = "build" ] && continue
		if [ -f "$${dir}main.go" ] || [ -f "$${dir}go.mod" ]; then
			echo "  building bridge: $$bridge"
			CGO_ENABLED=0 $(GO) build -ldflags="-s -w" -o $(BIN_DIR)/bridges/$$bridge "./$$dir"
		fi
	done

pack: build
	@VERSION=$$(git describe --tags --abbrev=0 2>/dev/null || echo "dev")
	@CPM=/workspace/cpm/build/bin/cpm
	@$${CPM} pack --bin $(BIN_DIR)/bridges --name core-mcp-bridges --version $$VERSION --os linux --arch amd64 --description "Core hardware MCP bridges"

test:
	$(GO) test ./... -v -count=1

lint:
	shellcheck scripts/build.sh
	$(GO) vet ./...

clean:
	rm -rf $(BUILD_DIR)
