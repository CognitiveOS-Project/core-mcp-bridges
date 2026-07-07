#!/bin/bash
set -euo pipefail

if ! command -v go &>/dev/null; then
    if [ ! -f /tmp/go/bin/go ]; then
        echo "Installing Go..."
        curl -sL https://go.dev/dl/go1.24.linux-amd64.tar.gz | tar -C /tmp -xz
    fi
    export PATH="/tmp/go/bin:$PATH"
fi

BUILD_DIR="$(cd "$(dirname "$0")/.." && pwd)/build"
BIN_DIR="${BUILD_DIR}/bin"
mkdir -p "${BIN_DIR}"

cd "$(dirname "$0")/.."

for dir in */; do
    bridge=$(basename "${dir}")
    [ "$bridge" = "internal" ] || [ "$bridge" = "build" ] && continue
    if [ -f "${dir}main.go" ] || [ -f "${dir}go.mod" ]; then
        echo "Building bridge: ${bridge}..."
        CGO_ENABLED=0 go build -ldflags="-s -w" -o "${BIN_DIR}/${bridge}" "./${dir}"
        echo "  -> ${BIN_DIR}/${bridge}"
    fi
done
