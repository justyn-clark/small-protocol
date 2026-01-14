# CLI Guide

Complete reference for the `small` command-line interface. Run all commands from your repository root.

## Where to Run Commands

**Recommended**: Run from the repository root (the directory containing `.small/`).

```bash
cd /path/to/your/project
small status
```

**Also works**: Running from inside `.small/` directory. The CLI resolves the workspace automatically.

```bash
cd /path/to/your/project/.small
small status  # Still works
```

**All commands** support the `--dir` flag to specify a different workspace:

```bash
small status --dir /path/to/other/project
```

### Git contract: what to commit

SMALL uses `.small/` in two different ways:

1) Runtime workspace (repo root)
   - Path: `./.small/`
   - Purpose: active execution state for the current run
   - Policy: do NOT commit
   - Reason: it churns every run and becomes noise

2) Published examples (committed)
   - Path: `examples/**/.small/` (and `spec/**/examples/.small/`)
   - Purpose: stable, reviewable examples of correct SMALL runs
   - Policy: DO commit
   - Reason: examples are part of the specification and documentation

In this repo, root `./.small/` is ignored, but example `.small/` folders are tracked.

### Temp directories

The following directories are ephemeral and should never be committed:

| Directory | Purpose |
|-----------|---------|
| `.tmp/` | Local scratch space for debugging (ignored) |
| `.small/archive/` | Run archives (local by default, see `small archive`) |
| OS temp (`/tmp/`, `$TMPDIR`) | Selftest and CI workspaces |

The `small selftest` command uses OS temp by default so it never touches repo state.

### Run archiving

Use `small archive` to preserve run state without committing `.small/`:

```bash
small archive
```

Archives are stored in `.small/archive/<replayId>/` with a manifest containing SHA256 hashes. Archives are local by default (ignored in .gitignore), but you may commit them in product repos if you want persistent lineage. See `small archive --help` for details.

## Live Progress Logging

Mutating commands append progress entries immediately (no end-of-run batching). Each
entry uses RFC3339Nano timestamps with fractional seconds and strictly increasing
order. Use `small progress migrate` to repair older logs.

**Commands that write progress:**

| Command | Progress entry behavior |
|---------|--------------------------|
| `small init` | `task_id: init`, `status: completed`, evidence about workspace creation |
| `small plan --add` | `task_id: <new task>`, `status: pending`, evidence about the new task |
| `small plan --done` | `task_id: <task>`, `status: completed`, evidence of completion |
| `small plan --pending` | `task_id: <task>`, `status: pending`, evidence of status change |
| `small plan --blocked` | `task_id: <task>`, `status: blocked`, evidence of status change |
| `small plan --depends` | `task_id: <task>`, evidence of dependency update |
| `small apply` | start entry `in_progress`, end entry `completed` or `blocked` |
| `small apply --dry-run` | `status: pending`, evidence of dry-run |
| `small reset` | `task_id: reset`, `status: completed`, evidence of reset |

`small handoff`, `small status`, `small doctor`, and `small verify` are read-only
and do not append progress.

## Commands

### small version

Print CLI version and supported protocol version.

```bash
small version
```

Output:
```
small v1.6.0 (SMALL Protocol v1.0.0)
```

No flags.

### small init

Initialize a new `.small/` directory with all five canonical artifacts.

```bash
small init --intent "Build a user authentication system"
```

**Flags:**

| Flag | Description |
|------|-------------|
| `--intent <string>` | Seed the intent field in intent.small.yml |
| `--force` | Overwrite existing .small/ directory |
| `--dir <path>` | Target directory (default: current) |

**What gets created:**

```
.small/
├── intent.small.yml       # Populated with --intent value
├── constraints.small.yml  # Template with example constraints
├── plan.small.yml         # Empty task list
├── progress.small.yml     # Empty entries list
└── handoff.small.yml      # Initial handoff state
```

`small init` also writes `.small/workspace.small.yml` containing workspace metadata (`small_version` and `kind`). Root workspaces use `kind: repo-root`, while example workspaces under `examples/**` retain `kind: examples`. Keep the repository root `.small/` directory local (do not commit it); shared sample workspaces live under `examples/**/.small/` so their metadata stays in version control.

**Common errors:**

| Error | Cause | Resolution |
|-------|-------|------------|
| `.small/ already exists` | Directory exists | Use `--force` to overwrite |
| `permission denied` | No write access | Check directory permissions |

### small validate

Validate all canonical artifacts against JSON schemas.

```bash
small validate
```

