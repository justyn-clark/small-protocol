# Invariants (v1.0.0)

The following are non-negotiable:

- `intent` and `constraints` MUST be human-owned
- `plan` and `progress` MUST be agent-owned
- `progress` MUST be append-only and evidence-backed
- Completed tasks in `plan.small.yml` MUST have corresponding progress entries
- `handoff.small.yml` MUST include a valid `replayId`
- Schema validation MUST pass
- `small_version` MUST equal `"1.0.0"`

Violations are hard errors.

## Ownership Rules

| Artifact    | Owner  |
|-------------|--------|
| intent      | Human  |
| constraints | Human  |
| plan        | Agent  |
| progress    | Agent  |
| handoff     | System |

Human-owned artifacts define the work and its boundaries.
Agent-owned artifacts record proposed and executed work.
System-owned artifacts capture state for resumption.

## Append-Only Requirement

The `progress.small.yml` artifact is append-only.

- Entries may not be modified after creation
- Entries may not be deleted
- Each entry must include evidence of execution

## Progress Timestamp Contract

Every progress entry timestamp must meet this contract:

- RFC3339Nano format
- Includes fractional seconds
- Strictly increasing within the file

Verification errors include the entry index and offending timestamp. If you have
older logs, run:

```bash
small progress migrate
```

## Version Constraint

All artifacts must declare:

```yaml
small_version: "1.0.0"
```

Artifacts with missing or mismatched versions will fail validation.

## Schema Validation

All artifacts must validate against the authoritative schemas located at:

```text
spec/small/v1.0.0/schemas/
```

Schema violations are hard errors and will block execution.

## Evidence Gate

When a task in `plan.small.yml` is marked as `status: completed`, there MUST be a corresponding entry in `progress.small.yml` that:

1. References the same `task_id`
2. Includes a valid RFC3339Nano `timestamp` with fractional seconds
3. Contains at least one evidence field (`evidence`, `verification`, `command`, `test`, `link`, or `commit`)

This ensures that completed tasks have auditable evidence of completion.

## Strict Mode Invariants

Strict mode is opt-in (`--strict`) and extends invariant enforcement with additional safety checks.

### Strict Invariant S1: Completed or Blocked Tasks Need Evidence

When `plan.small.yml` marks a task as `completed` or `blocked`, there must be at least one progress entry with:

- matching `task_id`
- non-empty `evidence` or `notes`

Failure messages include the task id, task title, and whether evidence is missing or empty.

### Strict Invariant S2: Progress Task IDs Must Be Known (or Meta)

Each progress entry must reference a task id that exists in `plan.small.yml`, unless the id starts with `meta/`.

### Strict Invariant S3: Handoff References Must Match Plan

If `handoff.small.yml` sets `resume.current_task_id`, the referenced task id must exist in the plan.

### Strict Invariant S4: Reconciliation Marker (Optional)

When enabled, strict mode can require a `meta/reconcile-plan` progress entry if the plan was retroactively edited after progress entries were logged. This guard is optional and disabled by default until reliable plan-change detection is available.

**Example failure:**

```yaml
# plan.small.yml
tasks:
  - id: "my-task"
    title: "Do something"
    status: "completed"  # Marked as completed

# progress.small.yml
entries: []  # No corresponding entry -> FAILS verify
```

**Fix:** Add a progress entry before marking the task complete:

```yaml
# progress.small.yml
entries:
  - task_id: "my-task"
    timestamp: "2026-01-12T12:00:00.000000000Z"
    evidence: "Implemented feature X"
```

## ReplayId Requirement

The `handoff.small.yml` artifact MUST include a `replayId` field with:

- `value`: A valid SHA256 hash (64 hex characters)
- `source`: Either `"auto"` or `"manual"`

This enables session replay and continuity tracking.

**Example failure:**

```yaml
# handoff.small.yml - FAILS verify (missing replayId)
summary: "Work in progress"
resume:
  next_steps: ["Continue task"]
links: []
```

**Fix:** Run `small handoff` to generate a valid replayId:

```bash
small handoff --summary "Work in progress"
```

Or manually add:

```yaml
replayId:
  value: "a1b2c3d4e5f6..."  # 64 hex characters
  source: "manual"
```

## CI Integration

The `small verify` command enforces all invariants and is designed for CI pipelines:

```bash
# In CI workflow
small verify --ci

# Exit codes:
# 0 - All artifacts valid
# 1 - Artifacts invalid (validation or invariant failures)
# 2 - System error (missing directory, read errors)
```

Common CI failure scenarios:

1. **Forgot to update SMALL files** - Completed tasks without progress entries
2. **Missing replayId** - handoff.small.yml lacks session tracking
3. **Schema violations** - Invalid artifact structure
4. **Owner mismatch** - Wrong owner for artifact type
