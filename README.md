# SMALL Protocol

**SMALL is a protocol for durable, agent-legible project continuity.**

SMALL defines five canonical artifacts that enable agents to understand, execute, and resume work across sessions. It is not a CMS, not an agent framework, and not MCP.

## What is SMALL?

SMALL (Schema, Manifest, Artifact, Lineage, Lifecycle) provides a minimal, strict protocol for agent continuity.

Agents and humans can use SMALL to:

- make project intent explicit
- define and validate artifact contracts
- track verifiable progress
- generate deterministic handoffs
- resume work across sessions, tools, and agents

SMALL is schema-first, file-based, and tool-agnostic.

## Specification Status

**SMALL v0.1 is frozen.**

- The v0.1 protocol is considered stable and complete.
- No breaking changes will be made to v0.1.
- All future evolution will occur in v0.2+.
- Tooling and extensions may evolve, but the v0.1 artifact contract is locked.

If you are implementing SMALL today, target **v0.1**.

### Core Principles

- **Explicit State**: All project state is stored in canonical YAML files
- **Ownership Rules**: Clear separation between human-owned and agent-owned artifacts
- **Verifiable Progress**: Every progress entry must include evidence
- **Deterministic Handoff**: Agents resume from a single entrypoint

## Installation

### Option 1: Install with Go

```bash
go install github.com/justyn-clark/small-protocol/cmd/small@latest
```

### Option 2: Build from Source

```bash
git clone https://github.com/justyn-clark/small-protocol.git
cd small-protocol
make small-build
# Binary will be at ./bin/small
```

### Option 3: Download Pre-built Binary

Download the latest release from [GitHub Releases](https://github.com/justyn-clark/small-protocol/releases) (coming soon).

## Quickstart

Verify installation:

```bash
small version
# small v0.1.0
# Supported spec versions: ["0.1"]
```

Initialize a new SMALL project:

```bash
small init --name myproject
# Initialized SMALL project in /path/to/project/.small
```

This creates `.small/` with all five canonical files:

- `intent.small.yml` - Project goals (human-owned)
- `constraints.small.yml` - Requirements (human-owned)
- `plan.small.yml` - Execution plan (agent-owned)
- `progress.small.yml` - Progress tracking (agent-owned)
- `handoff.small.yml` - Resume entrypoint (shared)

Validate your artifacts:

```bash
small validate
# All artifacts are valid
```

Lint for protocol invariants:

```bash
small lint
# All invariants satisfied
```

Add a task to the plan:

```bash
small plan --add "Implement authentication"
# Added task task-5: Implement authentication
```

Check project status:

```bash
small status
# Displays: artifacts, task summary, next actionable tasks, last handoff
```

Execute a command (dry-run by default):

```bash
small apply --dry-run --cmd "npm test"
# Dry-run mode: no command will be executed
# Would execute: npm test
```

Execute with recording:

```bash
small apply --cmd "make build" --task task-5
# Executing: make build
# [command output]
# Command completed successfully (exit code: 0)
```

Generate a handoff:

```bash
small handoff --recent 3
# Generated handoff.small.yml with 3 recent progress entries
```

### What lives in `.small/`

The `.small/` directory contains all project state as canonical YAML files. Intent and constraints are human-owned (you edit them). Plan and progress are agent-owned (tools manage them). Handoff is the single entrypoint for resuming work across sessions.

### Safety Defaults

- `small apply` defaults to dry-run mode if no `--cmd` is provided
- All commands are read-only except `init`, `plan`, and `apply`
- Progress is append-only (never deleted)
- Execution always records timestamp, command, exit code, and evidence

## CLI Command Model

The SMALL CLI provides deterministic, composable commands for managing SMALL artifacts:

### Core Commands

| Command | Description |
|---------|-------------|
| `small init` | Initialize a new SMALL project with all canonical artifacts |
| `small validate` | Validate all artifacts against JSON schemas |
| `small lint` | Check protocol invariants (secrets, ownership, append-only) |
| `small handoff` | Generate a deterministic handoff document for agent resume |
| `small version` | Display CLI and supported spec versions |

### Execution Commands

| Command | Description |
|---------|-------------|
| `small plan` | Create or update the plan artifact (add tasks, set status, manage dependencies) |
| `small status` | Display compact summary of project state (artifacts, tasks, progress) |
| `small apply` | Bounded execution gate that runs commands and records progress |

### Command Details

**`small plan`** - Deterministic plan management:
- `--add "Task description"` - Append a new task
- `--done <task-id>` - Mark task as completed
- `--pending <task-id>` - Mark task as pending
- `--blocked <task-id>` - Mark task as blocked
- `--depends <task-id>:<dep-id>` - Add dependency edge
- `--reset --yes` - Reset plan to template (destructive)

**`small status`** - Project state summary:
- `--json` - Machine-readable JSON output
- `--recent <n>` - Number of progress entries (default: 5)
- `--tasks <n>` - Number of actionable tasks (default: 3)

**`small apply`** - Bounded execution (not an LLM executor):
- `--cmd "<command>"` - Shell command to execute
- `--task <task-id>` - Associate with a specific task
- `--handoff` - Generate handoff after successful execution
- `--dry-run` - Record intent without executing (default if no --cmd)

All commands are safe-by-default and composable in CI pipelines.

## Repository Structure

This repository contains:

- **`spec/small/v0.1/`** - SMALL protocol specification
  - `SPEC.md` - Normative specification document
  - `schemas/` - JSON Schema definitions for all artifacts
  - `examples/.small/` - Example YAML files

- **`cmd/small/`** - Reference CLI implementation
  - Go-based tool for validating and managing SMALL artifacts
  - Commands: `init`, `validate`, `lint`, `handoff`, `plan`, `status`, `apply`, `version`

## Protocol Specification

See [`spec/small/v0.1/SPEC.md`](spec/small/v0.1/SPEC.md) for the complete specification.

### Canonical Artifacts

1. **`.small/intent.small.yml`** - Human-owned project intent/goals
2. **`.small/constraints.small.yml`** - Human-owned constraints/requirements
3. **`.small/plan.small.yml`** - Agent-owned execution plan
4. **`.small/progress.small.yml`** - Agent-owned progress tracking
5. **`.small/handoff.small.yml`** - Shared resume entrypoint

### Invariants

- Secrets never stored in any artifact
- Progress entries must include verifiable evidence
- Plan is disposable; progress is not (append-only)
- Handoff is the only resume entrypoint
- `small_version` must be `"0.1"` in all files

## Development

### Build CLI

```bash
make small-build
```

### Validate Examples

```bash
./bin/small validate --dir spec/small/v0.1/examples
```

### Run Tests

```bash
make small-test
```

## Status

SMALL v0.1 is the current stable protocol version. The specification is complete and the reference CLI implementation is available.

Breaking changes are expected until v1.0.

## Philosophy

- Schemas are law
- Explicit state beats implicit state
- Agents are tools, not magic
- Determinism beats "AI vibes"
- Infrastructure first, products follow

## Related Repos

- **small-cms**: Demo implementation and orchestration runtime (optional)

## License

Apache-2.0 - see [LICENSE](LICENSE) for details.

Copyright 2025 Justyn Clark