**Flags:**

| Flag | Description |
|------|-------------|
| `--dir <path>` | Directory containing .small/ |


**Schema resolution order:**

1. `--spec-dir` flag if provided
2. `SMALL_SPEC_DIR` environment variable
3. On-disk `spec/` directory in workspace
4. Embedded v1.0.0 schemas (fallback)

**What gets checked:**

- All five artifacts exist
- Each artifact parses as valid YAML
- Each artifact matches its JSON schema
- Required fields are present
- Field types are correct

**Common errors:**

| Error | Cause | Resolution |
|-------|-------|------------|
| `.small/ directory not found` | Wrong directory | Run from repo root or use --dir |
| `missing required field: intent` | Field not in YAML | Add the required field |
| `small_version must be "1.0.0"` | Wrong version | Use exact string "1.0.0" |
| `invalid type for field X` | Wrong YAML type | Check schema for expected type |

### small lint

Check invariant violations beyond schema validation.

```bash
small lint
```

**Flags:**

| Flag | Description |
|------|-------------|
| `--strict` | Enable strict mode (includes secret detection) |
| `--dir <path>` | Directory containing .small/ |
| `--spec-dir <path>` | Path to spec/ directory |

**What gets checked:**

- Version consistency across all artifacts
- Ownership rules (human owns intent/constraints, agent owns plan/progress)
- Evidence requirement in progress entries
- Secret detection (with `--strict`)

**Difference from validate:**

- `validate` checks structure (schema compliance)
- `lint` checks behavior (invariant compliance)

Both must pass for a valid workspace.

**Common errors:**

| Error | Cause | Resolution |
|-------|-------|------------|
| `owner must be "human"` | Wrong owner in intent/constraints | Set owner: "human" |
| `owner must be "agent"` | Wrong owner in plan/progress | Set owner: "agent" |
| `progress entry missing evidence` | No evidence field | Add evidence, command, commit, link, test, or verification |
| `potential secret detected` (strict) | Possible credential in artifact | Remove or redact the secret |

### small plan

Create or update tasks in plan.small.yml.

```bash
small plan --add "Implement user registration"
```

**Flags:**

| Flag | Description |
|------|-------------|
| `--add <string>` | Add a new task with given title |
| `--done <task-id>` | Mark task as completed |
| `--pending <task-id>` | Mark task as pending |
| `--blocked <task-id>` | Mark task as blocked |
| `--depends <id>:<dep-id>` | Add dependency (id depends on dep-id) |
| `--reset` | Reset plan to template (destructive) |
| `--yes` | Confirm destructive operations |
| `--dir <path>` | Directory containing .small/ |

**Task IDs:**

Task IDs are auto-generated as `task-1`, `task-2`, etc. The CLI handles ID assignment.

**Examples:**

```bash
# Add tasks
small plan --add "Set up database schema"
small plan --add "Implement API endpoints"
small plan --add "Write integration tests"

# Add dependency (task-3 depends on task-2)
small plan --depends "task-3:task-2"

# Update status
small plan --done task-1
small plan --blocked task-2

# Reset (requires confirmation)
small plan --reset --yes
```

**Common errors:**

| Error | Cause | Resolution |
|-------|-------|------------|
| `.small/ not found` | Workspace not initialized | Run `small init` first |
| `task not found: task-99` | Invalid task ID | Check plan.small.yml for valid IDs |
| `--reset requires --yes` | Safety check | Add --yes to confirm |

### small status

Show project state summary.

```bash
small status
```

**Flags:**

| Flag | Description |
|------|-------------|
| `--json` | Output in JSON format |
| `--recent <n>` | Number of recent progress entries (default: 5) |
| `--tasks <n>` | Number of next actionable tasks (default: 3) |
| `--dir <path>` | Directory containing .small/ |

**Text output:**

```
small v1.6.0

Artifacts:
  intent.small.yml [x]
  constraints.small.yml [x]
  plan.small.yml [x]
  progress.small.yml [x]
  handoff.small.yml [x]

Plan: 3 tasks
  pending: 2
  completed: 1
Next actionable: [task-2, task-3]

Recent progress (2 entries):
  [2025-01-04 10:00:00] task-1: completed
  [2025-01-04 09:30:00] task-1: in_progress

Last handoff: 2025-01-04T10:05:00Z
```

**JSON output:**

```bash
small status --json
```

Returns structured data for programmatic use.

**Common errors:**

| Error | Cause | Resolution |
|-------|-------|------------|
| `.small/ not found` | Workspace not initialized | Run `small init` first |

