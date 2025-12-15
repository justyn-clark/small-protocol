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
echo "== Create draft artifact =="

RESP=$(curl -sS -X POST "$API_BASE/artifacts"   -H "Content-Type: application/json"   -d @../workflow/example-artifact.draft.json)

echo "$RESP" | jq_or_cat

ART_ID=$(echo "$RESP" | python -c "import sys, json; print(json.load(sys.stdin)['id'])")
echo -n "$ART_ID" > "$TMP_DIR/artifact_id"
echo "Saved artifact_id=$ART_ID"
