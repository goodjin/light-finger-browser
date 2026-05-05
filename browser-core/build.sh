#!/usr/bin/env bash
set -euo pipefail

# One-click stealth browser build
# Usage: build.sh [--platform PLATFORM]

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
WORKSPACE_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
SRC_DIR="$WORKSPACE_ROOT/src"
PATCH_DIR="$WORKSPACE_ROOT/patches"
PATCH_TRAIN="$(cat "$WORKSPACE_ROOT/patch_train")"
OUT_DIR="$WORKSPACE_ROOT/out"
ARTIFACTS_DIR="$WORKSPACE_ROOT/artifacts"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

PLATFORM="${1:-}"
OS="$(uname | tr '[:upper:]' '[:lower:]')"

usage() {
    cat <<EOF
Usage: $0 [PLATFORM]

PLATFORM: darwin-arm64 | darwin-amd64 | linux-amd64 | windows-amd64 | all
           (default: auto-detect current platform)

Example:
  $0                  # auto-detect
  $0 darwin-arm64    # specific platform
  $0 all              # build all platforms
EOF
}

die() { echo -e "${RED}ERROR: $1${NC}" >&2; exit 1; }
info() { echo -e "${GREEN}[INFO] $1${NC}"; }
warn() { echo -e "${YELLOW}WARNING: $1${NC}" >&2; }

# Auto-detect platform
detect_platform() {
    local arch
    arch=$(uname -m)
    local os
    os=$(uname | tr '[:upper:]' '[:lower:]')
    case "$os-$arch" in
        darwin-arm64) echo "darwin-arm64" ;;
        darwin-x86_64) echo "darwin-amd64" ;;
        linux-x86_64) echo "linux-amd64" ;;
        *) die "Unknown platform: $os-$arch" ;;
    esac
}

[[ -z "$PLATFORM" ]] && PLATFORM=$(detect_platform)

info "Building stealth browser for: $PLATFORM"
info "Patch train: $PATCH_TRAIN"

# Check prerequisites
command -v ninja &>/dev/null || { warn "ninja not found. Using depot_tools gn+ninja."; }
command -v gn &>/dev/null || { warn "gn not found. Using depot_tools."; }

# Step 1: Fetch source if needed
if [[ ! -d "$SRC_DIR" || ! -f "$SRC_DIR/.repo" ]]; then
    info "Fetching Chromium source..."
    "$SCRIPT_DIR/scripts/fetch_chromium.sh"
else
    info "Using existing source at $SRC_DIR"
fi

cd "$SRC_DIR"

# Step 2: Apply patches
info "Applying stealth patches (train $PATCH_TRAIN)..."
"$SCRIPT_DIR/scripts/apply_patches.sh" \
    --src "$SRC_DIR" \
    --overlay "$PATCH_DIR" \
    --train "$PATCH_TRAIN" || die "Patch application failed"

# Step 3: Generate build config
info "Generating build configuration..."
mkdir -p "$OUT_DIR/release"

export PATH="$WORKSPACE_ROOT/depot_tools:$PATH"

# GN arguments for stealth browser
GN_ARGS="is_debug=false
is_official_build=true
is_component_build=false
proprietary_codecs=true
ffmpeg_branding=Chrome
chrome_pgo_phase=0
use_thin_lto=false
enable_nacl=false
treat_warnings_as_errors=false
fieldtrial_testing_like_official_build=true
"

case "$PLATFORM" in
    darwin-arm64)
        GN_ARGS="$GN_ARGS
target_cpu=\"arm64\"
target_os=\"mac\"
mac_sdk_min=10.15
"
        ;;
    darwin-amd64)
        GN_ARGS="$GN_ARGS
target_cpu=\"x64\"
target_os=\"mac\"
mac_sdk_min=10.15
"
        ;;
    linux-amd64)
        GN_ARGS="$GN_ARGS
target_cpu=\"x64\"
target_os=\"linux\"
"
        ;;
    windows-amd64)
        GN_ARGS="$GN_ARGS
target_cpu=\"x64\"
target_os=\"win\"
"
        ;;
    *) die "Unsupported platform: $PLATFORM" ;;
esac

echo "$GN_ARGS" | gn gen "$OUT_DIR/release" --args=- 2>&1 | head -20 || die "GN gen failed"

# Step 4: Build
info "Building Chromium (this may take 30-60 minutes)..."
ninja -C "$OUT_DIR/release" chrome "chrome/installer/linux/package.desktop" 2>&1 | tail -20 || die "Build failed"

# Step 5: Package
info "Packaging artifacts..."
mkdir -p "$ARTIFACTS_DIR/$PLATFORM"

case "$PLATFORM" in
    darwin-*)
        cp -R "$OUT_DIR/release/chrome_mac/Chromium.app" "$ARTIFACTS_DIR/$PLATFORM/" || true
        if [[ -d "$OUT_DIR/release/Chromium.app" ]]; then
            cp -R "$OUT_DIR/release/Chromium.app" "$ARTIFACTS_DIR/$PLATFORM/"
        fi
        ;;
    linux-amd64)
        cp "$OUT_DIR/release/chrome" "$ARTIFACTS_DIR/$PLATFORM/chromium"
        ;;
    windows-amd64)
        cp "$OUT_DIR/release/chrome.exe" "$ARTIFACTS_DIR/$PLATFORM/chromium.exe" || true
        ;;
esac

# Step 6: Generate artifacts.json
info "Generating artifacts.json..."
BROWSER_VERSION="$(cat "$WORKSPACE_ROOT/chromium_milestone" 2>/dev/null || echo "1.0").$PATCH_TRAIN.0"
REVISION="$(cat "$WORKSPACE_ROOT/chromium_revision" 2>/dev/null || echo "unknown")"

cat > "$ARTIFACTS_DIR/$PLATFORM/artifacts.json" <<EOF
{
  "version": 2,
  "channel": "stable",
  "browser_version": "$BROWSER_VERSION",
  "chromium_revision": "$REVISION",
  "patch_train": $PATCH_TRAIN,
  "platform": "$PLATFORM",
  "build_timestamp": "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
}
EOF

info "Build complete!"
info "Artifacts: $ARTIFACTS_DIR/$PLATFORM/"
ls -la "$ARTIFACTS_DIR/$PLATFORM/"

# Print summary
echo ""
echo "=== Build Summary ==="
echo "Platform:    $PLATFORM"
echo "Version:     $BROWSER_VERSION"
echo "Revision:    $REVISION"
echo "Patch Train: $PATCH_TRAIN"
