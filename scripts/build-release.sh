#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DIST_DIR="$ROOT_DIR/dist"
VERSION="${RELEASE_VERSION:-1.0.0}"
BUILD_PKG="./cmd/small"

mkdir -p "$DIST_DIR"
rm -f "$DIST_DIR"/small_"$VERSION"_*.tar.gz
rm -f "$DIST_DIR"/sha256sums.txt

build_target() {
  local goos="$1"
  local goarch="$2"
  local out_name="small_${VERSION}_${goos}_${goarch}.tar.gz"
  local tmp_dir
  tmp_dir="$(mktemp -d)"
  trap 'rm -rf "$tmp_dir"' RETURN

  GOOS="$goos" GOARCH="$goarch" CGO_ENABLED=0 \
    go build \
      -ldflags="-s -w -X github.com/justyn-clark/small-protocol/internal/version.Version=${VERSION} -X github.com/justyn-clark/small-protocol/internal/version.Commit=$(git rev-parse --short=12 HEAD) -X github.com/justyn-clark/small-protocol/internal/version.Date=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
      -o "$tmp_dir/small" \
      "$BUILD_PKG"

  tar -czf "$DIST_DIR/$out_name" -C "$tmp_dir" small
  rm -rf "$tmp_dir"
}

cd "$ROOT_DIR"

build_target darwin arm64
build_target darwin amd64
build_target linux amd64
build_target linux arm64

(
  cd "$DIST_DIR"
  shasum -a 256 small_"$VERSION"_*.tar.gz > sha256sums.txt
)

echo "Release artifacts:"
ls -1 "$DIST_DIR"/small_"$VERSION"_*.tar.gz "$DIST_DIR"/sha256sums.txt
