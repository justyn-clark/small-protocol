#!/usr/bin/env bash
set -euo pipefail

BIN="${BIN_PATH:-./bin/small}"
case "$BIN" in /*) ;; *) BIN="$(pwd)/${BIN#./}" ;; esac
PROOF_ROOT="${SMALL_PROOF_DIR:-$(mktemp -d -t small-session-proof.XXXXXX)}"
mkdir -p "$PROOF_ROOT/receipts"

git_identity() {
  git -C "$1" config user.name "SMALL local proof"
  git -C "$1" config user.email "small-proof@example.invalid"
}
json_value() { sed -n "s/.*\"$2\": \"\([^\"]*\)\".*/\1/p" "$1" | head -n 1; }
state_digest() { find "$1/.small" -type f -exec shasum -a 256 {} \; | sort | shasum -a 256 | awk '{print $1}'; }

SEED="$PROOF_ROOT/seed"
git init -q -b main "$SEED"
git_identity "$SEED"
"$BIN" init --dir "$SEED" --force --no-agents --intent "Local session proof"
"$BIN" handoff --dir "$SEED" --summary "Shared v1 baseline for local proof."
"$BIN" check --strict --dir "$SEED"
git -C "$SEED" add .small .gitignore
git -C "$SEED" commit -q -m "v1 baseline"

# Retain the observed shared-tail v1 merge conflict as a receipt.
V1_REMOTE="$PROOF_ROOT/v1.git"
git clone -q --bare "$SEED" "$V1_REMOTE"
CEREMONY_V1="$PROOF_ROOT/ceremony-v1"
git clone -q "$V1_REMOTE" "$CEREMONY_V1"
git_identity "$CEREMONY_V1"
V1_CEREMONY_OUT="$("$BIN" plan --dir "$CEREMONY_V1" --add "v1 ceremony task")"
V1_CEREMONY_TASK="$(printf '%s\n' "$V1_CEREMONY_OUT" | sed -nE 's/^Added task ([^:]+):.*/\1/p')"
V1_DIRTY_COUNT="$(git -C "$CEREMONY_V1" status --porcelain .small | wc -l | tr -d ' ')"
"$BIN" reconstruct --dir "$CEREMONY_V1" --task "$V1_CEREMONY_TASK" --json >"$PROOF_ROOT/receipts/ceremony-v1-resume.json"
V1_RESUME_BYTES="$(wc -c <"$PROOF_ROOT/receipts/ceremony-v1-resume.json" | tr -d ' ')"
V1_OBLIGATION_VISIBLE=0
if grep -q 'v1 ceremony task' "$PROOF_ROOT/receipts/ceremony-v1-resume.json"; then V1_OBLIGATION_VISIBLE=1; fi
git clone -q "$V1_REMOTE" "$PROOF_ROOT/v1-a"
git clone -q "$V1_REMOTE" "$PROOF_ROOT/v1-b"
git_identity "$PROOF_ROOT/v1-a"; git_identity "$PROOF_ROOT/v1-b"
"$BIN" plan --dir "$PROOF_ROOT/v1-a" --add "v1 branch A"
git -C "$PROOF_ROOT/v1-a" add .small && git -C "$PROOF_ROOT/v1-a" commit -q -m "v1 a"
git -C "$PROOF_ROOT/v1-a" push -q origin HEAD:refs/heads/v1-a
"$BIN" plan --dir "$PROOF_ROOT/v1-b" --add "v1 branch B"
git -C "$PROOF_ROOT/v1-b" add .small && git -C "$PROOF_ROOT/v1-b" commit -q -m "v1 b"
git -C "$PROOF_ROOT/v1-b" fetch -q origin v1-a
set +e
git -C "$PROOF_ROOT/v1-b" merge --no-edit FETCH_HEAD >"$PROOF_ROOT/receipts/v1-merge.txt" 2>&1
V1_MERGE_EXIT=$?
set -e
git -C "$PROOF_ROOT/v1-b" status --porcelain >>"$PROOF_ROOT/receipts/v1-merge.txt"
if [ "$V1_MERGE_EXIT" -eq 0 ]; then echo "expected v1 shared-tail merge conflict" >&2; exit 1; fi

