#!/usr/bin/env bash
set -euo pipefail

# Sign and notarize a macOS app bundle
# Usage: sign_darwin.sh <app-path> <apple-id> <app-password> <team-id>

usage() {
  cat <<EOF
Usage: $0 <app-path> <apple-id> <app-password> <team-id>

Example:
  $0 ./dist/fingerbrower.app "developer@example.com" "xxxx-xxxx-xxxx-xxxx" "A1B2C3D4E5"

Environment variables (alternative to args):
  FINGERBROWSER_APP_PATH
  APPLE_ID
  APP_PASSWORD
  TEAM_ID
EOF
}

APP_PATH="${FINGERBROWSER_APP_PATH:-${1:-}}"
APPLE_ID="${APPLE_ID:-${2:-}}"
APP_PASSWORD="${APP_PASSWORD:-${3:-}}"
TEAM_ID="${TEAM_ID:-${4:-}}"

[[ -z "$APP_PATH" || -z "$APPLE_ID" || -z "$APP_PASSWORD" || -z "$TEAM_ID" ]] && { usage; exit 1; }
[[ ! -d "$APP_PATH" ]] && echo "App not found: $APP_PATH" && exit 1

echo "Signing $APP_PATH with team $TEAM_ID..."

# Sign the app bundle
codesign --force --deep --sign "Developer ID Application: $TEAM_ID" \
  --options runtime \
  --entitlements build/darwin/fingerbrower.entitlements \
  "$APP_PATH"

echo "Signing complete. Submitting for notarization..."

# Submit for notarization
xcrun notarytool submit "$APP_PATH" \
  --apple-id "$APPLE_ID" \
  --password "$APP_PASSWORD" \
  --team-id "$TEAM_ID" \
  --wait

# Staple the notarization ticket
xcrun stapler staple "$APP_PATH"

echo "Notarization complete: $APP_PATH"
