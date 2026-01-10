# CLI Reference

## Core Commands

| Command          | Description                           |
|------------------|---------------------------------------|
| `small version`  | Print CLI and supported spec versions |
| `small init`     | Initialize `.small/` artifacts        |
| `small validate` | Schema validation                     |
| `small lint`     | Invariant enforcement                 |
| `small handoff`  | Generate resume checkpoint            |
| `small reset`    | Reset workspace for new run           |
| `small verify`   | CI/local enforcement gate             |
| `small doctor`   | Diagnose workspace issues             |

## Execution Commands

| Command        | Description                         |
|----------------|-------------------------------------|
| `small plan`   | Manage plan tasks                   |
| `small status` | Project state summary               |
| `small apply`  | Execute and record bounded commands |

## Common Flags

| Flag       | Description                          |
|------------|--------------------------------------|
| `--strict` | Fail on warnings                     |
| `--json`   | Output in JSON format                |
| `--help`   | Show help for any command            |

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

Generate a handoff checkpoint:

```bash
small handoff
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
small verify --strict  # Enable strict checks (secrets, insecure links)
```

Diagnose workspace issues:

```bash
small doctor  # Read-only, never mutates state
```