# Preview/apply migration once on the common baseline, defaulting to solo.
MIGRATION="$PROOF_ROOT/migration.json"
"$BIN" migrate --dir "$SEED" --to 2.0.0 --preview --namespace local-proof-baseline --json >"$MIGRATION"
EXPECTED_INPUT="$(json_value "$MIGRATION" expected_input_digest)"
test -n "$EXPECTED_INPUT"
"$BIN" migrate --dir "$SEED" --apply "$MIGRATION" --expect-state "$EXPECTED_INPUT" --json | tee "$PROOF_ROOT/receipts/migration-apply.json"
"$BIN" mode show --dir "$SEED" --json >"$PROOF_ROOT/mode.json"
MODE_FRONTIER="$(json_value "$PROOF_ROOT/mode.json" frontier)"
"$BIN" session start --dir "$SEED" --label mode-controller --json >"$PROOF_ROOT/controller.json"
CONTROLLER="$(json_value "$PROOF_ROOT/controller.json" session_id)"
"$BIN" mode show --dir "$SEED" --json >"$PROOF_ROOT/mode-after-start.json"
MODE_FRONTIER="$(json_value "$PROOF_ROOT/mode-after-start.json" frontier)"
"$BIN" mode set collaborative --dir "$SEED" --session "$CONTROLLER" --expect-state "$MODE_FRONTIER" --reason "local two-clone proof" --json >"$PROOF_ROOT/receipts/mode-change.json"
"$BIN" session close --dir "$SEED" --session "$CONTROLLER" --summary "Collaborative proof baseline prepared." --json >"$PROOF_ROOT/receipts/controller-close.json"
"$BIN" check --strict --dir "$SEED"
git -C "$SEED" add .small
git -C "$SEED" commit -q -m "migrate to session profile"

REMOTE="$PROOF_ROOT/session.git"
git clone -q --bare "$SEED" "$REMOTE"
CEREMONY_V2="$PROOF_ROOT/ceremony-v2"
git clone -q "$REMOTE" "$CEREMONY_V2"
git_identity "$CEREMONY_V2"
"$BIN" session start --dir "$CEREMONY_V2" --label ceremony --json >"$PROOF_ROOT/ceremony-v2-session.json"
CEREMONY_SESSION="$(json_value "$PROOF_ROOT/ceremony-v2-session.json" session_id)"
git -C "$CEREMONY_V2" add .small && git -C "$CEREMONY_V2" commit -q -m "ceremony session baseline"
V2_CEREMONY_OUT="$("$BIN" plan --dir "$CEREMONY_V2" --session "$CEREMONY_SESSION" --add "v2 ceremony task")"
V2_CEREMONY_TASK="$(printf '%s\n' "$V2_CEREMONY_OUT" | sed -nE 's/^Added task [^ ]+ \(([^)]+)\).*/\1/p')"
V2_DIRTY_COUNT="$(git -C "$CEREMONY_V2" status --porcelain .small | wc -l | tr -d ' ')"
"$BIN" reconstruct --dir "$CEREMONY_V2" --resume --session "$CEREMONY_SESSION" --task "$V2_CEREMONY_TASK" --json >"$PROOF_ROOT/receipts/ceremony-v2-resume.json"
V2_RESUME_BYTES="$(wc -c <"$PROOF_ROOT/receipts/ceremony-v2-resume.json" | tr -d ' ')"
V2_OBLIGATION_VISIBLE=0
if grep -q 'v2 ceremony task' "$PROOF_ROOT/receipts/ceremony-v2-resume.json"; then V2_OBLIGATION_VISIBLE=1; fi
printf 'v1_task_action_calls=1\nv1_session_setup_calls=0\nv1_tracked_files_dirtied=%s\nv1_resume_bytes=%s\nv1_obligation_visible=%s\nv2_task_action_calls=1\nv2_one_time_session_setup_calls=1\nv2_tracked_files_dirtied=%s\nv2_resume_bytes=%s\nv2_obligation_visible=%s\n' "$V1_DIRTY_COUNT" "$V1_RESUME_BYTES" "$V1_OBLIGATION_VISIBLE" "$V2_DIRTY_COUNT" "$V2_RESUME_BYTES" "$V2_OBLIGATION_VISIBLE" >"$PROOF_ROOT/receipts/ceremony.txt"
A="$PROOF_ROOT/a"; B="$PROOF_ROOT/b"
git clone -q "$REMOTE" "$A"; git clone -q "$REMOTE" "$B"
git_identity "$A"; git_identity "$B"

