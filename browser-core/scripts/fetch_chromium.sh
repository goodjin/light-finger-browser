#!/usr/bin/env bash
set -euo pipefail

# Fetch Chromium source using depot_tools
# Usage: fetch_chromium.sh [--rev REVISION]

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
WORKSPACE_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
SRC_DIR="$WORKSPACE_ROOT/src"

REV="${1:-}"
EXTRA_DEPS=(
    "src/build"
    "src/buildtools"
    "src/third_party/depot_tools"
)

usage() {
    cat <<EOF
Usage: $0 [--rev REVISION]

Fetch Chromium source using depot_tools.

Options:
  --rev REVISION   Specific git revision to checkout (default: latest stable)

Example:
  $0 --rev 85f7c2e8
EOF
}

# Check for depot_tools
if ! command -v fetch &>/dev/null; then
    echo "depot_tools not found. Installing..."
    if [[ ! -d "$WORKSPACE_ROOT/depot_tools" ]]; then
        git clone https://chromium.googlesource.com/chromium/tools/depot_tools.git "$WORKSPACE_ROOT/depot_tools"
    fi
    export PATH="$WORKSPACE_ROOT/depot_tools:$PATH"
fi

mkdir -p "$SRC_DIR"
cd "$SRC_DIR"

# Initialize repo if not already done
if [[ ! -f ".repo" ]]; then
    repo init -u https://chromium.googlesource.com/chromium/src.git \
        --depth 1 \
        --platform=all \
        "${EXTRA_DEPS[@]}"
fi

# Sync
repo sync -c -j$(nproc) --force-sync

# Checkout specific revision if specified
if [[ -n "$REV" ]]; then
    git checkout "$REV"
fi

# Record current revision
git rev-parse HEAD > "$WORKSPACE_ROOT/chromium_revision"
echo "chromium_milestone=$(git describe --tags | cut -d. -f1)" > "$WORKSPACE_ROOT/chromium_milestone"

echo "Chromium source fetched to $SRC_DIR"
echo "Revision: $(git rev-parse HEAD)"
