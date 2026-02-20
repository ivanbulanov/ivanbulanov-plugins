#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CLI_DIR="$(cd "$SCRIPT_DIR/../cli" && pwd)"
BIN_PATH="${CLI_DIR}/bin/atlassian-cloud"

# --- Ensure binary is built ---

if [[ ! -x "$BIN_PATH" ]]; then
    echo "Building atlassian-cloud CLI..." >&2

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
    echo "CLI built successfully." >&2
fi

# --- Check auth status ---

echo "CLI: $BIN_PATH"
"$BIN_PATH" auth status 2>&1 || true
