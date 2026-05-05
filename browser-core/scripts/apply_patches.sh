#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'EOF'
Usage: apply_patches.sh --src <chromium-src> --overlay <overlay-root> [--train N]

Applies patch overlay files to a Chromium source tree.
EOF
}

SRC_DIR=""
OVERLAY_DIR=""
PATCH_TRAIN=""

while [[ $# -gt 0 ]]; do
  case "$1" in
    --src)
      SRC_DIR="$2"
      shift 2
      ;;
    --overlay)
      OVERLAY_DIR="$2"
      shift 2
      ;;
    --train)
      PATCH_TRAIN="$2"
      shift 2
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "Unknown argument: $1"
      usage
      exit 1
      ;;
  esac
done

if [[ -z "$SRC_DIR" || -z "$OVERLAY_DIR" ]]; then
  usage
  exit 1
fi

if [[ -z "$PATCH_TRAIN" ]]; then
  PATCH_TRAIN="$(cat "$OVERLAY_DIR/patch_train" | tr -d '[:space:]')"
fi

if [[ -z "$PATCH_TRAIN" ]]; then
  echo "patch_train is required"
  exit 1
fi

DOMAINS=(identity rendering automation network behavior profile)

read_json_array() {
  local file="$1"
  local key="$2"
  python3 - "$file" "$key" <<'PY'
import json, sys
path = sys.argv[1]
key = sys.argv[2]
with open(path, "r", encoding="utf-8") as f:
    data = json.load(f)
items = data.get(key, [])
for item in items:
    print(item)
PY
}

for domain in "${DOMAINS[@]}"; do
  patchset="$OVERLAY_DIR/patches/$domain/train-$PATCH_TRAIN/patchset.json"
  if [[ ! -f "$patchset" ]]; then
    echo "missing patchset: $patchset"
    exit 1
  fi
  while IFS= read -r patch_id; do
    [[ -z "$patch_id" ]] && continue
    patch_name="${patch_id#${domain}/}"
    patch_dir="$OVERLAY_DIR/patches/$domain/train-$PATCH_TRAIN/$patch_name"
    metadata="$patch_dir/metadata.json"
    if [[ ! -f "$metadata" ]]; then
      echo "missing metadata: $metadata"
      exit 1
    fi
    while IFS= read -r patch_file; do
      [[ -z "$patch_file" ]] && continue
      file_path="$patch_dir/$patch_file"
      if [[ ! -f "$file_path" ]]; then
        echo "missing patch file: $file_path"
        exit 1
      fi
      git -C "$SRC_DIR" apply --whitespace=nowarn "$file_path"
    done < <(read_json_array "$metadata" "files")
  done < <(read_json_array "$patchset" "patches")
done
