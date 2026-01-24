# docs/agents/claude.md - Claude Adapter Guidance

This document describes how to use Claude with a SMALL-governed repository.

It does not override AGENTS.md.
If this document conflicts with AGENTS.md or SMALL CLI behavior, AGENTS.md and the CLI win.

## Required behavior

Claude must comply with AGENTS.md.

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

## Handling strict failures

When strict fails, do not guess.
Follow the CLI output and run the recommended fix commands.
Re-run `small check --strict` until it passes.

## Git operations

Git commit and push are allowed as normal developer actions.
They are not required to be executed through SMALL unless the repository or operator explicitly requires that.

If a git operation was attempted through apply and failed, record it as blocked with a checkpoint and proceed manually if desired.

## Evidence quality

Evidence should be specific:
- files changed
- commands executed
- tests run and results
- what remains and why
