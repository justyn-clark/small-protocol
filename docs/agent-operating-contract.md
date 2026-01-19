# Agent Operating Contract

This document defines the behavioral requirements for AI agents (Claude, GPT, or any LLM) operating within a SMALL-managed workspace.

These rules are non-negotiable. Violating them breaks the protocol's guarantees.

## Entry Protocol

### Always Read .small/ First

Before taking any action in a SMALL-managed repository, read the artifacts in this order:

1. `handoff.small.yml` - Current state and next steps
2. `intent.small.yml` - What the work is
3. `constraints.small.yml` - What must not be violated
4. `plan.small.yml` - Current task breakdown
5. `progress.small.yml` - What has been done

Do not skip this step. Do not assume you know the project state from conversation history.

### Verify the Workspace

After reading, run validation:

```bash
small validate
small lint
```

If either fails, fix the issues before proceeding. Do not proceed with invalid artifacts.

## Ownership Rules

### Never Edit Human-Owned Files

The following files are human-owned and must never be modified by an agent:

- `intent.small.yml`
- `constraints.small.yml`

If these files are missing required information, ask the human to provide it. Do not guess or fill in defaults.

### Only Append to Progress

The `progress.small.yml` file is append-only:

- Add new entries at the end of the `entries` array
- Never modify existing entries
- Never delete entries
- Every entry must include evidence of execution

Valid evidence fields (at least one required):
- `evidence` - Description of what happened
- `command` - Shell command that was executed
- `commit` - Git commit hash
- `link` - URL to external evidence
- `test` - Test execution details
- `verification` - Verification performed

### Handoff Is Required When Stopping

Before ending a session for any reason, generate a handoff:

```bash
small handoff --summary "Description of current state"
```

Reasons to handoff:
- Session ending (time, tokens, or context limit)
- Blocked on human input
- Task completion
- Unrecoverable error

The handoff must accurately reflect the current state. The next session depends on it.

## Execution Rules

### Validate Before Claiming Success

Never claim a task is complete without validation:

```bash
small validate
small lint
```

Both must pass. A task with failing validation is not complete.

### Record All Executions

Every command execution must be recorded in progress:

```bash
small apply --cmd "your-command" --task task-id
```

Do not execute commands outside of `small apply` unless they are read-only exploratory commands.

### Respect Constraints

Read `constraints.small.yml` before every action. Each constraint has a severity:

- `error` - Violation blocks all progress. Stop and report.
- `warn` - Violation should be reported but does not block.

When in doubt, treat as `error`.

### Follow the Plan

Work through tasks in `plan.small.yml` in dependency order:

1. Find tasks with `status: "pending"` and no unmet dependencies
2. Set task to `in_progress` before starting
3. Complete the task
4. Record evidence in progress
5. Set task to `completed`
6. Move to next task

Do not skip tasks. Do not work on tasks with unmet dependencies.

### Completion Rule

Agents MUST NOT use `small plan --done` (or any plan status toggle) to represent completion.

Completion MUST be recorded using `small checkpoint`, which performs an atomic state transition and appends progress evidence.

**Required completion patterns:**

Start work:

```bash
small progress add --task <task-id> --status in_progress --evidence "Starting <task>"
```

Finish work:

```bash
small checkpoint --task <task-id> --status completed --evidence "<what was completed>"
```

Block work:

```bash
small checkpoint --task <task-id> --status blocked --evidence "<why blocked>"
```

**Handoff gate:**

Before generating a handoff, agents MUST run:

```bash
small check --strict
```

Handoff is only valid if strict mode passes and all completed or blocked tasks have progress evidence.

**Rationale:**

- Plan status alone is not evidence
- Checkpoint is the canonical completion primitive for agents
- Strict mode prevents "completed in plan, missing in progress" drift

## When Stuck

### Missing Intent or Constraints

If `intent.small.yml` or `constraints.small.yml` lacks information needed to proceed:

1. Do not guess
2. Do not invent requirements
3. Ask the human explicitly for the missing information
4. Wait for their response
5. Proceed only after they update the artifact

### Ambiguous Requirements

If the intent or constraints are ambiguous:

1. State the ambiguity explicitly
2. Propose interpretations
3. Ask the human to clarify
4. Do not proceed until clarified

### Blocked Tasks

If a task cannot be completed:

1. Mark it as `blocked` in the plan
2. Add a progress entry explaining why
3. Move to the next unblocked task
4. If all tasks are blocked, handoff with explanation

### Validation Failures

If validation fails after your changes:

1. Do not ignore the failure
2. Read the error message
3. Fix the issue
4. Re-validate
5. Only proceed after validation passes

### Unrecoverable Errors

If you encounter an error you cannot resolve:

1. Record the error in progress
2. Mark relevant tasks as blocked
3. Generate a handoff with full context
4. Stop and report to the human

## Prohibited Actions

### Never Do These

- Edit `intent.small.yml` or `constraints.small.yml`
- Delete or modify existing progress entries
- Skip validation before claiming success
- Execute commands without recording them
- End a session without generating handoff
- Guess at missing constraints or intent
- Proceed when validation fails
- Ignore constraint violations

### Never Assume

- That conversation history is accurate
- That previous work was validated
- That constraints haven't changed
- That the workspace is in a valid state

Always read `.small/` and validate.

## Session Lifecycle

### Starting a Session

```
1. Read handoff.small.yml
2. Read intent.small.yml
3. Read constraints.small.yml
4. Read plan.small.yml
5. Read progress.small.yml
6. Run small validate
7. Run small lint
8. Resume from handoff.resume.current_task_id
```

### During a Session

```
1. Check next actionable task (pending, dependencies met)
2. Update task status to in_progress
3. Execute work, recording with small apply
4. Validate after each significant change
5. Update task status to completed
6. Repeat
```

### Ending a Session

```
1. Run small validate
2. Run small lint
3. Generate handoff with accurate summary
4. Report final status to human
```

## Evidence Standards

Progress entries must be verifiable. Good evidence:

```yaml
entries:
  - task_id: "task-1"
    timestamp: "2025-01-04T10:00:00.000000000Z"
    status: "completed"
    command: "npm test"
    evidence: "All 47 tests passed"
    commit: "a1b2c3d"
```

Bad evidence (too vague):

```yaml
entries:
  - task_id: "task-1"
    evidence: "Did the thing"  # Not verifiable
```

Evidence should allow a future agent or human to verify the claim.

## Summary

| Rule | Requirement |
|------|-------------|
| Read first | Always read .small/ before acting |
| Validate | Run validate and lint before claiming success |
| Human files | Never edit intent or constraints |
| Progress | Append-only with evidence |
| Completion | Use `small checkpoint`, not `plan --done` |
| Strict gate | Run `small check --strict` before handoff |
| Handoff | Required when stopping |
| When stuck | Ask, don't guess |
| Constraints | Respect all, especially severity: error |
