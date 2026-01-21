# SMALL Protocol Agent Guide

This is the operating contract for any AI agent or human acting like one. Follow it exactly. If anything here conflicts with `small --help`, the CLI wins.

SMALL governs state, not execution. The CLI is the only valid way to mutate `.small/`.

Target version: SMALL CLI 1.0.0

Scope: all repos that claim SMALL compliance

---

## Zero-trust rules

- Never manually edit `.small/progress.small.yml`
- Never invent timestamps
- Never mark tasks complete with plan toggles
- Completion is only via `small checkpoint`
- Never run `small handoff` unless `small check --strict` passes
- Never assume chat memory; always read state from disk and CLI

---

## Artifact roles

- `.small/intent.small.yml` - purpose and scope
- `.small/constraints.small.yml` - hard limits and policy
- `.small/plan.small.yml` - planned tasks
- `.small/progress.small.yml` - append-only audit trail
- `.small/handoff.small.yml` - resume context
- `.small/runs/` - archived lineage

**Workspace initialization**:
- `small init` - Create new workspace from scratch
- `small start` - Repair existing workspace or ensure handoff state

---

## Agent-safe command set

### Read-only

- `small version`
- `small status --json`
- `small emit --check --workspace root`
- `small doctor` (diagnose workspace issues - read-only)
- `small run list`
- `small run diff`

### Plan and progress

- `small plan --add "Task title"`
- `small progress add --task <task-id> --status in_progress --evidence "..." --notes "..."`
- `small checkpoint --task <task-id> --status completed|blocked --evidence "..." --notes "..."`

### Execution capture

- `small apply --cmd "<command>" --task <task-id>`

### Self-healing

- `small start` (repair handoff state, ensure replayId)
- `small fix --versions` (normalize small_version formatting)
- `small doctor` (diagnose and report issues)

### Gates and lifecycle

- `small check --strict`
- `small handoff --summary "..."`
- `small archive` (milestones only)

---

## Forbidden for agents

- Any timestamp override flags (`--at`, `--after`)
- Plan-only completion (`small plan --done`)
- Manual edits to `.small/*.yml` (except intent or constraints when explicitly authorized)
- Handoff before strict check passes

---

## Session lifecycle

Always begin clean. Always end with a gated handoff.

---

### Session start

Read artifacts on disk. Do not assume chat memory.

Then run:

```bash
small status --json
small check --strict
```

If strict fails, attempt self-healing:

```bash
small start  # Repair handoff state, ensure replayId exists
small check --strict
```

If strict still fails, diagnose:

```bash
small doctor  # Read-only diagnostic report
```

Fix the workspace before any new work. Never proceed with a failing workspace.

---

## Work loop

Repeat until all tasks are completed or blocked.

---

### Refresh state

```bash
small status --json
small check --strict
```

If strict fails, fix immediately before continuing.

---

### Select task

Choose a pending task with no unmet dependencies.

If none exists, add one:

```bash
small plan --add "Clear task title"
```

---

### Start task with evidence

```bash
small progress add \
  --task <task-id> \
  --status in_progress \
  --evidence "Starting: what will be done" \
  --notes "optional"
```

---

### Execute bounded changes

For any mutating command:

```bash
small apply --cmd "<command>" --task <task-id>
```

Read-only exploration may run outside `apply`.

Any command that changes files, deps, builds, tests, or migrations must use `apply`.

---

### Complete atomically

Completed:

```bash
small checkpoint \
  --task <task-id> \
  --status completed \
  --evidence "What changed and why" \
  --notes "optional"
```

Blocked:

```bash
small checkpoint \
  --task <task-id> \
  --status blocked \
  --evidence "Why blocked and what is needed" \
  --notes "optional"
```

---

### Enforce gate

```bash
small check --strict
```

Do not continue with a failing workspace.

---

## Session end

Always gate, then hand off:

```bash
small check --strict
small handoff --summary "Current state in one paragraph"
```

---

## Milestones only

When a meaningful milestone is reached:

```bash
small check --strict
small archive
```

---

## ReplayId threading

- Treat the replayId from `small status --json` as the primary key
- All progress and handoff context must match the active replayId
- Strict check should fail if replayId is missing or mismatched
- Use `small start` to repair missing or corrupted replayId

---

## Evidence quality

Good evidence:

- Precise
- Names files or systems touched
- Explains why the change was made

Bad evidence:

- "Fixed it"
- "Build works now"

---

## Stop rules

Stop and block if:

- Intent or constraints are missing or unclear
- Secrets or external access are required
- A human decision is needed
- Strict check cannot pass after self-healing attempts

Blocked handoff example:

```bash
small checkpoint \
  --task <task-id> \
  --status blocked \
  --evidence "Need decision on X before proceeding"

small check --strict
small handoff --summary "Blocked on decision X"
```

---

## Troubleshooting

If workspace state is corrupted:

1. Run `small doctor` for diagnostic report
2. Run `small start` to repair handoff state
3. Run `small fix --versions` if small_version formatting is invalid
4. Run `small check --strict` to verify repair

If strict check still fails, stop and request human intervention.

---

## Glossary

**Agent**: An LLM or automation acting through the SMALL CLI

**Evidence**: A factual description of what changed and where

**Checkpoint**: Atomic plan plus progress update with evidence

**Strict check**: The enforcement gate before any handoff

**Self-healing**: Automatic repair of workspace state using `small start`, `small fix`, or `small doctor`

---

This contract exists to prevent drift. Follow it exactly.
