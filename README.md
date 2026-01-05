# SMALL Protocol

**SMALL** is a formal execution protocol for making AI-assisted work legible, deterministic, and resumable.

It defines a small set of canonical, machine-readable artifacts that allow humans and automated agents to coordinate work without relying on ephemeral chat history or implicit context.

SMALL is **not**:

- an agent framework
- a prompt format
- a workflow engine

It is a **governance and continuity layer**.

---

## Versioning

This repository implements **SMALL Protocol v1.0.0**.

- v1.0.0 is **stable**
- Invariants are **locked**
- Schemas are **authoritative**
- Backward compatibility is **not guaranteed** prior to v1.0.0

---

## Installation

### Option 1 (Recommended): Install via Go

> Requires Go 1.22+ and installs a single `small` binary.

This installs the `small` CLI into Go's binary directory.

```bash
go install github.com/justyn-clark/small-protocol/cmd/small@latest
```

Verify installation:

```bash
small version
```

Expected output:

```
small v1.0.0
Supported spec versions: ["1.0.0"]
```

#### If `small` is not found

If you see:

```
zsh: command not found: small
```

Your Go bin directory is not on PATH. Fix it:

```bash
echo 'export PATH="$(go env GOPATH)/bin:$PATH"' >> ~/.zshrc
source ~/.zshrc
```

Then retry:

```bash
small version
```

---

### Option 2: Build from Source

```bash
git clone https://github.com/justyn-clark/small-protocol.git
cd small-protocol
make small-build
```

Run the binary directly:

```bash
./bin/small version
```

Optional: install globally:

```bash
sudo cp ./bin/small /usr/local/bin/small
small version
```

---

## Quick Start

Initialize a SMALL workspace:

```bash
small init --name myproject
```

This creates a `.small/` directory containing the five canonical artifacts:

```
.small/
├── intent.small.yml
├── constraints.small.yml
├── plan.small.yml
├── progress.small.yml
└── handoff.small.yml
```

Validate your artifacts:

```bash
small validate
```

Check project status:

```bash
small status
```

---

## The Five Canonical Artifacts

| Artifact | Owner | Purpose |
|----------|-------|---------|
| `intent.small.yml` | Human | Declares what the work is |
| `constraints.small.yml` | Human | Declares what must not change |
| `plan.small.yml` | Agent | Proposed execution steps |
| `progress.small.yml` | Agent | Verified execution evidence |
| `handoff.small.yml` | System | Serialized resume checkpoint |

All artifacts:

- **MUST** declare `small_version: "1.0.0"`
- **MUST** validate against the v1.0.0 schemas
- **MUST** satisfy locked invariants

---

## CLI Commands

### Core Commands

| Command | Description |
|---------|-------------|
| `small version` | Print CLI version and supported spec versions |
| `small init` | Initialize a new SMALL project with all canonical artifacts |
| `small validate` | Validate all artifacts against JSON schemas |
| `small lint` | Check protocol invariants (secrets, ownership, append-only) |
| `small handoff` | Generate a deterministic handoff document for agent resume |

### Execution Commands

| Command | Description |
|---------|-------------|
| `small plan` | Create or update the plan artifact |
| `small status` | Display compact summary of project state |
| `small apply` | Bounded execution gate that runs commands and records progress |

### Command Details

**`small plan`** - Deterministic plan management:

```bash
small plan --add "Task description"    # Append a new task
small plan --done <task-id>            # Mark task as completed
small plan --pending <task-id>         # Mark task as pending
small plan --blocked <task-id>         # Mark task as blocked
small plan --depends <task-id>:<dep>   # Add dependency edge
small plan --reset --yes               # Reset plan to template
```

**`small status`** - Project state summary:

```bash
small status              # Human-readable output
small status --json       # Machine-readable JSON output
small status --recent 10  # Show 10 recent progress entries
small status --tasks 5    # Show 5 actionable tasks
```

**`small apply`** - Bounded execution:

```bash
small apply --dry-run --cmd "npm test"           # Preview without executing
small apply --cmd "make build" --task task-5     # Execute and record
small apply --cmd "go test" --handoff            # Execute and generate handoff
```

---

## Validation

Validate the current workspace:

```bash
small validate
```

Strict validation (enforces secret detection and invariant hard-fail):

```bash
small validate --strict
```

---

## Handoff Generation

Generate a resumable checkpoint:

```bash
small handoff
```

This produces a `handoff.small.yml` artifact that allows a different agent or human to resume work deterministically.

---

## Invariants (v1.0.0)

The following are non-negotiable:

- `intent` and `constraints` **MUST** be human-owned
- `plan` and `progress` **MUST** be agent-owned
- `progress` entries **MUST** contain evidence
- Schema validation **MUST** pass before execution is considered valid
- `small_version` **MUST** be exactly `"1.0.0"`

**Violations are hard errors.**

---

## Specification

Authoritative specification and schemas:

```
spec/small/v1.0.0/
├── SPEC.md
├── schemas/
└── examples/
```

Earlier versions are retained for historical reference only.

See [`spec/small/v1.0.0/SPEC.md`](spec/small/v1.0.0/SPEC.md) for the complete specification.

---

## Philosophy

SMALL treats execution as a first-class artifact.

When systems fail, SMALL leaves evidence instead of speculation.

It functions as a **flight recorder** for agentic workflows.

### Core Principles

- Schemas are law
- Explicit state beats implicit state
- Agents are tools, not magic
- Determinism beats "AI vibes"
- Failure is safer than ambiguity

---

## What SMALL Is Not

| What It's Not | Why |
|---------------|-----|
| A CMS | SMALL stores project state metadata, not content |
| A task runner | `small apply` records execution; it does not orchestrate it |
| A multi-agent framework | SMALL is single-writer by design |
| An LLM executor | It's a protocol, not an AI product |

---

## Development

### Build CLI

```bash
make small-build
```

### Validate Examples

```bash
./bin/small validate --dir spec/small/v1.0.0/examples
```

### Run Tests

```bash
make small-test
```

---

## Requirements

- Go 1.22 or later

---

## License

Apache License 2.0 - See [LICENSE](LICENSE) for details.

Copyright 2025 Justyn Clark

---

## Status

**SMALL Protocol v1.0.0 is production-ready.**

Tooling will evolve. The protocol will not.
