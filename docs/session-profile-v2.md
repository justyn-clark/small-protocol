# SMALL 2.0.0 session profile

The normative contract is
[spec/small/v2.0.0/SPEC.md](https://github.com/justyn-clark/small-protocol/blob/main/spec/small/v2.0.0/SPEC.md)
and the design decision is
[ADR 0001](https://github.com/justyn-clark/small-protocol/blob/main/docs/adr/0001-session-profile-v2.md).

## Layout and authority

| Path | Tracked | Authority |
|---|---:|---|
| `.small/profile.json` | yes | Project/profile identity and initial mode |
| `.small/policy/*.small.yml` | yes | Human-owned intent and constraints |
| `.small/sessions/<id>.json` | yes | Immutable writer descriptor |
| `.small/events/<session>/<id>.json` | yes | Immutable authoritative history |
| `.small/receipts/sha256/<digest>.json` | yes | Typed evidence receipt |
| `.small/evidence/blobs/sha256/<digest>` | yes when saved | Portable evidence bytes |
| `.small/imports/v1/<id>/` | yes | Original bytes and migration manifest |
| `.small-cache/` | no | Local locks, selection, transactions, command logs |
| `.small-runs/` | no | Full-state local snapshots |

Generated plan/status/handoff views are reducer projections. They are not a
second tracked authority.

## Lifecycle and evidence

A child command outcome, a task lifecycle status, and verified acceptance are
three different facts. `small apply` records a typed `cli_captured` receipt and
leaves task acceptance unchanged. `small checkpoint --evidence ...` is the
explicit task acceptance boundary. It atomically publishes the transition and
receipt. Saved files use `small evidence save`; `small evidence verify` detects
missing bytes, traversal, symlinks, digest changes, and size changes.

Typical solo lifecycle after migration:

```bash
small session start --label implementation --json
small plan --add "Implement feature" --session <session-id>
small progress add --task task-2 --status in_progress --session <session-id>
small apply --task task-2 --session <session-id> --cmd "go test ./..."
small checkpoint --task task-2 --status completed --evidence "tests reviewed" --session <session-id>
small session close --summary "Implemented feature. Next: review deployment." --session <session-id>
```

## Migration

Migration is never automatic. Preview is read-only and needs a shared namespace
so independent clones of the same baseline derive the same identities:

```bash
small migrate --to 2.0.0 --preview --namespace acme/project-baseline --json > /tmp/migration.json
small migrate --apply /tmp/migration.json --expect-state <expected_input_digest> --json
```

Apply re-hashes every legacy input, stages and strictly validates the candidate,
archives every original byte under the import id, then swaps state through a
recoverable journal. Changed inputs fail stale. Reapplying the same plan is an
idempotent success. `small migrate --recover` completes an interrupted swap.
The local legacy backup is retained under `.small-cache/migrations/`.

After migration, legacy root plan/progress/handoff files exist only inside the
immutable import archive. v1 writers are supported on unmigrated workspaces.
Supported v1 binaries may fail or report no v1 artifacts on v2 because the five
root files are absent; they must not be used to write v2 state. Arbitrary historical
binaries that ignored versions cannot be guaranteed safe.

## Limits

SMALL provides cooperative locking only within one checkout. It does not offer
distributed consensus, source-code conflict resolution, authorization from
actor/model labels, retry scheduling, model routing, or proof that two passing
branch receipts imply the integrated application passes. Re-run integrated
acceptance after merging source changes.

LoopExec or another governor may call one SMALL command at a time. SMALL records
the call and its result; it does not become a second retry or agent scheduler.

## Release measurements

The gated reducer performance fixture records elapsed time and Go allocation
totals without turning wall-clock values into a pass threshold. On the 2026-09-13
release candidate it reduced 10,000 events in 40.814917 ms with 4,844,848 bytes
of allocation delta (22,143,600 total allocation), and 100,000 events in
559.053959 ms with 58,078,800 bytes of allocation delta (212,976,096 total
allocation). Resume output remains separately bounded by `--limit` and
`--max-bytes`; truncation is explicit.

`scripts/proof-multi-session.sh` also writes `receipts/ceremony.txt`. It compares
one v1 task-add action with one v2 task-add action after the one-time session
start, recording CLI invocations, tracked files dirtied, bounded resume bytes,
and whether the pending obligation was recovered. This is a model-free ceremony
measurement only; it makes no token-savings or cross-model superiority claim.
The 2026-09-13 candidate measured one task-action invocation in both profiles:
v1 dirtied three tracked state files and produced an 815-byte task resume; v2,
after one one-time session-start invocation, dirtied one immutable event path and
produced a 10,783-byte bounded resume from the migrated multi-session fixture.
Both outputs recovered the named pending obligation. The size difference reflects
different fixture/history content and is not a product comparison.

V2 run snapshots and archives preserve the entire authoritative `.small` tree,
including import originals and receipts. Snapshot verification covers every
regular file. Checkout refuses a tampered snapshot and requires `--force` to
replace a differing live tree; archive refuses partial `--include` selections.