"$BIN" session start --dir "$A" --label clone-a --json >"$PROOF_ROOT/a-session.json"
A_SESSION="$(json_value "$PROOF_ROOT/a-session.json" session_id)"
"$BIN" plan --dir "$A" --session "$A_SESSION" --add "same readable title" | tee "$PROOF_ROOT/receipts/a-task.txt"
"$BIN" handoff --dir "$A" --session "$A_SESSION" --summary "Clone A independent narrative." --json >"$PROOF_ROOT/receipts/a-handoff.json"
git -C "$A" add .small && git -C "$A" commit -q -m "clone a independent session"
git -C "$A" push -q origin HEAD:refs/heads/clone-a

"$BIN" session start --dir "$B" --label clone-b --json >"$PROOF_ROOT/b-session.json"
B_SESSION="$(json_value "$PROOF_ROOT/b-session.json" session_id)"
"$BIN" plan --dir "$B" --session "$B_SESSION" --add "same readable title" | tee "$PROOF_ROOT/receipts/b-task.txt"
"$BIN" handoff --dir "$B" --session "$B_SESSION" --summary "Clone B independent narrative." --json >"$PROOF_ROOT/receipts/b-handoff.json"
git -C "$B" add .small && git -C "$B" commit -q -m "clone b independent session"
git -C "$B" fetch -q origin clone-a
git -C "$B" merge -q --no-edit FETCH_HEAD
"$BIN" check --strict --dir "$B"
"$BIN" reconstruct --dir "$B" --limit 20 --json >"$PROOF_ROOT/receipts/independent-merge.json"
git -C "$B" push -q origin HEAD:main
git -C "$A" fetch -q origin main
git -C "$A" merge -q --no-edit origin/main

# From the same frontier, create incompatible outcomes for the shared imported task.
A_OUT="$("$BIN" progress add --dir "$A" --session "$A_SESSION" --task task-1 --status completed --evidence "clone A claim")"
A_EVENT="$(printf '%s\n' "$A_OUT" | awk '{print $NF}')"
printf '%s\n' "$A_OUT" >"$PROOF_ROOT/receipts/a-outcome.txt"
git -C "$A" add .small && git -C "$A" commit -q -m "shared task completed in a"
git -C "$A" push -q origin HEAD:refs/heads/conflict-a

B_OUT="$("$BIN" progress add --dir "$B" --session "$B_SESSION" --task task-1 --status blocked --evidence "clone B claim")"
B_EVENT="$(printf '%s\n' "$B_OUT" | awk '{print $NF}')"
printf '%s\n' "$B_OUT" >"$PROOF_ROOT/receipts/b-outcome.txt"
git -C "$B" add .small && git -C "$B" commit -q -m "shared task blocked in b"
git -C "$B" fetch -q origin conflict-a
git -C "$B" merge -q --no-edit FETCH_HEAD

"$BIN" reconcile --dir "$B" --preview --json >"$PROOF_ROOT/reconcile.json"
CONFLICT_ID="$(json_value "$PROOF_ROOT/reconcile.json" id)"
CONFLICT_FRONTIER="$(json_value "$PROOF_ROOT/reconcile.json" frontier)"
test -n "$CONFLICT_ID"; test -n "$CONFLICT_FRONTIER"
set +e
"$BIN" check --strict --dir "$B" >"$PROOF_ROOT/receipts/conflicted-strict.txt" 2>&1
CONFLICT_CHECK_EXIT=$?
set -e
if [ "$CONFLICT_CHECK_EXIT" -eq 0 ]; then echo "strict unexpectedly accepted semantic conflict" >&2; exit 1; fi
set +e
"$BIN" handoff --dir "$B" --session "$B_SESSION" --summary "Must be refused while conflicted." >"$PROOF_ROOT/receipts/conflicted-handoff.txt" 2>&1
CONFLICT_HANDOFF_EXIT=$?
set -e
if [ "$CONFLICT_HANDOFF_EXIT" -eq 0 ]; then echo "handoff unexpectedly bypassed conflict" >&2; exit 1; fi

RESOLUTION="$PROOF_ROOT/resolution.json"
printf '{"profile":"2.0.0","conflict_id":"%s","heads":["%s","%s"],"selected_event":"%s","reason":"local proof reviewed both heads","resolver":"local proof","session_id":"%s","expected_frontier":"%s"}\n' "$CONFLICT_ID" "$A_EVENT" "$B_EVENT" "$A_EVENT" "$B_SESSION" "$CONFLICT_FRONTIER" >"$RESOLUTION"
"$BIN" reconcile --dir "$B" --apply "$RESOLUTION" --expect-state "$CONFLICT_FRONTIER" --session "$B_SESSION" --json >"$PROOF_ROOT/receipts/resolution.json"
"$BIN" check --strict --dir "$B"
"$BIN" handoff --dir "$B" --session "$B_SESSION" --summary "Both clone narratives retained. Reviewed conflict resolved to clone A outcome; fresh clone should recheck private evidence." --json >"$PROOF_ROOT/receipts/resolved-handoff.json"

