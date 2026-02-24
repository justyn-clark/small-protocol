# Installation

## Supported platforms

- macOS: amd64, arm64
- Linux: amd64, arm64

Release asset names are stable:

- `small-vX.Y.Z-darwin-amd64.tar.gz`
- `small-vX.Y.Z-darwin-arm64.tar.gz`
- `small-vX.Y.Z-linux-amd64.tar.gz`
- `small-vX.Y.Z-linux-arm64.tar.gz`
- `checksums.txt`

## Option 1: curl installer (recommended)

Install latest:

```bash
curl -fsSL https://smallprotocol.dev/install.sh | bash
```

Pin to a version:

```bash
curl -fsSL https://smallprotocol.dev/install.sh | bash -s -- --version v1.0.2
```

Install behavior:

- Downloads release metadata from GitHub API
- Downloads the platform archive and `checksums.txt`
- No compilation required. Pre-built binaries are downloaded and checksum verified.
- Verifies SHA256 before extraction and install
- Installs to `~/.local/bin/small` by default (no sudo)

Optional flags:

- `--system`: install to `/usr/local/bin/small` (uses sudo if needed)
- `--dir PATH`: custom install directory
- `--release-json PATH`: use local release metadata (testing only)
- `--dry-run`: print resolved release/install details only

## Option 2: npm global install

```bash
npm i -g @small-protocol/small
small --version
```

npm install behavior:

- Maps npm version `X.Y.Z` to release tag `vX.Y.Z`
- Fetches tagged release metadata from GitHub API
- Downloads platform archive and `checksums.txt`
- Verifies SHA256 before extracting `small` into the package `vendor/` directory

If the binary is missing after install:

```bash
npm rebuild @small-protocol/small
```

## Option 3: Go install

```bash
go install github.com/justyn-clark/small-protocol/cmd/small@latest
small --version
```

## Uninstall

curl default install:

```bash
rm -f ~/.local/bin/small
```

curl system install:

```bash
sudo rm -f /usr/local/bin/small
```

npm global install:

```bash
npm uninstall -g @small-protocol/small
```
