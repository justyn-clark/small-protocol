# CLI Reference

Use `small --help` and `small <command> --help` for the exact current surface. This page is the high-level map of the shipped CLI.

## Workspace Setup And Authoring

| Command | Description |
|---------|-------------|
| `small init` | Initialize `.small/` artifacts |
| `small draft` | Create draft intent or constraints artifacts for human review |
| `small accept` | Accept draft intent or constraints into canonical files |
| `small agents` | Manage the SMALL harness block in `AGENTS.md` |
| `small plan` | Manage plan tasks and dependencies |
| `small progress` | Append or migrate progress entries |
| `small checkpoint` | Update plan status and progress atomically |
| `small apply` | Execute one bounded command and record the outcome |
| `small handoff` | Generate or update `handoff.small.yml` |
| `small start` | Initialize or repair run handoff state |
| `small mode` | Show or explicitly change v2 solo/collaborative policy |
| `small policy` | Show or atomically revise v2 human-owned policy material |
| `small session` | Start, list, close, resume, or take over v2 sessions |
| `small migrate` | Preview/apply/recover explicit v1-to-v2 migration |
| `small reconcile` | Preview and apply explicit semantic resolutions |
| `small evidence` | Save and verify typed portable evidence receipts |

For a v2 policy revision, run `small policy show --json`, review the proposed
intent/constraint files, then use `small policy revise --intent <file>
--constraints <file> --expect-state <frontier> --reason <text>`. Either file
flag may be omitted to retain its current material. The policy event and changed
files publish in one recoverable transaction.

## Validation And Inspection

| Command | Description |
|---------|-------------|
| `small validate` | Validate canonical artifacts against schemas |
| `small lint` | Enforce invariant rules beyond schema validation |
| `small check` | Run validate, lint, and verify |
| `small verify` | CI/local enforcement gate |
| `small doctor` | Diagnose workspace issues and suggest fixes |
| `small status` | Show compact signal-first project state |
| `small health` | Report strict status, replay id, artifact digest, and snapshot freshness across workspaces |
| `small reconstruct` | Reconstruct task or run evidence from plan, progress, and handoff state |
| `small emit` | Emit structured SMALL state in JSON |
| `small selftest` | Verify the installed CLI and runtime basics |

## Maintenance And History

| Command | Description |
|---------|-------------|
| `small fix` | Normalize or repair known SMALL artifact issues |
| `small reset` | Start a new run without losing audit history |
| `small run` | Snapshot, list, diff, show, and restore run history |
| `small archive` | Archive the current run state for lineage retention |
| `small version` | Print CLI and supported spec versions |
| `small completion` | Generate shell completion scripts |

## Common Flags

| Flag | Description |
|------|-------------|
| `--dir` | Target directory containing `.small/` when supported by the command |
| `--workspace` | Workspace scope (`root`, `examples`, or `any`) when supported by the command |
| `--strict` | Enable strict mode for lint, verify, or check |
| `--json` | Emit machine-readable output when supported |
| `--help` | Show command help |
| `-v`, `--version` | Print version from the root command |

## Examples

Initialize a new workspace:

```bash
small init --intent "Build authentication service"
```

Check project status:

```bash
small status
small check --strict
```

Inspect run history:

```bash
small run snapshot
small run list --limit 10
small run diff <fromReplayId> <toReplayId> --full
small run verify <replayId>
```

Inspect health and reconstruct durable evidence:

```bash
small health
small health ../other-repo --json
small reconstruct --task task-12
small reconstruct --resume --session <id> --limit 50 --max-bytes 65536 --json
small reconstruct --since 2026-01-01T00:00:00.000000000Z --json
```

Repair common state drift:

```bash
small fix --runtime-layout
small doctor
```