printf 'portable proof\n' >"$PROOF_ROOT/portable.txt"
"$BIN" evidence save --dir "$B" --file "$PROOF_ROOT/portable.txt" --task task-1 --validator local-proof --outcome passed --json >"$PROOF_ROOT/receipts/portable-evidence.json"
"$BIN" evidence declare-unavailable --dir "$B" --task task-1 --reason "private branch log intentionally not copied" --json >"$PROOF_ROOT/receipts/private-unavailable.json"
git -C "$B" add .small && git -C "$B" commit -q -m "resolve semantic conflict and evidence state"
git -C "$B" push -q origin HEAD:main

FRESH="$PROOF_ROOT/fresh"
git clone -q "$REMOTE" "$FRESH"; git_identity "$FRESH"
"$BIN" check --strict --dir "$FRESH"
"$BIN" reconstruct --dir "$FRESH" --resume --session "$B_SESSION" --limit 20 --max-bytes 32768 --json >"$PROOF_ROOT/receipts/fresh-resume.json"
set +e
"$BIN" evidence verify --dir "$FRESH" --json >"$PROOF_ROOT/receipts/fresh-evidence.json" 2>&1
EVIDENCE_EXIT=$?
set -e
if [ "$EVIDENCE_EXIT" -eq 0 ]; then echo "unavailable private evidence was not reported" >&2; exit 1; fi

# Stale state and collaborative-to-solo with active sessions both refuse.
set +e
"$BIN" mode set solo --dir "$FRESH" --session "$B_SESSION" --expect-state 0000000000000000000000000000000000000000000000000000000000000000 --reason stale >"$PROOF_ROOT/receipts/stale-mode.txt" 2>&1
STALE_EXIT=$?
set -e
test "$STALE_EXIT" -ne 0
"$BIN" mode show --dir "$FRESH" --json >"$PROOF_ROOT/fresh-mode.json"
FRESH_FRONTIER="$(json_value "$PROOF_ROOT/fresh-mode.json" frontier)"
set +e
"$BIN" mode set solo --dir "$FRESH" --session "$B_SESSION" --expect-state "$FRESH_FRONTIER" --reason "should refuse active owners" >"$PROOF_ROOT/receipts/solo-refusal.txt" 2>&1
SOLO_EXIT=$?
set -e
test "$SOLO_EXIT" -ne 0

# A supported old v1 CLI may not understand v2, but a read-only probe must not mutate it.
OLD_SMALL="${OLD_SMALL:-/opt/homebrew/bin/small}"
if [ -x "$OLD_SMALL" ] && [ "$OLD_SMALL" != "$BIN" ]; then
  BEFORE="$(state_digest "$FRESH")"
  set +e
  "$OLD_SMALL" status --dir "$FRESH" >"$PROOF_ROOT/receipts/old-cli-v2.txt" 2>&1
  OLD_EXIT=$?
  set -e
  AFTER="$(state_digest "$FRESH")"
  test "$BEFORE" = "$AFTER"
  printf 'old_cli_exit=%s\nauthoritative_digest_before=%s\nauthoritative_digest_after=%s\n' "$OLD_EXIT" "$BEFORE" "$AFTER" >>"$PROOF_ROOT/receipts/old-cli-v2.txt"
fi

git -C "$FRESH" status --porcelain >"$PROOF_ROOT/receipts/fresh-git-status.txt"
if [ -s "$PROOF_ROOT/receipts/fresh-git-status.txt" ]; then echo "fresh proof clone is dirty" >&2; exit 1; fi

printf 'proof_root=%s\nv1_merge_exit=%s\nsemantic_check_exit=%s\nconflicted_handoff_exit=%s\nevidence_exit=%s\nstale_exit=%s\nsolo_refusal_exit=%s\nfinal_frontier=%s\n' "$PROOF_ROOT" "$V1_MERGE_EXIT" "$CONFLICT_CHECK_EXIT" "$CONFLICT_HANDOFF_EXIT" "$EVIDENCE_EXIT" "$STALE_EXIT" "$SOLO_EXIT" "$FRESH_FRONTIER" | tee "$PROOF_ROOT/receipts/summary.txt"
