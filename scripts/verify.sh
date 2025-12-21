#!/bin/bash
set -euo pipefail

# Verification script for small-protocol
# All steps must pass. Exit on first failure.

echo "=== Step 1: Go version ==="
go version

echo ""
echo "=== Step 2: go mod tidy (must produce no diffs) ==="
if ! git diff --quiet -- go.mod go.sum 2>/dev/null; then
    echo "ERROR: go.mod or go.sum has local diffs before tidy"
    git status --porcelain
    echo "--- go.mod/go.sum diff ---"
    git diff -- go.mod go.sum
    exit 1
fi
go mod tidy
if ! git diff --quiet -- go.mod go.sum 2>/dev/null; then
    echo "ERROR: go mod tidy produced diffs"
    git diff -- go.mod go.sum
    exit 1
fi
echo "✓ go mod tidy produced no diffs"

echo ""
echo "=== Step 3: Build CLI ==="
make small-build

echo ""
echo "=== Step 4: Version command ==="
./bin/small version

echo ""
echo "=== Step 5: Initialize test project ==="
rm -rf .small
./bin/small init --name testproject --force

echo ""
echo "=== Step 6: Validate repo root ==="
./bin/small validate

echo ""
echo "=== Step 7: Validate examples directory ==="
./bin/small validate --dir spec/small/v0.1/examples

echo ""
echo "=== Step 8: Lint examples directory ==="
./bin/small lint --dir spec/small/v0.1/examples

echo ""
echo "=== Step 9: Generate handoff ==="
./bin/small handoff --dir . --recent 3

echo ""
echo "=== Step 10: Run tests and format check ==="
make small-test small-format-check

echo ""
echo "=== All verification steps passed ==="

