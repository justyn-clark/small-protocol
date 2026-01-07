# Releasing

This document describes how SMALL Protocol releases are created and verified.

## Release Process

Releases are automated via GitHub Actions when a version tag is pushed.

### Creating a Release

```bash
git tag -a v1.0.0 -m "SMALL v1.0.0"
git push origin v1.0.0
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
# Download binary and checksums
curl -LO https://github.com/justyn-clark/small-protocol/releases/download/v1.0.0/small-protocol_1.0.0_Darwin_arm64.tar.gz
curl -LO https://github.com/justyn-clark/small-protocol/releases/download/v1.0.0/checksums.txt

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
