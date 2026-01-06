# SMALL Protocol

**SMALL (Schema, Manifest, Artifact, Lineage, Lifecycle)** is a formal execution protocol for making AI-assisted work legible, deterministic, and resumable.

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
command -v small && small version
```

Expected output:

```
/Users/you/go/bin/small
small v1.0.0
Supported spec versions: ["1.0.0"]
```

#### If `small` is not found

If `command -v small` returns nothing, the Go bin directory is not on PATH.

**zsh (macOS default):**

```bash
echo 'export PATH="$(go env GOPATH)/bin:$PATH"' >> ~/.zshrc
source ~/.zshrc
```

**bash:**

```bash
echo 'export PATH="$(go env GOPATH)/bin:$PATH"' >> ~/.bashrc
source ~/.bashrc
```

**Common fixes:**

- Ensure `$(go env GOPATH)/bin` is on PATH
- Restart your terminal or source your shell rc file
- Confirm you installed to the same Go you're running (`which go`)
- On macOS with Homebrew: ensure you're using the Homebrew Go (`brew --prefix go`)

Then verify:

```bash
command -v small   # Should print a path
small version      # Should print version info
```

If `command -v small` still returns nothing, you are not installed correctly.

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
small init --intent "My project description"
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

## End-to-End: 5 Minutes

A complete first-run walkthrough. Run these commands from your project root.

### 1. Install

```bash
go install github.com/justyn-clark/small-protocol/cmd/small@latest
```

### 2. Verify installation

```bash
command -v small          # Should print: /path/to/go/bin/small
small version             # Should print: small v1.0.0
```

### 3. Initialize a project

```bash
mkdir myproject && cd myproject
small init --intent "My project description"
```

### 4. Validate and lint

```bash
small validate            # Schema validation
small lint                # Invariant checks
```

### 5. Add a task

```bash
small plan --add "Implement user authentication"
small status              # See the new task
```

### 6. Preview execution

```bash
small apply --dry-run --cmd "echo 'Hello SMALL'"
```

### 7. Execute and record

```bash
small apply --cmd "echo 'Hello SMALL'"
small status --recent 1   # See the recorded progress entry
```

### 8. Generate handoff

```bash
small handoff
```

### 9. Verify artifacts exist

```bash
ls -la .small/
git status                # .small/ artifacts ready to commit
```

You now have a working SMALL project with tracked execution history.

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

## What a SMALL Project Looks Like

A SMALL project is always the same **five canonical files**. The set does not change.

```
.small/
├── intent.small.yml        # Stable: what the work is
├── constraints.small.yml   # Stable: what must not change
├── plan.small.yml          # Grows: tasks added/updated over time
├── progress.small.yml      # Grows: append-only verified execution log
└── handoff.small.yml       # Updates: checkpoint regenerated on handoff
```

### How artifacts evolve

| Artifact | Behavior |
|----------|----------|
| `intent` | Stable. Set once, rarely edited. |
| `constraints` | Stable. Set once, rarely edited. |
| `plan` | Grows. Tasks are added, updated, marked done. |
| `progress` | Append-only. Each verified execution adds an entry. |
| `handoff` | Regenerated. Updates when you run `small handoff`. |

Growth happens **inside** `plan` and `progress`, not by multiplying file types.

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

The CLI includes embedded v1.0.0 schemas, so validation works in any repository without requiring the small-protocol spec directory.

**Schema resolution order:**

1. `--spec-dir` flag (or `$SMALL_SPEC_DIR` env var) - explicit path to spec directory
2. On-disk schemas in `spec/small/v1.0.0/schemas/` - for development in small-protocol repo
3. Embedded schemas - default for installed CLI

Strict validation (enforces secret detection and invariant hard-fail):

```bash
small lint --strict
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

## Explicit Non-Goals

| What It's Not | Why |
|---------------|-----|
| A CMS | SMALL stores project state metadata, not content |
| A task runner | `small apply` records execution; it does not orchestrate it |
| A multi-agent framework | SMALL intentionally enforces a single active workstream per `.small/` directory to preserve determinism, replay integrity, and unambiguous authorship |
| An LLM executor | It's a protocol, not an AI product |

---

## Enterprise Posture: Audit, Security, Lineage

SMALL is **stateless by design**: it maintains no centralized service, daemon, or runtime memory. State is represented exclusively through explicit, versioned artifacts that can be committed, exported, or ingested elsewhere.

### Centralized lineage

Centralized lineage capture is done by **exporting/ingesting** SMALL artifacts into your existing systems, not by making SMALL itself a server.

**Recommended patterns:**

| Environment | Pattern |
|-------------|---------|
| Git-based teams | Commit `.small/` to git for immutable history |
| CI/CD pipelines | Archive `.small/progress.small.yml` and `small status --json` as build artifacts |
| Regulated environments | Ingest progress entries into Splunk, Datadog, ELK, or a database as an audit stream |

### What SMALL does not replace

SMALL does not replace IAM, RBAC, or access control. It complements them by making agent execution **legible**: who did what, when, and with what evidence.

### The flight recorder

SMALL functions as a **flight recorder** for agentic work. When something breaks, you have evidence, not vibes. Every verified execution is logged with timestamps, commands, and outcomes. This makes post-incident review and compliance audits straightforward.

---

## Development

### Build CLI

```bash
make small-build
```

This automatically syncs schemas from `spec/small/v1.0.0/schemas/` to the embedded location before building.

### Validate Examples

```bash
./bin/small validate --dir spec/small/v1.0.0/examples
```

When running inside the small-protocol repo, the CLI automatically uses on-disk schemas for development.

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

The SMALL Protocol defines structure and invariants only; tooling, automation, and integrations are intentionally out of scope and left to implementations.

Tooling will evolve. The protocol will not.
