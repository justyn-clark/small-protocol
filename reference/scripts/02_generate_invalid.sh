#!/usr/bin/env bash
set -euo pipefail

API_BASE="${API_BASE:-http://localhost:8000}"
TMP_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/.tmp"
mkdir -p "$TMP_DIR"

jq_or_cat() {{
  if command -v jq >/dev/null 2>&1; then
    jq .
  else
    cat
  fi
}}
echo "== Update artifact with invalid data (simulate bad agent output) =="

ART_ID=$(cat "$TMP_DIR/artifact_id")
curl -sS -X PATCH "$API_BASE/artifacts/$ART_ID"   -H "Content-Type: application/json"   -d @../workflow/example-artifact.invalid.json | jq_or_cat

echo "Updated artifact with invalid data."