### small apply

Execute a command and record results in progress.small.yml.

```bash
small apply --cmd "npm test" --task task-1
```

**Flags:**

| Flag | Description |
|------|-------------|
| `--cmd <string>` | Shell command to execute |
| `--task <task-id>` | Associate with specific task |
| `--dry-run` | Record intent without executing |
| `--handoff` | Generate handoff after success |
| `--dir <path>` | Directory containing .small/ |

**Execution flow:**

1. Records start entry (status: in_progress)
2. Executes command via `sh -lc "<cmd>"`
3. Records completion entry with exit code
4. If exit code 0: status completed
5. If exit code != 0: status blocked

**Dry-run mode:**

Without `--cmd`, or with `--dry-run`, records intent without execution:

```bash
small apply --task task-1 --dry-run
```

**With handoff:**

Generate handoff automatically after successful execution:

```bash
small apply --cmd "npm test" --task task-1 --handoff
```

**Common errors:**

| Error | Cause | Resolution |
|-------|-------|------------|
| `.small/ not found` | Workspace not initialized | Run `small init` first |
| `command failed with exit code 1` | Command returned error | Check command output, fix issue |

### small handoff

Generate or update handoff.small.yml from current plan state.

```bash
small handoff --summary "Completed authentication module"
```

**Flags:**

| Flag | Description |
|------|-------------|
| `--summary <string>` | Custom summary text |
| `--replay-id <string>` | Manual replayId override (64 hex chars, normalized to lowercase) |
| `--dir <path>` | Directory containing .small/ |

**What gets generated:**

- `summary` - Provided text or auto-generated
- `resume.current_task_id` - First in_progress task
- `resume.next_steps` - Titles of pending/in_progress tasks
- `links` - Empty array (populate manually if needed)
- `replayId` - Deterministic identifier for the run (always included)

**ReplayId:**

ReplayId is the stable identifier for a SMALL run. The CLI emits replayId automatically by hashing the run-defining artifacts (intent + plan + optional constraints). The hash is deterministic: same inputs produce the same replayId across machines.

```yaml
replayId:
  value: "a1b2c3d4..."  # 64 hex chars (SHA256), stored in lowercase
  source: "auto"        # "auto" when generated, "manual" when provided
```

If you supply `--replay-id`, the CLI will validate the value (must be 64 hex characters) and normalize it to lowercase before storing, marking `source: "manual"`. This means you can provide uppercase or lowercase hex, and the result will always be lowercase.

```bash
# Auto-generated (default)
small handoff

# Manual override
small handoff --replay-id a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2
```

**When to handoff:**

- End of work session
- Blocked on human input
- Context limit approaching
- Task milestone completed
- Unrecoverable error

**Common errors:**

| Error | Cause | Resolution |
|-------|-------|------------|
| `.small/ not found` | Workspace not initialized | Run `small init` first |
| `no tasks in plan` | Empty plan | Add tasks with `small plan --add` |
| `invalid replayId format` | Manual replayId not 64 hex | Provide a valid 64-character hex string |

### small reset

Reset the workspace for a new run while preserving audit history.

```bash
small reset --yes
```

**Flags:**

| Flag | Description |
|------|-------------|
| `--yes`, `-y` | Non-interactive mode (skip confirmation) |
| `--keep-intent` | Preserve intent.small.yml |
| `--dir <path>` | Directory containing .small/ |

**What gets reset (ephemeral files):**

- `intent.small.yml` (unless `--keep-intent`)
- `plan.small.yml`
- `handoff.small.yml`

**What gets preserved (audit files):**

- `progress.small.yml` (append-only audit trail)
- `constraints.small.yml` (human-owned)

**When to reset:**

- Starting a new work run/session
- After completing a major milestone
- When artifacts have become stale

**Common errors:**

| Error | Cause | Resolution |
|-------|-------|------------|
| `.small/ does not exist` | Workspace not initialized | Run `small init` first |
| `Reset cancelled` | User declined confirmation | Use `--yes` to skip confirmation |

### small progress migrate

Rewrite progress timestamps to the strict contract (RFC3339Nano, fractional seconds,
strictly increasing). This command is explicit and only runs when invoked.

```bash
small progress migrate
```

**Flags:**

| Flag | Description |
|------|-------------|
| `--dir <path>` | Directory containing .small/ |
| `--workspace <scope>` | Workspace scope (`root`, `examples`, or `any`) |

**Behavior:**

