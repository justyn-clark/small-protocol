# @small-protocol/small

@small-protocol/small is the CLI for the SMALL Protocol - deterministic execution state for auditable, resumable AI-assisted engineering workflows.

It gives humans and agents a shared, file-based contract for intent, constraints, plan, progress, and handoff so work can be resumed, verified, and audited without relying on chat memory.

## Install

Latest:

```bash
curl -fsSL https://smallprotocol.dev/install.sh | bash
```

Pinned version:

```bash
curl -fsSL https://smallprotocol.dev/install.sh | bash -s -- --version v1.0.8
```

npm:

```bash
npm i -g @small-protocol/small
small version
```

## Why it exists

Most agent workflows lose state, drift across tools, and become hard to verify. SMALL makes execution state explicit, resumable, and machine-legible.

## Quick start

```bash
small init --intent "Ship a deterministic release process"
small check
small status
small handoff
```

## What this package does

- Maps npm version `X.Y.Z` to GitHub release tag `vX.Y.Z`
- Downloads the native SMALL binary for your platform from GitHub Releases
- Verifies SHA256 using `checksums.txt` before extraction
- Exposes the installed binary as `small` on your PATH

## Learn more

- Documentation: https://smallprotocol.dev
- Installation guide: https://github.com/justyn-clark/small-protocol/blob/main/docs/installation.md
- GitHub: https://github.com/justyn-clark/small-protocol
- Releases: https://github.com/justyn-clark/small-protocol/releases

## Migration note

Canonical runtime lineage locations are now `.small-runs/` and `.small-archive/`.

Legacy `.small/archive/` and `.small/runs/` layouts can be repaired with:

```bash
small fix --runtime-layout
```
