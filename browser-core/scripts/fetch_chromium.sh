#!/usr/bin/env bash
set -euo pipefail

# Fetch Chromium source using depot_tools
# Usage: fetch_chromium.sh [--rev REVISION]

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
WORKSPACE_ROOT="$SCRIPT_DIR"
SRC_DIR="$WORKSPACE_ROOT/src"

REV="${1:-}"

usage() {
    cat <<EOF
Usage: $0 [--rev REVISION]

Fetch Chromium source using depot_tools.

Options:
  --rev REVISION   Specific git revision to checkout (default: latest stable)

Prerequisites:
  - Xcode Command Line Tools: xcode-select --install
  - Disk space: ~50GB

Example:
  $0 --rev 85f7c2e8
EOF
}

# depot_tools location
DEPOT_TOOLS="$WORKSPACE_ROOT/scripts/depot_tools"

# Install depot_tools if not found
if ! command -v repo &>/dev/null; then
    echo "Installing depot_tools..."
    if [[ ! -d "$DEPOT_TOOLS" ]]; then
        git clone https://chromium.googlesource.com/chromium/tools/depot_tools.git "$DEPOT_TOOLS"
    fi
    export PATH="$DEPOT_TOOLS:$PATH"
fi

mkdir -p "$SRC_DIR"
cd "$SRC_DIR"

# Initialize repo if not already done
if [[ ! -f ".repo" ]]; then
    echo "Initializing repo (this may take a few minutes)..."
    repo init -u https://chromium.googlesource.com/chromium/src.git \
        --depth=1 \
        --no-tags \
        --no-branch-info
fi

# Sync (download ~40GB, may take 1-4 hours depending on network)
echo "Syncing repository (~40GB, may take 1-4 hours)..."
repo sync -c -j$(nproc) --force-sync --no-tags --no-clone-bundle

# Checkout specific revision if specified
if [[ -n "$REV" ]]; then
    echo "Checking out revision: $REV"
    repo forall -c 'git checkout "$REV"' 2>/dev/null || git checkout "$REV"
fi

# Record current state
REVISION=$(git rev-parse HEAD)
MILESTONE=$(git describe --tags 2>/dev/null | cut -d. -f1 || echo "unknown")

echo "$REVISION" > "$WORKSPACE_ROOT/chromium_revision"
echo "chromium_milestone=$MILESTONE" > "$WORKSPACE_ROOT/chromium_milestone"
echo "chromium_revision=$REVISION" >> "$WORKSPACE_ROOT/chromium_milestone"

echo ""
echo "=== Fetch Complete ==="
echo "Source: $SRC_DIR"
echo "Revision: $REVISION"
echo "Milestone: $MILESTONE"
echo ""
echo "Next: ./build.sh"
