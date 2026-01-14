#!/usr/bin/env bash
set -euo pipefail

# Verification script for small-protocol
# Creates an isolated SMALL workspace so repo-root .small is never touched.
# Fails on any dirty working tree changes or go mod tidy diffs.

BIN="${BIN_PATH:-./bin/small}"

WORKDIR="${SMALL_VERIFY_DIR:-.tmp/small-verify}"
SMALLDIR="$WORKDIR/.small"

rm -rf "$WORKDIR"
mkdir -p "$WORKDIR"

echo "=== Step 1: Go version ==="
GO_VER="$(go version)"
echo "$GO_VER"

echo "=== Enforcing Go toolchain version ==="
echo "$GO_VER" | grep -E "go1\.22\." >/dev/null || {
  echo "ERROR: Go 1.22.x is required, found: $GO_VER"
  exit 1
}
echo "✓ Go toolchain pinned to 1.22.x"

echo "=== README acronym check ==="
grep -q 'SMALL (Schema, Manifest, Artifact, Lineage, Lifecycle)' README.md || {
  echo "ERROR: README must define SMALL as: Schema, Manifest, Artifact, Lineage, Lifecycle"
  exit 1
}
echo "✓ README acronym is correct"

echo "=== Step 2: go mod tidy (must produce no diffs) ==="
if ! git diff --quiet; then
  echo "ERROR: working tree is dirty before verify. Commit or stash changes."
  git status --porcelain
  exit 1
fi

go mod tidy

if ! git diff --quiet; then
  echo "ERROR: go mod tidy produced diffs"
  git status --porcelain
  git diff
  exit 1
fi
echo "✓ go mod tidy produced no diffs"

echo "=== Step 3: Build CLI ==="
make sync-schemas
make small-build

echo "=== Step 4: Version command ==="
"$BIN" version

echo "=== Step 5: Initialize isolated test workspace ==="
"$BIN" init --dir "$WORKDIR" --force --intent "CI verify workspace"
echo "Initialized SMALL project in $SMALLDIR"

echo "=== Step 6: Validate spec examples ==="
"$BIN" validate --dir spec/small/v1.0.0/examples

echo "=== Step 7: Lint spec examples ==="
"$BIN" lint --dir spec/small/v1.0.0/examples

echo "=== Step 8: Generate handoff in isolated workspace ==="
"$BIN" handoff --dir "$WORKDIR" --summary "Verification checkpoint"

echo "=== Step 9: Test plan command in isolated workspace ==="
ADD_OUT="$("$BIN" plan --dir "$WORKDIR" --add "Verification test task")"
echo "$ADD_OUT"
TASK_ID="$(echo "$ADD_OUT" | sed -nE 's/^Added task ([^:]+):.*/\1/p' | tail -n1)"
if [ -z "$TASK_ID" ]; then
  echo "ERROR: could not parse task id from plan --add output"
  exit 1
fi
"$BIN" plan --dir "$WORKDIR" --done "$TASK_ID"

echo "=== Step 10: Test status command in isolated workspace ==="
"$BIN" status --dir "$WORKDIR"
"$BIN" status --dir "$WORKDIR" --json >/dev/null

echo "=== Step 11: Test apply command (dry-run) in isolated workspace ==="
"$BIN" apply --dir "$WORKDIR" --dry-run
"$BIN" apply --dir "$WORKDIR" --dry-run --cmd "echo hello"

echo "=== Step 12: Test apply command (execution) in isolated workspace ==="
"$BIN" apply --dir "$WORKDIR" --cmd "echo 'SMALL apply test'"

echo "=== Step 13: Verify isolated workspace ==="
"$BIN" verify --dir "$WORKDIR"

echo "=== Step 14: Run tests and format check ==="
make small-test
make small-format-check

echo "=== All verification steps passed ==="
