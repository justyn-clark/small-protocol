# Getting Started with SMALL

This guide answers the most common beginner questions and walks you through your first SMALL-managed project.

## Do I Manually Type These Files?

No. The `small init` command creates all five canonical artifacts for you with valid starter content.

```bash
small init --intent "Build a REST API for user management"
```

After initialization, you only need to edit two files:

- `intent.small.yml` - Define your project goal and scope
- `constraints.small.yml` - Define rules the agent must follow

The agent handles the rest. You never manually create or edit `plan.small.yml`, `progress.small.yml`, or `handoff.small.yml`.

## Who Owns What?

SMALL enforces strict ownership to prevent confusion between human intent and agent execution.

| Artifact | Owner | Who Edits | When |
|----------|-------|-----------|------|
| `intent.small.yml` | Human | You | Once at project start, rarely after |
| `constraints.small.yml` | Human | You | Once at project start, rarely after |
| `plan.small.yml` | Agent | Agent | As work is planned |
| `progress.small.yml` | Agent | Agent | As work is executed |
| `handoff.small.yml` | System | Agent | At session boundaries |

**The rule is simple**: humans define what and why; agents define how and record that they did it.

## Minimal File Examples

Each file below is valid v1.0.0 SMALL. These are the minimal required fields.

### intent.small.yml (Human-Owned)

```yaml
small_version: "1.0.0"
owner: "human"
intent: "Build a REST API for user management"
scope:
  include:
    - "src/"
    - "api/"
  exclude:
    - "node_modules/"
    - ".git/"
success_criteria:
  - "All endpoints return valid JSON"
  - "Authentication works for protected routes"
```

### constraints.small.yml (Human-Owned)

```yaml
small_version: "1.0.0"
owner: "human"
constraints:
  - id: "no-secrets"
    rule: "No secrets or credentials in code or artifacts"
    severity: "error"
  - id: "no-breaking-changes"
    rule: "Do not modify existing public API signatures"
    severity: "error"
```

### plan.small.yml (Agent-Owned)

```yaml
small_version: "1.0.0"
owner: "agent"
tasks:
  - id: "task-1"
    title: "Set up project structure"
    status: "pending"
```

### progress.small.yml (Agent-Owned)

```yaml
small_version: "1.0.0"
owner: "agent"
entries:
  - task_id: "task-1"
    timestamp: "2025-01-04T10:00:00.000000000Z"
    status: "completed"
    evidence: "Created src/ directory structure"
    commit: "abc1234"
```

### handoff.small.yml (System-Owned)

```yaml
small_version: "1.0.0"
owner: "agent"
summary: "Project initialized. Ready to implement endpoints."
resume:
  current_task_id: "task-2"
  next_steps:
    - "Implement user creation endpoint"
    - "Add input validation"
links: []
```

## The Recommended Workflow

Follow this loop for any SMALL-managed project:

### Step 1: Initialize

```bash
small init --intent "Your project description"
```

This creates `.small/` with all five artifacts populated with starter content.

### Step 2: Human Fills Intent and Constraints

Open `intent.small.yml` and define:
- What you're building (the `intent` field)
- What files are in scope (`scope.include` and `scope.exclude`)
- How you'll know it's done (`success_criteria`)

Open `constraints.small.yml` and define:
- Rules the agent must never violate
- Each constraint needs an `id`, `rule`, and `severity`

### Step 3: Validate Your Edits

```bash
small validate
```

Fix any schema errors before proceeding.

### Step 4: Agent Generates Plan

The agent reads `.small/` and creates tasks in `plan.small.yml`. As a human, you can also add tasks manually:

```bash
small plan --add "Implement authentication middleware"
```

### Step 5: Agent Executes and Logs

The agent works through tasks, recording progress:

```bash
small apply --cmd "npm test" --task task-1
```

Each execution appends entries to `progress.small.yml` with evidence.

### Step 6: Agent Hands Off

When stopping work (end of session, context limit, or task completion):

```bash
small handoff --summary "Completed auth middleware, tests passing"
```

This generates `handoff.small.yml` for the next session to resume from.

### Step 7: Resume

A new session (same agent or different) reads `handoff.small.yml` first to understand current state and next steps.

## Validating the Workspace

Always validate before claiming work is complete:

```bash
# Check schemas
small validate

# Check invariants (ownership, evidence, version)
small lint

# Check with strict mode (includes secret detection)
small lint --strict
```

All three must pass for a valid SMALL workspace.

## Checking Status

Get a summary of the current project state:

```bash
small status
```

This shows:
- Which artifacts exist
- Task counts by status
- Recent progress entries
- Last handoff timestamp

For machine-readable output:

```bash
small status --json
```

## Common First-Time Mistakes

**Editing agent-owned files manually**: Don't edit `plan.small.yml`, `progress.small.yml`, or `handoff.small.yml` by hand. Use CLI commands or let the agent manage them.

**Forgetting to validate**: Always run `small validate` after editing human-owned files. Schema errors block all progress.

**Missing evidence in progress**: Every progress entry needs at least one evidence field (`evidence`, `command`, `commit`, `link`, `test`, or `verification`).

**Wrong version string**: The version must be the string `"1.0.0"`, not the number `1.0.0`. YAML treats unquoted numbers differently.

## Next Steps

- [CLI Guide](cli-guide.md) - Detailed command reference with error handling
- [Agent Operating Contract](agent-operating-contract.md) - Rules for AI agents using SMALL
- [Invariants](invariants.md) - Non-negotiable protocol rules
