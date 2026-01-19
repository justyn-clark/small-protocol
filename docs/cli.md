# CLI Reference

## Core Commands

| Command          | Description                           |
|------------------|---------------------------------------|
| `small version`  | Print CLI and supported spec versions |
| `small init`     | Initialize `.small/` artifacts        |
| `small validate` | Schema validation                     |
| `small lint`     | Invariant enforcement                 |
| `small check`    | Run validate, lint, and verify        |
| `small handoff`  | Generate resume checkpoint            |
| `small reset`    | Reset workspace for new run           |
| `small verify`   | CI/local enforcement gate             |
| `small doctor`   | Diagnose workspace issues             |
| `small selftest` | Built-in CLI self-test                |
| `small archive`  | Archive run state for lineage         |
| `small run`      | Run history snapshots and diff        |

## Execution Commands

| Command            | Description                         |
|--------------------|-------------------------------------|
| `small plan`       | Manage plan tasks                   |
| `small status`     | Project state summary               |
| `small emit`       | Emit structured JSON state          |
| `small apply`      | Execute and record bounded commands |
| `small progress`   | Progress utilities (add, migrate)   |
| `small checkpoint` | Update plan and progress atomically |

## Common Flags

| Flag          | Description                              |
|---------------|------------------------------------------|
| `--dir`       | Target directory containing .small/      |
| `--workspace` | Workspace scope (root, examples, or any) |
| `--strict`    | Enable strict mode (lint, verify, check) |
| `--json`      | Output in JSON format (status, check)    |
| `--help`      | Show help for any command                |

## Examples

Initialize a new workspace:

```bash
small init --intent "Build authentication service"
```

Validate all artifacts:

```bash
small validate
```

Check project status:

```bash
small status
```

Emit JSON for integrations:

```bash
small emit --check --workspace root
```

Generate a handoff checkpoint:

```bash
small handoff
```

Snapshot and list run history:

```bash
small run snapshot
small run list
```

Reset workspace for a new run:

```bash
small reset --yes
small reset --keep-intent  # Preserve intent.small.yml
```

Verify artifacts for CI:

```bash
small verify        # Exit 0 if valid, 1 if invalid, 2 if system error
small verify --ci   # Minimal output for CI pipelines
small verify --strict  # Enable strict checks (invariants, secrets, insecure links)
```

The `verify` command enforces:

- All completed tasks must have progress entries with evidence
- `handoff.small.yml` must include a valid `replayId`
- Schema validation for all artifacts
- Ownership rules (human/agent) for each artifact type
- Strict mode adds S1-S3 invariants for evidence, task ids, and handoff alignment

If you forget to update SMALL files after completing work, `verify` will fail.

Diagnose workspace issues:

```bash
small doctor  # Read-only, never mutates state
```
