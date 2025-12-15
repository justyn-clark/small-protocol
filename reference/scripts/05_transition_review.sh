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
echo "== Transition draft -> generated -> reviewed =="

ART_ID=$(cat "$TMP_DIR/artifact_id")

# draft -> generated
curl -sS -X POST "$API_BASE/artifacts/$ART_ID/transition"   -H "Content-Type: application/json"   -d '{"manifest_name":"content-publishing","to_state":"generated","actor_type":"system","actor_id":"reference-script"}' | jq_or_cat

# generated -> reviewed
curl -sS -X POST "$API_BASE/artifacts/$ART_ID/transition"   -H "Content-Type: application/json"   -d '{"manifest_name":"content-publishing","to_state":"reviewed","actor_type":"human","actor_id":"reviewer"}' | jq_or_cat

echo "Transitions to reviewed complete."
