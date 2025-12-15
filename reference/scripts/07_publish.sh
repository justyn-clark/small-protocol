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
echo "== Transition approved -> published (publish gate) =="

ART_ID=$(cat "$TMP_DIR/artifact_id")
curl -sS -X POST "$API_BASE/artifacts/$ART_ID/transition"   -H "Content-Type: application/json"   -d '{"manifest_name":"content-publishing","to_state":"published","actor_type":"system","actor_id":"publisher"}' | jq_or_cat
