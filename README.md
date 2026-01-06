# SMALL Protocol

[![CI](https://github.com/justyn-clark/small-protocol/actions/workflows/ci.yml/badge.svg)](https://github.com/justyn-clark/small-protocol/actions/workflows/ci.yml)
[![Release](https://github.com/justyn-clark/small-protocol/actions/workflows/release.yml/badge.svg)](https://github.com/justyn-clark/small-protocol/actions/workflows/release.yml)
[![Go Version](https://img.shields.io/badge/go-1.22%2B-blue)](https://go.dev)
[![License](https://img.shields.io/badge/license-Apache%202.0-blue)](LICENSE)

**SMALL (Schema, Manifest, Artifact, Lineage, Lifecycle)** is a formal execution protocol for making AI-assisted work **legible, deterministic, and resumable**.

It defines a minimal, fixed set of canonical, machine-readable artifacts that replace ephemeral chat history with durable execution state.

SMALL is **not**:

- an agent framework
- a prompt format
- a workflow engine
- a multi-agent system

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

```bash
go install github.com/justyn-clark/small-protocol/cmd/small@latest
```

Verify installation:

```bash
command -v small && small version
```

Expected output:

```text
/Users/you/go/bin/small
small v1.0.0
Supported spec versions: ["1.0.0"]
```

#### If `small` is not found

If `command -v small` returns nothing, your Go bin directory is not on PATH.

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

Then retry:

```bash
command -v small
small version
```

### Option 2: Build from Source

```bash
git clone https://github.com/justyn-clark/small-protocol.git
cd small-protocol
make small-build
```

Run directly:

```bash
./bin/small version
```

Optional global install:

```bash
sudo cp ./bin/small /usr/local/bin/small
small version
```

---

## Quick Start

> Run all small commands from your repository root.
> If executed inside `.small/`, SMALL will still resolve the workspace automatically.

Initialize a SMALL workspace:

```bash
small init --intent "My project description"
```

This creates a `.small/` directory with the five canonical artifacts:

```text
.small/
├── intent.small.yml
├── constraints.small.yml
├── plan.small.yml
├── progress.small.yml
└── handoff.small.yml
```

Validate artifacts:

```bash
small validate
```

Check project status:

```bash
small status
```

---

## What a SMALL Project Looks Like

A SMALL project always consists of exactly five files.
The set is fixed; growth happens inside the artifacts.

```text
.small/
├── intent.small.yml        # Stable: what the work is
├── constraints.small.yml   # Stable: what must not change
├── plan.small.yml          # Grows: tasks added and updated
├── progress.small.yml      # Append-only execution log
└── handoff.small.yml       # Regenerated resume checkpoint
```

### Artifact evolution

| Artifact    | Behavior                 |
|-------------|--------------------------|
| intent      | Set once, rarely edited  |
| constraints | Set once, rarely edited  |
| plan        | Grows over time          |
| progress    | Append-only              |
| handoff     | Regenerated on handoff   |

---

## The Five Canonical Artifacts

| Artifact              | Owner  | Purpose                         |
|-----------------------|--------|---------------------------------|
| intent.small.yml      | Human  | Declares what the work is       |
| constraints.small.yml | Human  | Declares what must not change   |
| plan.small.yml        | Agent  | Proposed execution steps        |
| progress.small.yml    | Agent  | Verified execution evidence     |
| handoff.small.yml     | System | Serialized resume checkpoint    |

All artifacts:

- MUST declare `small_version: "1.0.0"`
- MUST validate against v1.0.0 schemas
- MUST satisfy locked invariants

---

## CLI Commands

### Core Commands

| Command          | Description                        |
|------------------|------------------------------------|
| `small version`  | Print CLI and supported spec versions |
| `small init`     | Initialize `.small/` artifacts     |
| `small validate` | Schema validation                  |
| `small lint`     | Invariant enforcement              |
| `small handoff`  | Generate resume checkpoint         |

### Execution Commands

| Command        | Description                           |
|----------------|---------------------------------------|
| `small plan`   | Manage plan tasks                     |
| `small status` | Project state summary                 |
| `small apply`  | Execute and record bounded commands   |

---

## Invariants (v1.0.0)

The following are non-negotiable:

- `intent` and `constraints` MUST be human-owned
- `plan` and `progress` MUST be agent-owned
- `progress` MUST be append-only and evidence-backed
- Schema validation MUST pass
- `small_version` MUST equal `"1.0.0"`

> Violations are hard errors.

---

## Specification

Authoritative specification and schemas:

```text
spec/small/v1.0.0/
├── SPEC.md
├── schemas/
└── examples/
```

Earlier versions are retained for historical reference only.

---

## Enterprise Posture

SMALL is stateless by design.

It runs no servers, daemons, or background processes.
Lineage and audit are captured by exporting artifacts into existing systems.

| Environment | Pattern                                          |
|-------------|--------------------------------------------------|
| Git         | Commit `.small/` for immutable history           |
| CI/CD       | Archive progress and status output               |
| Audit       | Ingest artifacts into Splunk, ELK, Datadog, or a database |

SMALL does not replace IAM, RBAC, or enforcement systems.
It makes execution legible and provable.

---

## Philosophy

SMALL treats execution as a first-class artifact.

When systems fail, SMALL leaves evidence instead of speculation.

It functions as a flight recorder for agentic workflows.

---

## Explicit Non-Goals

| Not This             | Why                                       |
|----------------------|-------------------------------------------|
| Task runner          | Execution is recorded, not orchestrated   |
| Multi-agent framework| Single-writer preserves determinism       |
| LLM product          | Protocol only                             |
| Permission system    | Describes constraints, does not enforce them |

---

## Development

```bash
make verify
make small-build
make small-test
```

### Requirements

- Go 1.22 or later

---

## License

Apache License 2.0

See [LICENSE](LICENSE) for details.

---

## Status

**SMALL Protocol v1.0.0 is production-ready.**

Tooling will evolve.
The protocol will not.