- Parses every entry timestamp
- Normalizes to UTC RFC3339Nano with fractional seconds
- Adds nanosecond offsets when entries collide
- Fails fast on unparseable timestamps (no rewrite)

### small verify

CI and local enforcement gate for SMALL artifacts.

```bash
small verify
```

**Flags:**

| Flag | Description |
|------|-------------|
| `--strict` | Enable strict mode (secrets, insecure links) |
| `--ci` | CI mode (minimal output, just errors) |
| `--dir <path>` | Directory containing .small/ |
| `--workspace <scope>` | Workspace scope (`root`, `examples`, or `any`; default `root`) |

**Exit codes:**

| Code | Meaning |
|------|---------|
| 0 | All artifacts valid |
| 1 | Artifacts invalid (validation or invariant failures) |
| 2 | System error (missing directory, read errors) |

**What gets checked:**

- All required files exist
- Schema validation of all artifacts
- Invariant enforcement (ownership, required fields)
- Progress timestamps must be RFC3339Nano with fractional seconds and strict ordering
- Completed plan tasks require at least one progress entry referencing the task before verify passes
- ReplayId format validation (if present)

**CI integration example:**

```yaml
# GitHub Actions
- name: Verify SMALL artifacts
  run: small verify --ci --strict
```

**Common errors:**

| Error | Cause | Resolution |
|-------|-------|------------|
| `Missing required files` | Files don't exist | Run `small init` |
| `Schema validation failed` | Invalid structure | Fix the schema violation |
| `Invariant violation` | Protocol rule broken | Fix the invariant |
| `progress entries missing for completed plan tasks: <task ids>` | Completed task lacks a corresponding progress entry | Record at least one progress entry referencing every completed task before re-running verify |

#### Workspace metadata

`small verify` loads `.small/workspace.small.yml` and expects `kind` to be either `repo-root` or `examples`. Example workspaces under `examples/**` keep their own `.small/` directories with `kind: examples`; the repository root uses `kind: repo-root`. Any other value produces `Workspace validation failed: invalid workspace kind "<value>"; valid kinds: ["repo-root", "examples"]`, so regenerate the metadata (for example with `small init`) or set the kind explicitly to a supported value. Keep the root `.small/` directory local (do not commit it); the `examples/**/.small/` directories are the only `.small` artifacts that stay in source control.

### small selftest

Run a built-in self-test to verify CLI functionality in an isolated workspace.

```bash
small selftest
```

**Flags:**

| Flag | Description |
|------|-------------|
| `--keep` | Do not delete temp directory after test |
| `--dir <path>` | Run selftest in explicit directory (default: OS temp) |
| `--workspace <scope>` | Workspace scope for verify step (default: any) |

**Steps performed:**

1. `init` - Creates a selftest workspace
2. `plan --add` - Adds a test task
3. `plan --done` - Marks the task complete
4. `apply --dry-run` - Records a dry-run
5. `handoff` - Generates handoff with replayId
6. `verify` - Validates the workspace

**Example output:**

```
SMALL Selftest
==============
Workspace: /var/folders/.../small-selftest-123456

[1/6] init... OK
[2/6] plan --add... OK
[3/6] plan --done... OK
[4/6] apply --dry-run... OK
[5/6] handoff... OK
[6/6] verify... OK

All selftest steps passed!
```

**When to use selftest:**

- Day-1 verification after installing the CLI
- Debugging CLI issues
- CI smoke test
- Verifying a new release works correctly

**Common errors:**

| Error | Cause | Resolution |
|-------|-------|------------|
| `selftest failed at step "verify"` | Workspace invalid | Check error details, may indicate CLI bug |
| `failed to create temp directory` | Permission issue | Check OS temp permissions or use --dir |

### small archive

Archive the current run state for lineage retention without committing `.small/`.

```bash
small archive
```

**Flags:**

| Flag | Description |
|------|-------------|
| `--dir <path>` | Directory containing .small/ |
| `--out <path>` | Output directory (default: .small/archive/<replayId>/) |
| `--include <files>` | Files to include (default: all canonical artifacts) |

**What gets archived:**

- All canonical SMALL artifacts (intent, constraints, plan, progress, handoff)
- workspace.small.yml
- A manifest (`archive.small.yml`) with SHA256 hashes for integrity verification

**Manifest contents:**

```yaml
small_version: "1.0.0"
archived_at: "2025-01-15T10:30:00.123456789Z"
source_dir: "/path/to/project"
replayId: "a1b2c3d4..."
files:
  - name: "intent.small.yml"
    sha256: "abc123..."
  - name: "plan.small.yml"
    sha256: "def456..."
```

