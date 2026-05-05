#!/usr/bin/env bash
set -euo pipefail

# Create DMG from app bundle
# Usage: package_dmg.sh <app-path> [output-dmg]

APP_PATH="${1:-}"
OUTPUT_DMG="${2:-./dist/fingerbrower.dmg}"

[[ -z "$APP_PATH" || ! -d "$APP_PATH" ]] && { echo "Usage: $0 <app-path> [output-dmg]"; exit 1; }

DMG_DIR=$(mktemp -d)
VOL_NAME="Fingerbrower"

# Create Applications symlink
ln -sf /Applications "$DMG_DIR/Applications"

# Copy app bundle
cp -R "$APP_PATH" "$DMG_DIR/"

# Create DMG
hdiutil create -volname "$VOL_NAME" \
  -srcfolder "$DMG_DIR" \
  -ov -format UDZO \
  -imagekey zlib-level=9 \
  "$OUTPUT_DMG"

rm -rf "$DMG_DIR"
echo "DMG created: $OUTPUT_DMG"
