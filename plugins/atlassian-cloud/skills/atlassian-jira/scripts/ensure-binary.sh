#!/usr/bin/env bash
set -euo pipefail

CLI_DIR="${CLAUDE_PLUGIN_ROOT}/cli"
BIN_PATH="${CLI_DIR}/bin/atlassian-cloud"

# Check if binary exists and is executable
if [[ -x "$BIN_PATH" ]]; then
    exit 0
fi

echo "Building atlassian-cloud CLI..." >&2

# Ensure Go is available
if ! command -v go &>/dev/null; then
    if command -v mise &>/dev/null; then
        echo "Installing Go via mise..." >&2
        mise install go@latest
        eval "$(mise env)"
    else
        echo "ERROR: Go is not installed. Install Go or mise first." >&2
        echo "  Option 1: Install mise (https://mise.jdx.dev)" >&2
        echo "  Option 2: Install Go directly (https://go.dev/dl/)" >&2
        exit 1
    fi
fi

mkdir -p "${CLI_DIR}/bin"
cd "$CLI_DIR"
go build -o bin/atlassian-cloud .

echo "atlassian-cloud CLI built successfully" >&2
