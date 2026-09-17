#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

BIN="${BIN_PATH:-./bin/small}"
if [ ! -x "$BIN" ]; then
  make small-build
fi

count=0
for workspace in examples/*; do
  if [ ! -d "$workspace/.small" ]; then
    continue
  fi

  printf 'Checking %s...\n' "$workspace"
  "$BIN" check --strict --dir "$workspace" --workspace examples
  count=$((count + 1))
done

if [ "$count" -eq 0 ]; then
  echo "ERROR: no example workspaces found" >&2
  exit 1
fi

printf 'All %d example workspaces passed strict validation.\n' "$count"
