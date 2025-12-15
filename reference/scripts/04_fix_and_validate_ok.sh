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
echo "== Fix artifact back to valid draft and validate OK =="

ART_ID=$(cat "$TMP_DIR/artifact_id")

# restore valid data from fixture
VALID_DATA=$(python -c "import json; d=json.load(open('../workflow/example-artifact.draft.json')); print(json.dumps({'data': d['data']}))")
curl -sS -X PATCH "$API_BASE/artifacts/$ART_ID"   -H "Content-Type: application/json"   -d "$VALID_DATA" | jq_or_cat

RESP=$(curl -sS -X POST "$API_BASE/artifacts/$ART_ID/validate"   -H "Content-Type: application/json")
echo "$RESP" | jq_or_cat

OK=$(echo "$RESP" | python -c "import sys, json; print('true' if json.load(sys.stdin).get('ok') else 'false')")
if [ "$OK" != "true" ]; then
  echo "ERROR: validation did not pass"
  exit 1
fi

echo "Validation passed."
