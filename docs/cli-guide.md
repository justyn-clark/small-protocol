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
| `--spec-dir <path>` | Path to spec/ directory for schemas |

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
| `--dir <path>` | Directory containing .small/ |

**What gets generated:**

- `summary` - Provided text or auto-generated
- `resume.current_task_id` - First in_progress task
- `resume.next_steps` - Titles of pending/in_progress tasks
- `links` - Empty array (populate manually if needed)

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
```
