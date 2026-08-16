#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CLI_DIR="$(cd "$SCRIPT_DIR/../cli" && pwd)"
BIN_PATH="${CLI_DIR}/bin/atlassian-cloud"
PLUGIN_JSON="${SCRIPT_DIR}/../.claude-plugin/plugin.json"
STAMP_FILE="${CLI_DIR}/bin/.build-version"

# Plugin version drives rebuilds: the version stamped into bin/.build-version at
# build time is compared against the current plugin.json version. A plugin
# update therefore forces a fresh binary even when a stale one is present and its
# mtime happens to look newer than the (timestamp-preserved) extracted source.
PLUGIN_VERSION=""
if [[ -f "$PLUGIN_JSON" ]]; then
    PLUGIN_VERSION="$(sed -n 's/.*"version"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' "$PLUGIN_JSON" | head -n1)"
fi

# --- Ensure binary is built and up to date ---

needs_build=0
if [[ ! -x "$BIN_PATH" ]]; then
    needs_build=1
elif [[ -n "$PLUGIN_VERSION" && "$(cat "$STAMP_FILE" 2>/dev/null)" != "$PLUGIN_VERSION" ]]; then
    echo "Plugin version changed (now ${PLUGIN_VERSION}); rebuilding..." >&2
    needs_build=1
elif [[ -n "$(find "$CLI_DIR" -type f \( -name '*.go' -o -name 'go.mod' -o -name 'go.sum' \) -newer "$BIN_PATH" -print -quit 2>/dev/null)" ]]; then
    echo "Source changed since last build; rebuilding..." >&2
    needs_build=1
fi

if [[ "$needs_build" -eq 1 ]]; then
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

    # Record the version we just built so the next run rebuilds on a version bump.
    if [[ -n "$PLUGIN_VERSION" ]]; then
        echo "$PLUGIN_VERSION" > "$STAMP_FILE"
    fi
fi

# --- Check auth status ---

echo "CLI: $BIN_PATH"
"$BIN_PATH" auth status 2>&1 || true

# --- Report the Mermaid renderer ---

# Deliberately non-fatal. Only `confluence publish` needs mmdc, and only for
# documents that contain Mermaid fences, so a missing binary must not block
# Jira or read-only Confluence work. Reporting it here moves discovery to the
# front of the workflow instead of leaving it to fail mid-publish. Installing
# it is the user's decision: this script never fetches it.
MMDC_BIN="${MMDC:-mmdc}"
if MMDC_RESOLVED="$(command -v "$MMDC_BIN" 2>/dev/null)"; then
    echo "mmdc: $MMDC_RESOLVED"
else
    echo "mmdc: not found (needed only by 'confluence publish' for Mermaid diagrams;" \
         "install with 'npm install -g @mermaid-js/mermaid-cli')"
fi
