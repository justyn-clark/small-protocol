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

## Documentation

| Document | Description |
|----------|-------------|
| [Installation](docs/installation.md) | Install via Go or build from source |
| [Quick Start](docs/quickstart.md) | Initialize and validate a SMALL workspace |
| [CLI Reference](docs/cli.md) | Command reference |
| [Invariants](docs/invariants.md) | Non-negotiable protocol rules |
| [Enterprise Integration](docs/enterprise.md) | Git, CI/CD, and audit patterns |
| [Philosophy](docs/philosophy.md) | Design rationale and non-goals |

## Quick Start

```bash
go install github.com/justyn-clark/small-protocol/cmd/small@latest
small init --intent "My project description"
small validate
```

## Releases

Tagged releases publish cross-platform binaries to [GitHub Releases](https://github.com/justyn-clark/small-protocol/releases).

### Creating a Release

```bash
git tag -a v1.0.0 -m "SMALL v1.0.0"
git push origin v1.0.0
```

The release workflow builds binaries for:
- macOS (amd64, arm64)
- Linux (amd64, arm64)
- Windows (amd64, arm64)

### Verifying Downloads

Each release includes a `checksums.txt` file with SHA256 hashes:

```bash
# Download the binary and checksums
curl -LO https://github.com/justyn-clark/small-protocol/releases/download/v1.0.0/small_1.0.0_Darwin_arm64.tar.gz
curl -LO https://github.com/justyn-clark/small-protocol/releases/download/v1.0.0/checksums.txt

# Verify checksum
sha256sum -c checksums.txt --ignore-missing
```

## License

Apache License 2.0. See [LICENSE](LICENSE).
