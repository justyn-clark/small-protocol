#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TMP_DIR="$(mktemp -d)"
trap 'rm -rf "$TMP_DIR"' EXIT

os_name() {
  case "$(uname -s)" in
    Darwin) echo "darwin" ;;
    Linux) echo "linux" ;;
    *) echo "unsupported OS: $(uname -s)" >&2; exit 1 ;;
  esac
}

arch_name() {
  case "$(uname -m)" in
    x86_64|amd64) echo "amd64" ;;
    arm64|aarch64) echo "arm64" ;;
    *) echo "unsupported arch: $(uname -m)" >&2; exit 1 ;;
  esac
}

sha256_file() {
  local file="$1"
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$file" | awk '{print $1}'
    return
  fi
  shasum -a 256 "$file" | awk '{print $1}'
}

TAG="v9.9.9"
OS="$(os_name)"
ARCH="$(arch_name)"
ASSET_NAME="small-${TAG}-${OS}-${ARCH}.tar.gz"

mkdir -p "$TMP_DIR/assets" "$TMP_DIR/bin" "$TMP_DIR/build"

cat > "$TMP_DIR/build/small" <<'BINARY'
#!/usr/bin/env bash
if [ "${1:-}" = "--version" ] || [ "${1:-}" = "version" ]; then
  echo "small v9.9.9-test"
  exit 0
fi
echo "small test binary"
BINARY
chmod +x "$TMP_DIR/build/small"

tar -czf "$TMP_DIR/assets/$ASSET_NAME" -C "$TMP_DIR/build" small
HASH="$(sha256_file "$TMP_DIR/assets/$ASSET_NAME")"
printf "%s  %s\n" "$HASH" "$ASSET_NAME" > "$TMP_DIR/assets/checksums.txt"

cat > "$TMP_DIR/release.json" <<JSON
{
  "tag_name": "$TAG",
  "assets": [
    {
      "name": "$ASSET_NAME",
      "browser_download_url": "file://$TMP_DIR/assets/$ASSET_NAME"
    },
    {
      "name": "checksums.txt",
      "browser_download_url": "file://$TMP_DIR/assets/checksums.txt"
    }
  ]
}
JSON

bash "$ROOT_DIR/scripts/install.sh" --release-json "$TMP_DIR/release.json" --dir "$TMP_DIR/bin"
OUTPUT="$($TMP_DIR/bin/small --version)"
echo "$OUTPUT" | grep -q "small v9.9.9-test"

cp "$TMP_DIR/assets/checksums.txt" "$TMP_DIR/assets/checksums-corrupt.txt"
printf '0%.0s' {1..64} > "$TMP_DIR/zerohash.txt"
CORRUPT_HASH="$(cat "$TMP_DIR/zerohash.txt")"
printf "%s  %s\n" "$CORRUPT_HASH" "$ASSET_NAME" > "$TMP_DIR/assets/checksums-corrupt.txt"

cat > "$TMP_DIR/release-corrupt.json" <<JSON
{
  "tag_name": "$TAG",
  "assets": [
    {
      "name": "$ASSET_NAME",
      "browser_download_url": "file://$TMP_DIR/assets/$ASSET_NAME"
    },
    {
      "name": "checksums.txt",
      "browser_download_url": "file://$TMP_DIR/assets/checksums-corrupt.txt"
    }
  ]
}
JSON

if bash "$ROOT_DIR/scripts/install.sh" --release-json "$TMP_DIR/release-corrupt.json" --dir "$TMP_DIR/bin-corrupt" >/dev/null 2>&1; then
  echo "expected installer to fail on checksum mismatch" >&2
  exit 1
fi
