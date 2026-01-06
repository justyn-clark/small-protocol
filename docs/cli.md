# CLI Reference

## Core Commands

| Command          | Description                           |
|------------------|---------------------------------------|
| `small version`  | Print CLI and supported spec versions |
| `small init`     | Initialize `.small/` artifacts        |
| `small validate` | Schema validation                     |
| `small lint`     | Invariant enforcement                 |
| `small handoff`  | Generate resume checkpoint            |

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
