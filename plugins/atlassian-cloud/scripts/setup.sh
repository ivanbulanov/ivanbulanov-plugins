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

# Printed when the build cannot proceed. A plugin cannot grant itself sandbox
# exemptions — sandbox settings are user-side only — so the most useful thing
# this script can do is name the exact entry to add, with the real path already
# filled in. allowWrite is not offered as the first answer: writes under
# ~/.claude are protected and an allowWrite entry does not lift that, and the
# build needs the Go caches and the module proxy as well.
sandbox_help() {
    cat >&2 <<HELP

ERROR: $1

If Claude Code's Bash sandbox is enabled, this is expected: building writes
the binary into the plugin directory, writes the Go build and module caches,
and fetches modules over the network. The plugin cannot exempt itself.

Add this to ~/.claude/settings.json and start a new session:

  "sandbox": {
    "excludedCommands": ["${SCRIPT_DIR}/setup.sh*"]
  }

Only this build script then runs outside the sandbox; the CLI itself stays
inside it. The CLI still needs its own hosts allowed:

  "sandbox": {
    "network": { "allowedDomains": ["*.atlassian.net", "api.atlassian.com", "api.media.atlassian.com"] }
  }

If the sandbox is off, this is a plain permissions problem: check that you can
write to the path above.
HELP
}

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

    if ! mkdir -p "${CLI_DIR}/bin" 2>/dev/null || ! touch "${CLI_DIR}/bin/.writable" 2>/dev/null; then
        sandbox_help "cannot write to ${CLI_DIR}/bin"
        exit 1
    fi
    rm -f "${CLI_DIR}/bin/.writable"

    cd "$CLI_DIR"
    if ! go build -o bin/atlassian-cloud .; then
        # go build also writes the module and build caches, which live outside
        # the plugin, and fetches modules over the network. Any of the three
        # can be what the sandbox refused, so say so rather than leaving the
        # user with a bare Go error.
        sandbox_help "go build failed"
        exit 1
    fi
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
