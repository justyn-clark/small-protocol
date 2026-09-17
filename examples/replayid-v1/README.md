# Lab: deterministic replay ID

**Question:** Can two machines identify the same v1 run without a central
coordinator?

This completed v1 workspace shows SMALL's deterministic handoff identity. When
the run-defining intent, plan, and constraints are unchanged, the CLI derives
the same lowercase SHA-256 replay ID.

## Inspect it

From the repository root:

```bash
small check --strict --dir examples/replayid-v1 --workspace examples
small status --json --dir examples/replayid-v1
```

Open [`.small/handoff.small.yml`](.small/handoff.small.yml) and look for:

```yaml
replayId:
  value: <64 lowercase hexadecimal characters>
  source: auto
```

`source: auto` means the CLI derived the identity from canonical inputs.
`--replay-id` remains available for a validated manual override when an
external process already owns the identity.

## Why it matters

A replay ID is not an execution engine or a distributed lock. It is a stable
continuity anchor: the handoff can prove which run-defining state it summarizes
without relying on terminal history or chat memory.
