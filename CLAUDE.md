# Role and Repository Context

You are working in a SMALL Protocol repository.

This repository uses SMALL as a state protocol, not an execution or orchestration system.

Execution is performed by external tools.
SMALL governs durable project state.

---

# MANDATORY: SMALL Protocol Enforcement

All AI agent work in this repository MUST follow SMALL Protocol. These rules are NON-NEGOTIABLE.

Agents must interact with SMALL only via the CLI.
Manual editing of .small/*.yml files is discouraged except where explicitly stated.

---

# Authoritative Rule

The output of `small --help` is the source of truth.

If documentation or comments contradict the CLI, follow the CLI.

---

# At Session Start (REQUIRED)

Run:

`small status --json`

- Use this to understand current state.
- Do NOT assume tasks, intent, or progress.

If you need a machine-readable view of full state:

`small emit --workspace root`

---

# Planning Work (REQUIRED)

To add new work:

`small plan --add "Task title"`

- Do NOT invent tasks mentally.
- Do NOT use TodoWrite or any other task tracking.

To understand the current plan:

- Read `.small/plan.small.yml` directly if needed.
- There is NO `small plan --show` command.

---

# Progress and State Mutation (CRITICAL)

NEVER manually edit `progress.small.yml`.

Progress entries MUST be written via CLI commands.

## To log progress during work

`small progress add --task <task-id> --status in_progress --evidence "What was done"`

- Timestamps are auto-generated.
- Timestamps are guaranteed RFC3339Nano and strictly monotonic.
- You MUST NOT construct timestamps manually.

## To complete or block a task (atomic)

Completed:

`small checkpoint --task <task-id> --status completed --evidence "What was completed"`

Blocked:

`small checkpoint --task <task-id> --status blocked --evidence "Why this is blocked"`

This:

- Updates the plan
- Appends a progress entry
- Performs an atomic state transition

---

# Executing Commands (WHEN NEEDED)

Execution is optional and external.

If you must run a command:

`small apply --cmd "<command>" --task <task-id>`

- `apply` runs exactly one command.
- It records execution outcome as progress.
- It does NOT orchestrate workflows.

---

# Verification and Enforcement

Default enforcement command:

`small check`

- Runs validate, lint, and verify.
- Use this for humans, CI, and agents.

Explicit enforcement gate (still valid):

`small verify`

- CI-grade enforcement.
- Not deprecated.
- Used internally by `small check`.

---

# Inspecting State for Integrations or Loops

For structured, machine-readable state:

`small emit --check --workspace root`

Use `emit` to:

- Determine what work remains
- Inspect recent progress
- Read enforcement results
- Integrate with loops, harnesses, or CI

`emit` is read-only unless `--check` is used.

---

# At Session End or Context Risk (REQUIRED)

Before ending work or if context is running low:

`small handoff`

If completing a milestone:

`small archive`

---

# Forbidden Actions

- DO NOT use TodoWrite or any external task tracking.
- DO NOT manually edit `progress.small.yml`.
- DO NOT invent timestamps.
- DO NOT assume task state without calling `small status` or `small emit`.
- DO NOT orchestrate workflows inside SMALL.

---

# Mental Model (IMPORTANT)

- SMALL governs state, not execution.
- Execution systems come and go.
- State is durable, auditable, and resumable.

Think in terms of:

- intent
- constraints
- plan
- progress
- checkpoints
- enforcement

Not in terms of:

- steps
- loops
- retries
- orchestration

---

# Artifact Roles

- `.small/intent.small.yml` - Authoritative intent and scope
- `.small/constraints.small.yml` - Constraints and policy
- `.small/plan.small.yml` - Planned tasks and dependencies
- `.small/progress.small.yml` - Append-only audit trail (CLI-written)
- `.small/handoff.small.yml` - Resume context
- `.small/runs/` - Archived lineage

---

# Final Rule

If you are unsure how to proceed:

1. Run `small status --json`
2. Run `small emit`
3. Choose the smallest valid state mutation
4. Use `progress add` or `checkpoint`
5. Verify with `small check`

This file exists to prevent drift.
Follow it exactly.
