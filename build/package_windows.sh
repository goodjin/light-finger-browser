#!/usr/bin/env bash
set -euo pipefail

# Sign Windows installer (NSIS output from Wails)
# Usage: package_windows.sh <installer-path> [output-dir]

INSTALLER_PATH="${1:-}"
OUTPUT_DIR="${2:-./dist}"

[[ -z "$INSTALLER_PATH" || ! -f "$INSTALLER_PATH" ]] && { echo "Usage: $0 <installer-path> [output-dir]"; exit 1; }

SIGNED_PATH="$OUTPUT_DIR/signed_$(basename "$INSTALLER_PATH")"

if command -v osslsigncode >/dev/null 2>&1; then
  echo "Signing Windows installer..."
  osslsigncode sign \
    -t http://timestamp.digicert.com \
    -sha256 \
    -n "Fingerbrower" \
    -i "https://fingerbrower.example.com" \
    -in "$INSTALLER_PATH" \
    -out "$SIGNED_PATH"
  echo "Signed installer: $SIGNED_PATH"
else
  echo "osslsigncode not found. Install: brew install osslsigncode"
  echo "Skipping signing. Original installer: $INSTALLER_PATH"
fi
