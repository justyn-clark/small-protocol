# Codex Adapter Guidance (Stub)

This document describes how to use Codex with a SMALL-governed repository.

It does not override AGENTS.md.
If this document conflicts with AGENTS.md or SMALL CLI behavior, AGENTS.md and the CLI win.

## Status

This is a placeholder adapter guide.
It exists to keep the repository vendor-neutral and to support future Codex integrations.

## Required behavior

Codex must comply with AGENTS.md.

Key reminders:
- Do not edit `.small/` directly.
- Do not write repo files except via `small apply`.
- Progress is append-only.
- Completion is only via `small checkpoint`.
- `small check --strict` must pass before handoff or archive.

## Standard loop (manual or tool-driven)

Read state:
- `small status --json`
- `small check --strict`

Plan:
- ensure a task exists for the next change

Execute (writes only via apply):
- `small apply --cmd "<command>" --task task-N`

Record progress:
- use progress add and checkpoint via the CLI

Validate:
- `small check --strict`

Stop safely:
- `small handoff --summary "<what changed and what is true now>"`

Optionally preserve the run:
- snapshot and archive only when strict passes

## Future integration notes (non-authoritative)

A Codex integration should:
- treat `.small/` as the only durable state interface
- execute bounded steps
- record progress and checkpoints per task
- fail fast on strict violations
- prefer deterministic, replayable commands
