# Releasing

This document describes how SMALL Protocol releases are created and verified.

## Tag Policy

- Keep `v1.0.0` forever. It is the first stable contract tag and main reference point.
- Keep `v1.0.1` forever. It is the patch that fixes `go install` resolution and packaging consistency.
- Going forward: use SemVer and never move public tags. If a correction is needed, publish `v1.0.2` (or next patch), do not retag.

## Release Process

Releases are automated via GitHub Actions when a version tag is pushed.

### Creating a Release

```bash
git tag -a vX.Y.Z -m "SMALL vX.Y.Z"
git push origin vX.Y.Z
```

The release workflow:
1. Runs `go mod tidy`
2. Syncs embedded schemas
3. Builds binaries for all platforms
4. Creates archives and checksums
5. Publishes to GitHub Releases

### Supported Platforms

| OS      | Architecture |
|---------|--------------|
| macOS   | amd64, arm64 |
| Linux   | amd64, arm64 |
| Windows | amd64, arm64 |

## Verifying Downloads

Each release includes `checksums.txt` with SHA256 hashes.

```bash
# Download binary and checksums (example: v1.0.1)
curl -LO https://github.com/justyn-clark/small-protocol/releases/download/v1.0.1/small-protocol_1.0.1_Darwin_arm64.tar.gz
curl -LO https://github.com/justyn-clark/small-protocol/releases/download/v1.0.1/checksums.txt

# Verify
sha256sum -c checksums.txt --ignore-missing
```

On macOS without `sha256sum`:

```bash
shasum -a 256 -c checksums.txt --ignore-missing
```

## Version Injection

Release binaries embed version information via ldflags:

| Variable | Source |
|----------|--------|
| Version  | Git tag (e.g., `1.0.0`) |
| Commit   | Git commit SHA |
| Date     | Build timestamp (RFC3339) |

Verify with:

```bash
small version
```

Expected output:

```text
small v1.0.0
Supported spec versions: ["1.0.0"]
```

## Snapshot Builds

Development builds use the snapshot template:

```
{{ .Version }}-dev+{{ .ShortCommit }}
```

Example: `v0.0.0-dev+abc1234`

To create a snapshot:

```bash
goreleaser release --snapshot --clean
```

## GoReleaser Configuration

The release is configured in `.goreleaser.yaml`. Key settings:

- **version**: GoReleaser v2 format
- **ldflags**: Inject version into `internal/version` package
- **archives**: tar.gz for Unix, zip for Windows
- **checksums**: SHA256 in `checksums.txt`

## Pre-release Checklist

Before tagging a release:

1. Ensure `make verify` passes
2. Ensure working tree is clean (`git status`)
3. Review CHANGELOG or commit history since last release
4. Confirm schema files are current in `internal/specembed/schemas/`

## Release Body Policy

- Keep the `v1.0.0` release body as the long-form launch narrative.
- Keep `v1.0.1` as a short patch note and link back to `v1.0.0` for full launch context.
- For later patch releases, keep notes concise and link to the relevant foundational release context when needed.

## Post-Release Package Verification

Run these after publishing package index updates:

### Homebrew

```bash
brew tap justyn-clark/tap
brew install small
small version
```

### Scoop

```bash
scoop bucket add justyn-clark https://github.com/justyn-clark/scoop-bucket
scoop install small
small.exe --help
```
