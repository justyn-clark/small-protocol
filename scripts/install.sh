#!/usr/bin/env bash
set -euo pipefail

REPO="justyn-clark/small-protocol"
DEFAULT_INSTALL_DIR="$HOME/.local/bin"
SYSTEM_INSTALL_DIR="/usr/local/bin"

VERSION=""
INSTALL_DIR="$DEFAULT_INSTALL_DIR"
USE_SYSTEM=0
DIR_SPECIFIED=0
SYSTEM_SPECIFIED=0
DRY_RUN=0
RELEASE_JSON_FILE=""

usage() {
  cat <<USAGE
Usage: install.sh [options]

Installs SMALL after downloading a release archive and verifying SHA256.
No compilation required. Pre-built binaries are downloaded and checksum verified.

Options:
  --version vX.Y.Z   Install an explicit tag (default: latest release)
  --dir PATH         Install directory override (default: ~/.local/bin)
  --system           Install to /usr/local/bin (may require sudo)
  --release-json     Use local release JSON file (testing only)
  --dry-run          Resolve release and print actions without changes
  -h, --help         Show this help
USAGE
}

while [ "$#" -gt 0 ]; do
  case "$1" in
    --version)
      [ "$#" -ge 2 ] || { echo "error: --version requires a value" >&2; exit 1; }
      VERSION="$2"
      shift 2
      ;;
    --dir)
      [ "$#" -ge 2 ] || { echo "error: --dir requires a value" >&2; exit 1; }
      INSTALL_DIR="$2"
      DIR_SPECIFIED=1
      shift 2
      ;;
    --system)
      USE_SYSTEM=1
      SYSTEM_SPECIFIED=1
      INSTALL_DIR="$SYSTEM_INSTALL_DIR"
      shift
      ;;
    --release-json)
      [ "$#" -ge 2 ] || { echo "error: --release-json requires a value" >&2; exit 1; }
      RELEASE_JSON_FILE="$2"
      shift 2
      ;;
    --dry-run)
      DRY_RUN=1
      shift
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "error: unknown argument: $1" >&2
      usage >&2
      exit 1
      ;;
  esac
done

if [ "$DIR_SPECIFIED" -eq 1 ] && [ "$SYSTEM_SPECIFIED" -eq 1 ]; then
  echo "error: --system cannot be combined with --dir" >&2
  exit 1
fi

require_cmd() {
  command -v "$1" >/dev/null 2>&1 || {
    echo "error: required command not found: $1" >&2
    exit 1
  }
}

require_cmd curl
require_cmd tar
require_cmd python3
require_cmd uname

sha256_file() {
  local file="$1"
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$file" | awk '{print $1}'
    return
  fi
  if command -v shasum >/dev/null 2>&1; then
    shasum -a 256 "$file" | awk '{print $1}'
    return
  fi
  echo "error: missing checksum tool (need sha256sum or shasum)" >&2
  exit 1
}

os_name() {
  case "$(uname -s)" in
    Darwin) echo "darwin" ;;
    Linux) echo "linux" ;;
    *) echo "error: unsupported OS $(uname -s)" >&2; exit 1 ;;
  esac
}

arch_name() {
  case "$(uname -m)" in
    x86_64|amd64) echo "amd64" ;;
    arm64|aarch64) echo "arm64" ;;
    *) echo "error: unsupported architecture $(uname -m)" >&2; exit 1 ;;
  esac
}

api_endpoint() {
  if [ -n "$VERSION" ]; then
    echo "https://api.github.com/repos/$REPO/releases/tags/$VERSION"
  else
    echo "https://api.github.com/repos/$REPO/releases/latest"
  fi
}

curl_fetch() {
  local url="$1"
  local out="$2"

  if [ -n "${GITHUB_TOKEN:-}" ]; then
    curl -fsSL \
      -H "Accept: application/vnd.github+json" \
      -H "Authorization: Bearer ${GITHUB_TOKEN}" \
      "$url" \
      -o "$out"
  else
    curl -fsSL \
      -H "Accept: application/vnd.github+json" \
      "$url" \
      -o "$out"
  fi
}

asset_url_by_name() {
  local json_file="$1"
  local asset_name="$2"

  python3 - "$json_file" "$asset_name" <<"PY"
import json
import sys

with open(sys.argv[1], "r", encoding="utf-8") as f:
    release = json.load(f)

needle = sys.argv[2]
for asset in release.get("assets", []):
    if asset.get("name") == needle:
        print(asset.get("browser_download_url", ""))
        raise SystemExit(0)

raise SystemExit(1)
PY
}

tag_name() {
  local json_file="$1"
  python3 - "$json_file" <<"PY"
import json
import sys

with open(sys.argv[1], "r", encoding="utf-8") as f:
    release = json.load(f)

print(release.get("tag_name", ""))
PY
}

validate_archive_paths() {
  local archive="$1"
  local list_file="$TMP_DIR/tar-entries.txt"

  if ! tar -tzf "$archive" >"$list_file"; then
    echo "error: unable to list archive entries for $archive" >&2
    exit 1
  fi

  if ! python3 - "$list_file" <<"PY"
from pathlib import PurePosixPath
import sys

bad = []
with open(sys.argv[1], "r", encoding="utf-8", errors="replace") as f:
    for raw in f:
        entry = raw.strip()
        if not entry:
            continue
        while entry.startswith("./"):
            entry = entry[2:]
        if not entry:
            continue
        p = PurePosixPath(entry)
        if p.is_absolute() or ".." in p.parts:
            bad.append(entry)

if bad:
    for item in bad[:5]:
        print(item)
    raise SystemExit(1)
PY
  then
    echo "error: archive contains unsafe path entries (path traversal or absolute paths)" >&2
    exit 1
  fi
}