**Requirements:**

- `handoff.small.yml` must have a valid replayId (run `small handoff` first)

**When to archive:**

- Before starting a new run (to preserve the previous run's state)
- At project milestones
- Before destructive operations
- For audit/compliance requirements

**Common errors:**

| Error | Cause | Resolution |
|-------|-------|------------|
| `missing replayId` | handoff lacks session ID | Run `small handoff` first |
| `.small/ does not exist` | Workspace not initialized | Run `small init` first |

**Storage policy:**

Archives are stored in `.small/archive/<replayId>/` by default and are ignored in `.gitignore`. You may:

1. Keep archives local (default) for debugging/reference
2. Commit archives in product repos for persistent lineage
3. Copy archives to external storage for compliance

### small doctor

Diagnose workspace issues and suggest fixes. This command is **read-only** and never mutates state.

```bash
small doctor
```

**Flags:**

| Flag | Description |
|------|-------------|
| `--dir <path>` | Directory to diagnose |

**What gets diagnosed:**

- Missing or malformed .small files
- Schema violations
- Invariant violations
- Version consistency
- Run state analysis (task status distribution)
- Progress entry count
- Handoff next steps

**Example output:**

```
SMALL Doctor Report
===================

[OK] Workspace: .small/ directory exists
[OK] Files: All required files present
[OK] Schema: All artifacts pass schema validation
[OK] Invariant: All protocol invariants satisfied
[OK] Run State: Tasks: 5 total, 2 completed, 1 in_progress, 2 pending, 0 blocked
     -> Ready to continue current task
[OK] Progress: 10 progress entries recorded

Summary: All checks passed!
```

**When to use doctor:**

- First time setting up a workspace
- When you encounter unexpected errors
- Before starting a new session
- Debugging CI failures

**Common output states:**

| Status | Meaning |
|--------|---------|
| `[OK]` | Check passed |
| `[WARN]` | Non-critical issue, review suggestion |
| `[ERROR]` | Critical issue, must fix |

## Error Resolution Guide

### "No .small/ directory found"

The CLI cannot find the workspace.

**Resolution:**

1. Run from repository root: `cd /path/to/project`
2. Or initialize: `small init --intent "..."`
3. Or specify path: `small status --dir /path/to/project`

### "Schema validation failed"

An artifact doesn't match the expected structure.

**Resolution:**

1. Check the error message for the specific field
2. Compare against examples in [Getting Started](getting-started.md)
3. Ensure `small_version: "1.0.0"` (quoted string)
4. Ensure required fields are present

### "Running inside .small/ directory"

Not an error. The CLI handles this automatically and resolves to the parent workspace. This is informational.

### "Cannot find schemas"

The CLI cannot locate JSON schemas for validation.

**Resolution (in order of preference):**

1. Use embedded schemas (automatic, no action needed)
2. Set environment variable: `export SMALL_SPEC_DIR=/path/to/spec`
3. Use flag: `small validate --spec-dir /path/to/spec`

### "Invariant violation"

A behavioral rule was broken.

**Resolution:**

1. Read the violation message
2. Check [Invariants](invariants.md) for the rule
3. Fix the artifact
4. Re-run `small lint`

### "Permission denied"

File system access issue.

**Resolution:**

1. Check directory ownership
2. Check file permissions
3. Ensure you have write access to `.small/`

## Command Cheat Sheet

```bash
# Initialize
small init --intent "Project description"

# Validate
small validate
small lint
small lint --strict

# Plan management
small plan --add "Task title"
small plan --done task-1
small plan --blocked task-2
small plan --depends "task-2:task-1"

# Status
small status
small status --json

# Execute
small apply --cmd "npm test" --task task-1
small apply --cmd "make build" --task task-2 --handoff

# Handoff
small handoff --summary "Session complete"
small handoff --replay-id abc123...  # Manual replayId override

# New run
small reset --yes           # Reset ephemeral files
small reset --keep-intent   # Keep intent, reset others
small progress migrate      # Normalize progress timestamps

# CI/Verification
small verify                # Exit 0 valid, 1 invalid, 2 error
small verify --ci --strict  # CI mode with strict checks
small selftest              # Built-in CLI self-test
small selftest --keep       # Keep temp workspace for inspection

# Diagnosis
small doctor                # Read-only workspace diagnosis

# Archiving
small archive               # Archive current run to .small/archive/
small archive --out ./backup  # Archive to custom directory
```
