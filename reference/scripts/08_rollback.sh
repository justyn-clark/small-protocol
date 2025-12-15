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
echo "== Rollback artifact to version 1 =="

ART_ID=$(cat "$TMP_DIR/artifact_id")

curl -sS -X POST "$API_BASE/artifacts/$ART_ID/rollback"   -H "Content-Type: application/json"   -d '{"target_version":1,"actor_type":"human","actor_id":"ops"}' | jq_or_cat

echo "-- Events after rollback"
curl -sS "$API_BASE/artifacts/$ART_ID/events" | jq_or_cat
