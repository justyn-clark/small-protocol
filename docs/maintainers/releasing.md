# Releasing

This document describes how SMALL Protocol releases are created and verified.

## Tag policy

- Keep `v1.0.0` forever. It is the first stable contract tag and main reference point.
- Keep `v1.0.1` forever. It is the patch that fixes `go install` resolution and packaging consistency.
- Use SemVer and never move public tags.

## Release process

Releases are automated via GitHub Actions when a version tag is pushed.

```bash
git tag -a vX.Y.Z -m "SMALL vX.Y.Z"
git push origin vX.Y.Z
```

The release workflow:

1. Runs `go mod tidy`
2. Syncs embedded schemas
3. Builds binaries for supported platforms
4. Publishes archives plus `checksums.txt` to GitHub Releases

## Release assets (stable naming)

- `small-vX.Y.Z-darwin-amd64.tar.gz`
- `small-vX.Y.Z-darwin-arm64.tar.gz`
- `small-vX.Y.Z-linux-amd64.tar.gz`
- `small-vX.Y.Z-linux-arm64.tar.gz`
- `small-vX.Y.Z-windows-amd64.zip`
- `small-vX.Y.Z-windows-arm64.zip`
- `checksums.txt`

## Verifying downloads

```bash
curl -LO https://github.com/justyn-clark/small-protocol/releases/download/v1.0.8/small-v1.0.8-darwin-arm64.tar.gz
curl -LO https://github.com/justyn-clark/small-protocol/releases/download/v1.0.8/checksums.txt

shasum -a 256 -c checksums.txt --ignore-missing
```

## Installer paths

- curl installer endpoint: `https://smallprotocol.dev/install.sh`
- npm package: `@small-protocol/small`

Both install paths verify SHA256 using `checksums.txt` before install.

## npm publish policy (OIDC-only)

`@small-protocol/small` publish is OIDC-only and must produce provenance.

- Required workflow environment: Node 24 and npm 11.10.1
- Required publish command: `npm publish --provenance --access public`
- Required workflow permissions: `id-token: write`
- Required source alignment: Git tag `vX.Y.Z` must match package version `X.Y.Z`

Token-based publish (`NPM_TOKEN`) is not part of the standard release path.

Examples:

- Git tag: `v1.0.3`
- npm version: `1.0.3`

See [release checklist](./release-checklist.md) for the full sequence.