TMP_DIR="$(mktemp -d)"
cleanup() {
  rm -rf "$TMP_DIR"
}
trap cleanup EXIT

RELEASE_JSON="$TMP_DIR/release.json"
if [ -n "$RELEASE_JSON_FILE" ]; then
  cp "$RELEASE_JSON_FILE" "$RELEASE_JSON"
else
  curl_fetch "$(api_endpoint)" "$RELEASE_JSON"
fi

TAG="$(tag_name "$RELEASE_JSON")"
if [ -z "$TAG" ]; then
  echo "error: could not read tag_name from release metadata" >&2
  exit 1
fi

OS="$(os_name)"
ARCH="$(arch_name)"
ASSET_NAME="small-${TAG}-${OS}-${ARCH}.tar.gz"
CHECKSUMS_NAME="checksums.txt"

ASSET_URL="$(asset_url_by_name "$RELEASE_JSON" "$ASSET_NAME" || true)"
CHECKSUMS_URL="$(asset_url_by_name "$RELEASE_JSON" "$CHECKSUMS_NAME" || true)"

# Fallback for older naming convention (v1.0.2 and earlier)
if [ -z "$ASSET_URL" ]; then
  OS_OLD=""
  case "$OS" in
    darwin) OS_OLD="Darwin" ;;
    linux) OS_OLD="Linux" ;;
    windows) OS_OLD="Windows" ;;
  esac

  ARCH_OLD="$ARCH"
  if [ "$ARCH" = "amd64" ]; then ARCH_OLD="x86_64"; fi
  TAG_OLD="${TAG#v}"
  ASSET_NAME_OLD="small-protocol_${TAG_OLD}_${OS_OLD}_${ARCH_OLD}.tar.gz"

  ASSET_URL_OLD="$(asset_url_by_name "$RELEASE_JSON" "$ASSET_NAME_OLD" || true)"
  if [ -n "$ASSET_URL_OLD" ]; then
    echo "Using legacy asset pattern fallback: $ASSET_NAME_OLD"
    ASSET_NAME="$ASSET_NAME_OLD"
    ASSET_URL="$ASSET_URL_OLD"
  fi
fi

if [ -z "$ASSET_URL" ]; then
  echo "error: release $TAG is missing asset: $ASSET_NAME" >&2
  exit 1
fi
if [ -z "$CHECKSUMS_URL" ]; then
  echo "error: release $TAG is missing asset: $CHECKSUMS_NAME" >&2
  exit 1
fi

if [ "$DRY_RUN" -eq 1 ]; then
  echo "Resolved release: $TAG"
  echo "Resolved asset: $ASSET_NAME"
  echo "Checksums asset: $CHECKSUMS_NAME"
  echo "Install dir: $INSTALL_DIR"
  exit 0
fi

mkdir -p "$TMP_DIR/downloads"
ARCHIVE_PATH="$TMP_DIR/downloads/$ASSET_NAME"
CHECKSUMS_PATH="$TMP_DIR/downloads/$CHECKSUMS_NAME"

curl_fetch "$ASSET_URL" "$ARCHIVE_PATH"
curl_fetch "$CHECKSUMS_URL" "$CHECKSUMS_PATH"

EXPECTED_SUM="$(awk -v n="$ASSET_NAME" '
  $2 == n || $2 == "*" n { print $1; found=1; exit }
  END { if (!found) exit 1 }
' "$CHECKSUMS_PATH" || true)"

if [ -z "$EXPECTED_SUM" ]; then
  echo "error: could not find checksum for $ASSET_NAME in checksums.txt" >&2
  exit 1
fi

ACTUAL_SUM="$(sha256_file "$ARCHIVE_PATH")"
if [ "$EXPECTED_SUM" != "$ACTUAL_SUM" ]; then
  echo "error: checksum mismatch for $ASSET_NAME" >&2
  echo "expected: $EXPECTED_SUM" >&2
  echo "actual:   $ACTUAL_SUM" >&2
  exit 1
fi

validate_archive_paths "$ARCHIVE_PATH"

EXTRACT_DIR="$TMP_DIR/extract"
mkdir -p "$EXTRACT_DIR"
tar -xzf "$ARCHIVE_PATH" --no-same-owner -C "$EXTRACT_DIR"

BIN_PATH="$(find "$EXTRACT_DIR" -type f -name small | head -n 1 || true)"
if [ -z "$BIN_PATH" ]; then
  echo "error: extracted archive did not contain a small binary" >&2
  exit 1
fi

if [ -d "$INSTALL_DIR" ] && [ ! -w "$INSTALL_DIR" ]; then
  if [ "$USE_SYSTEM" -eq 1 ]; then
    sudo install -d "$INSTALL_DIR"
  else
    echo "error: install dir is not writable: $INSTALL_DIR" >&2
    echo "hint: use --system or choose a writable --dir" >&2
    exit 1
  fi
else
  mkdir -p "$INSTALL_DIR"
fi

if [ -w "$INSTALL_DIR" ]; then
  install -m 0755 "$BIN_PATH" "$INSTALL_DIR/small"
else
  sudo install -m 0755 "$BIN_PATH" "$INSTALL_DIR/small"
fi

echo "Installed small $TAG to $INSTALL_DIR/small"

case ":$PATH:" in
  *":$INSTALL_DIR:"*)
    ;;
  *)
    echo "Add $INSTALL_DIR to PATH, for example:"
    echo "  export PATH=\"$INSTALL_DIR:\$PATH\""
    ;;
esac
