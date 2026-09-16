# Docs Sync Contract

This repository is the canonical source for public SMALL documentation.

## Integration model

- Model A: vendored canonical docs into docs site repo (`smallprotocol.dev`)

## Canonical path mapping

- `docs/installation.md` -> `smallprotocol.dev/app/modules/docs/content/canonical/installation.md`
- `docs/quickstart.md` -> `smallprotocol.dev/app/modules/docs/content/canonical/quickstart.md`
- `docs/getting-started.md` -> `smallprotocol.dev/app/modules/docs/content/canonical/getting-started.md`
- `docs/session-profile-v2.md` -> `smallprotocol.dev/app/modules/docs/content/canonical/session-profile-v2.md`
- `docs/maintainers/releasing.md` -> `smallprotocol.dev/app/modules/docs/content/canonical/releasing.md`
- `docs/maintainers/release-checklist.md` -> `smallprotocol.dev/app/modules/docs/content/canonical/release-checklist.md`
- `spec/small/v1.0.0/` -> `smallprotocol.dev/vendor/small-protocol/spec/small/v1.0.0/`
- `spec/small/v2.0.0/` -> `smallprotocol.dev/vendor/small-protocol/spec/small/v2.0.0/`

## Sync command

Run from `smallprotocol.dev`:

```bash
bash scripts/sync-small-protocol-docs.sh
```

This performs:

1. spec vendor sync (`scripts/sync-spec.sh`)
2. canonical docs copy (`scripts/sync-canonical-docs.sh`)
3. docs registry regeneration (`bun run generate-content-registry`)

The website also owns explanatory and navigation pages. When a protocol profile
or command surface changes, review those pages for claims that are broader than
v1; canonical file parity alone is not a complete website update.

## Required verification gates

Run from `smallprotocol.dev`:

```bash
bash scripts/check-canonical-docs.sh
bash scripts/check-spec-vendor.sh
bun run build
```
