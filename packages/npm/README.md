# @small-protocol/small

@small-protocol/small is the CLI for the SMALL Protocol - deterministic execution state for auditable, resumable AI-assisted engineering workflows.

It gives humans and agents a shared, file-based contract for intent, constraints, plan, progress, and handoff so work can be resumed, verified, and audited without relying on chat memory.

Version 1.1.0 supports the stable SMALL 1.0.0 artifact profile and the
separately versioned SMALL 2.0.0 session profile. New workspaces remain v1 by
default. V2 migration is explicit, starts in solo mode unless collaboration is
requested, and preserves the original v1 bytes.

## Install

Latest:

```bash
curl -fsSL https://smallprotocol.dev/install.sh | bash
```

Pinned version:

```bash
curl -fsSL https://smallprotocol.dev/install.sh | bash -s -- --version vX.Y.Z
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

To opt an existing reviewed workspace into the v2 session profile, preview the
migration first and apply it with the returned input digest:

```bash
small migrate --to 2.0.0 --preview --namespace my-project-baseline --json > /tmp/migration.json
small migrate --apply /tmp/migration.json --expect-state <expected_input_digest>
small session start --label first-v2-session
```

Collaboration is opt-in. Omit `--mode collaborative` during migration to keep
the default solo policy.

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
