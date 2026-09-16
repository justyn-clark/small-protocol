# Multi-machine and multi-session operation

This note describes the implemented SMALL 2.0.0 session profile. The v1.0.0
five-file profile remains single-writer and is not silently upgraded.

## Product decision

Solo operation is the default. It permits one active writing session in a
checkout and requires an explicit close, linked resume, or attributable
takeover. Collaborative operation is opt-in with:

```bash
small mode show --json
small mode set collaborative --expect-state <frontier> --reason "why" --session <id>
```

Both modes use the same immutable event store, reducer, evidence rules, and
semantic conflict detection. Mode changes writer policy, not truth semantics.
There is no distributed lease: disconnected clones cannot know whether another
writer is active.

## What Git can and cannot do

Each session writes unique descriptor and event paths, so independent appends
normally merge without a shared-tail text conflict. Git still transports bytes;
the SMALL reducer determines whether combined claims agree. Incompatible task
outcomes, divergent definitions or modes, competing ownership, writer-chain
forks, missing parents, and altered immutable IDs fail validation or surface as
semantic conflicts.

`small check --strict` and authoritative handoff refuse unresolved conflicts.
Use `small reconcile --preview --json`, review every competing head, then apply
a resolution file containing the conflict id, all heads, selected event,
resolver, reason, and exact expected frontier. The losing records remain.

SMALL does not merge application source, stash dirty changes, install a Git
driver, run hooks, or select a winner by timestamp.

## Identity and ordering

- Project, session, event, task, and receipt IDs are separate opaque identities.
- Imported aliases such as `task-1` remain readable but are not identity.
- Per-session sequence plus `previous_event` establishes writer order.
- Explicit parents establish cross-session causality.
- Wall time is annotation only and may be skewed or move backward.
- `replayId` remains v1 lineage terminology; a v2 frontier is a freshness token,
  not a session or task id.

## Fresh-clone recovery

Local active-session selection and locks live under ignored `.small-cache/` and
are intentionally absent in a clone. Pass an explicit prior session to inspect,
then create a fresh linked writer rather than reusing its writable ID:

```bash
small reconstruct --resume --session <prior-session> --limit 50 --max-bytes 65536 --json
small session start --parent-session <prior-session> --from-handoff <event-id>
```

Portable evidence is tracked by content digest. Private or missing evidence is
reported unavailable; it is never treated as verified merely because an old
task said it passed.
