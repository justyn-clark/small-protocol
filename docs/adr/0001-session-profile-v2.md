# ADR 0001: session-capable SMALL v2 profile

Status: accepted

Date: 2026-09-13

## Context

SMALL v1.0.0 stores mutable plan, progress, and handoff projections in shared
YAML files. That format remains useful and supported for sequential work, but
its global timestamp rule and shared write targets cannot safely represent
independent writers. Adding identity and causal fields to v1 is not compatible
with its closed schemas.

## Decision

Introduce protocol profile `2.0.0`. Migration is explicit; v1 workspaces never
migrate automatically. The CLI reads both profiles with separate validators.

The default v2 policy is `solo`. `collaborative` is opt-in. Both modes use the
same immutable event storage, canonical encoding, reducer, evidence semantics,
and reconciliation gates. A mode changes writer policy only.

Tracked v2 authority consists of a stable project profile, human policy files,
immutable session descriptors, one immutable event per path, immutable receipts,
and byte-for-byte v1 imports. Generated views, active-session selection, locks,
indexes, and command logs are local cache. Plan status and handoff text are
projections; they are not independently writable truth.

Events use opaque random IDs, per-session sequence and previous-event links,
explicit observed parents, and a digest of deterministic canonical JSON with
the digest field omitted. Wall time and actor/model labels are annotations, not
ordering or authorization. Identical IDs with different content are corruption.

The reducer preserves all valid facts and emits semantic conflicts. It never
chooses a task result, mode change, takeover, or resolution by timestamp or
display order. Resolutions are new events bound to the expected frontier and all
competing heads. Strict authoritative handoff fails while conflicts remain.

Local writes use an ignored checkout lock and atomic exclusive event creation.
This is not a distributed lock. Git transports immutable paths; SMALL does not
pull, stash, stage, commit, or resolve application code. No merge driver is
required.

## Consequences

- v1 syntax and validation continue for unmigrated workspaces.
- v2 is not a transparent v1 patch and cannot be losslessly downgraded.
- Fresh clones reconstruct from tracked authority without local cache.
- Independent records normally merge as separate paths, while incompatible
  decisions remain visible after a clean Git merge.
- Disconnected machines may still act concurrently; ownership is advisory and
  stale contributions are preserved for reconciliation.
- SMALL remains independent of model calls, agent spawning, retries, scheduling,
  billing, and distributed consensus. LoopExec remains the loop governor.
