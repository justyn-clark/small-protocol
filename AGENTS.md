# AGENTS.md - SMALL Agent Contract (Authoritative)

This repository is governed by SMALL.

The `.small/` directory is state. State is authoritative.
Conversation, terminal scrollback, and chat logs are not authoritative.

If anything in this file conflicts with `small --help` or actual CLI behavior, the CLI wins.

This file is designed to be obeyed, not interpreted.

## Scope

Applies to any automation, agent, human-with-a-script, or tool that performs work in this repo.

## Non-negotiables

- NEVER manually edit any file under `.small/` (including plan, progress, handoff, logs, archive, runs).
- NEVER invent, override, or "fix" timestamps.
- NEVER reuse a task id for unrelated work.
- NEVER claim completion by changing plan status alone.
- NEVER produce a handoff unless `small check --strict` passes.
- NEVER write repo files outside `small apply`.
- NEVER assume chat history is state. Read state from disk.

Violations are contract breaches.

## Authority order

1. `small` CLI behavior and `small --help`
2. `.small/` artifacts
3. This file
4. Everything else (including chat)

## Required operating sequence

Before any work:
- `small status --json`
- `small check --strict`

If strict fails:
- Follow the CLI error output.
- Use `small fix ...` commands when provided.
- Re-run `small check --strict` until it passes.

No passing strict check means no handoff and no archive.

## Planning rule

No plan, no work.

Before any change to the repository:
- Ensure a plan task exists that describes the work.
- The task id used for execution MUST match the work being performed.

Operational bookkeeping tasks are allowed only if they are explicitly permitted by the repo rules or the CLI provides a dedicated pathway. When in doubt, add a plan task.

## Write rule (hard)

All repository modifications MUST occur via `small apply`.

This includes, but is not limited to:
- code edits
- formatting
- dependency changes
- generating files
- running tools that write output
- running tests that write snapshots or golden files

If a command can modify files, run it through `small apply`.

## Progress rule (append-only)

Progress is append-only. Do not delete, rewrite, or "clean up" progress history.

Start work on a task:
- record in_progress via the CLI

Finish work on a task:
- record completion only via `small checkpoint`

If work is cancelled or moved outside SMALL:
- record `blocked` via `small checkpoint` with clear evidence that it was intentionally cancelled and why

Example evidence must be specific:
- what happened
- where it happened (paths, commands)
- what remains (if anything)

## Determinism rule

Each step MUST be:
- bounded (one clear action)
- attributable (one task id)
- evidenced (recorded via progress and checkpoint)
- resumable (a future operator can pick up from `.small/`)

## Handoff rule

A handoff is required when a run stops and will be resumed later, or when handing work to another operator.

A handoff MUST:
- be written only after `small check --strict` passes
- summarize what changed and what state is now true
- list the next actionable task(s) or explain why none exist

## Snapshot and archive rule

Snapshot and archive are run artifacts. They should be created by the operator when a run is considered complete enough to preserve.

Do not archive a broken state. Archive only when strict check passes.

## No-network and secrets rule

Do not exfiltrate secrets.
Do not run network actions unless explicitly required and explicitly allowed by the operator and environment.

## Minimal definition of "done"

A task is "done" only when:
- the required repo changes exist
- tests and checks required by the plan have been run (as applicable)
- a `small checkpoint` records completion evidence
- `small check --strict` passes

That is the only definition of done in this repo.
