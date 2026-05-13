#!/bin/bash
set -e

echo "=== Building fingerbrower ==="

# Build with wails (skip mod tidy to avoid dependency issues)
echo "Building wails app..."
wails build -m

# Copy Chromium.app to bundle Resources
APP_BUNDLE="./build/bin/fingerbrower.app"
RESOURCES_DIR="$APP_BUNDLE/Contents/Resources"
SOURCE_BROWSER="./resources/selfbuilt/Chromium.app"
TARGET_BROWSER="$RESOURCES_DIR/selfbuilt/Chromium.app"

if [ -d "$SOURCE_BROWSER" ]; then
    echo "Copying Chromium.app to bundle Resources..."
    mkdir -p "$RESOURCES_DIR/selfbuilt"
    rm -rf "$TARGET_BROWSER"
    cp -R "$SOURCE_BROWSER" "$TARGET_BROWSER"
    echo "Done! Browser binary at: $TARGET_BROWSER/Contents/MacOS/Chromium"
else
    echo "Warning: $SOURCE_BROWSER not found, skipping browser binary copy"
fi

echo "=== Build complete ==="
