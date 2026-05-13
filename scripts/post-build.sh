#!/usr/bin/env bash
# Post-build script to copy Chromium.app into the built app
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
APP_PATH="$PROJECT_ROOT/build/bin/fingerbrower.app"
RESOURCES_DIR="$APP_PATH/Contents/Resources"
CHROMIUM_SOURCE="$PROJECT_ROOT/browser-core/artifacts/Chromium.app"
CHROMIUM_DEST="$RESOURCES_DIR/selfbuilt/Chromium.app"

# Check if Chromium.app exists
if [ ! -d "$CHROMIUM_SOURCE" ]; then
    echo "Error: Chromium.app not found at $CHROMIUM_SOURCE"
    exit 1
fi

# Check if app was built
if [ ! -d "$APP_PATH" ]; then
    echo "Error: fingerbrower.app not found at $APP_PATH"
    exit 1
fi

# Remove old Chromium if exists
if [ -d "$CHROMIUM_DEST" ]; then
    rm -rf "$CHROMIUM_DEST"
fi

# Create parent directory
mkdir -p "$RESOURCES_DIR"

# Copy Chromium.app using rsync for proper handling
echo "Copying Chromium.app to $CHROMIUM_DEST..."
rsync -a "$CHROMIUM_SOURCE/" "$CHROMIUM_DEST/"

echo "Done! Chromium.app size: $(du -sh "$CHROMIUM_DEST" | cut -f1)"
