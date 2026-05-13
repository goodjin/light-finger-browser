#!/usr/bin/env bash
set -euo pipefail

# Color output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

usage() {
  cat <<EOF
Usage: $0 <platform> [-o OUTPUT_DIR]

Platforms: darwin-arm64, darwin-amd64, windows-amd64, linux-amd64, all

Options:
  -o    Output directory (default: ./dist)

Examples:
  $0 darwin-arm64 -o ./dist
  $0 all -o ./release
EOF
}

warn() { echo -e "${YELLOW}WARNING: $1${NC}" >&2; }
die() { echo -e "${RED}ERROR: $1${NC}" >&2; exit 1; }
info() { echo -e "${GREEN}[INFO] $1${NC}"; }

# Parse arguments
PLATFORM=""
OUTPUT_DIR="./dist"

while [[ $# -gt 0 ]]; do
  case "$1" in
    -o) OUTPUT_DIR="$2"; shift 2 ;;
    -h|--help) usage; exit 0 ;;
    darwin-arm64|darwin-amd64|windows-amd64|linux-amd64|all) PLATFORM="$1"; shift ;;
    *) die "Unknown platform: $1" ;;
  esac
done

[[ -z "$PLATFORM" ]] && { usage; exit 1; }

# Check tools
command -v wails >/dev/null 2>&1 || die "wails not found. Install: go install github.com/wailsapp/wails/v2/cmd/wails@latest"
command -v go >/dev/null 2>&1 || die "go not found"

info "Building fingerbrower for $PLATFORM..."
info "Output: $OUTPUT_DIR"

mkdir -p "$OUTPUT_DIR"

if [[ "$PLATFORM" == "all" ]]; then
  for p in darwin-arm64 darwin-amd64 windows-amd64 linux-amd64; do
    $0 "$p" -o "$OUTPUT_DIR"
  done
else
  wails build -platform "$PLATFORM" -o "$OUTPUT_DIR" || die "Build failed"
  info "Build complete: $OUTPUT_DIR"

  # Copy Chromium.app for darwin builds
  if [[ "$PLATFORM" == darwin-* ]]; then
    SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    POST_BUILD="$SCRIPT_DIR/../scripts/post-build.sh"
    if [[ -x "$POST_BUILD" ]]; then
      "$POST_BUILD"
    fi
  fi
fi
