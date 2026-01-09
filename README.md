# SMALL Protocol

[![CI](https://github.com/justyn-clark/small-protocol/actions/workflows/ci.yml/badge.svg)](https://github.com/justyn-clark/small-protocol/actions/workflows/ci.yml)
[![Release](https://github.com/justyn-clark/small-protocol/actions/workflows/release.yml/badge.svg)](https://github.com/justyn-clark/small-protocol/actions/workflows/release.yml)
[![Go Version](https://img.shields.io/badge/go-1.22%2B-blue)](https://go.dev)
[![License](https://img.shields.io/badge/license-Apache%202.0-blue)](LICENSE)

**SMALL (Schema, Manifest, Artifact, Lineage, Lifecycle)** is a formal execution protocol for making AI-assisted work legible, deterministic, and resumable.

It defines a minimal, fixed set of machine-readable artifacts that replace ephemeral chat history with durable execution state.

## What SMALL Is Not

- An agent framework
- A prompt format
- A workflow engine
- A multi-agent system

SMALL is a **governance and continuity layer**.

## The Five Canonical Artifacts

| Artifact                | Owner  | Purpose                       |
|-------------------------|--------|-------------------------------|
| `intent.small.yml`      | Human  | Declares what the work is     |
| `constraints.small.yml` | Human  | Declares what must not change |
| `plan.small.yml`        | Agent  | Proposed execution steps      |
| `progress.small.yml`    | Agent  | Verified execution evidence   |
| `handoff.small.yml`     | System | Serialized resume checkpoint  |

All artifacts must declare `small_version: "1.0.0"` and validate against the authoritative schemas.

## Version

This repository implements **SMALL Protocol v1.0.0**.

- v1.0.0 is stable
- Invariants are locked
- Schemas are authoritative

## Specification

The authoritative specification is located at:

```
spec/small/v1.0.0/
├── SPEC.md
├── schemas/
└── examples/
```

## Getting Started

**New to SMALL?** Start here:

1. [Getting Started Guide](docs/getting-started.md) - Answers "Do I manually type these files?" and walks through the workflow
2. [Agent Operating Contract](docs/agent-operating-contract.md) - Required reading for AI agents using SMALL

**Human workflow**: Edit `intent.small.yml` and `constraints.small.yml`. The agent handles the rest.

**Agent workflow**: Read `.small/` first, respect ownership rules, validate before claiming success, handoff when stopping.

## Documentation

| Document | Description |
|----------|-------------|
| [Getting Started](docs/getting-started.md) | First-time user guide with examples |
| [Agent Operating Contract](docs/agent-operating-contract.md) | Behavioral rules for AI agents |
| [CLI Guide](docs/cli-guide.md) | Detailed command reference with error handling |
| [Installation](docs/installation.md) | Install via Go, binaries, or source |
| [Quick Start](docs/quickstart.md) | Initialize and validate a SMALL workspace |
| [CLI Reference](docs/cli.md) | Command summary table |
| [Invariants](docs/invariants.md) | Non-negotiable protocol rules |
| [Enterprise Integration](docs/enterprise.md) | Git, CI/CD, and audit patterns |
| [Philosophy](docs/philosophy.md) | Design rationale and non-goals |
| [Releasing](docs/releasing.md) | How releases are built and verified |

## Quick Start

Run all commands from your repository root.

```bash
# Install
go install github.com/justyn-clark/small-protocol/cmd/small@latest

# Initialize
small init --intent "My project description"

# Validate
small validate
```

Or download pre-built binaries from [GitHub Releases](https://github.com/justyn-clark/small-protocol/releases).
See [Installation](docs/installation.md) for checksum verification and PATH setup.

## License

Apache License 2.0. See [LICENSE](LICENSE).
