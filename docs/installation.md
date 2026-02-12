# Installation

## Version Guidance

- Recommended install: `v1.0.1` (patch release).
- `v1.0.0` is the initial stable protocol release; see the [v1.0.0 release notes](https://github.com/justyn-clark/small-protocol/releases/tag/v1.0.0) for the full launch changelog.

## Requirements

- Go 1.22 or later (for Go-based install or source development)
- Or: download pre-built binaries (no Go required)

## Option 1: Download Pre-built Binaries (No Go Required)

Download artifacts from [GitHub Releases](https://github.com/justyn-clark/small-protocol/releases).

```bash
# Example for macOS ARM64 (v1.0.1)
curl -LO https://github.com/justyn-clark/small-protocol/releases/download/v1.0.1/small-protocol_1.0.1_Darwin_arm64.tar.gz
curl -LO https://github.com/justyn-clark/small-protocol/releases/download/v1.0.1/checksums.txt

# Verify checksum
shasum -a 256 -c checksums.txt --ignore-missing

# Extract and install
tar -xzf small-protocol_1.0.1_Darwin_arm64.tar.gz
sudo mv small /usr/local/bin/

# Verify
small version
```

## Option 2 (Recommended for Go Users): Install via Go

Install the `small` binary directly using Go:

```bash
go install github.com/justyn-clark/small-protocol/cmd/small@v1.0.1
```

This installs `small` into your Go bin directory (usually `~/go/bin`).

Verify installation:

```bash
command -v small
small version
```

If `small` is not found, your Go bin directory is not on `PATH`.

zsh (macOS default):

```bash
echo 'export PATH="$(go env GOPATH)/bin:$PATH"' >> ~/.zshrc
source ~/.zshrc
```

bash:

```bash
echo 'export PATH="$(go env GOPATH)/bin:$PATH"' >> ~/.bashrc
source ~/.bashrc
```

Then retry:

```bash
command -v small
small version
```

## Option 3: Build from Source (Development Only)

For contributors working on SMALL itself:

```bash
git clone https://github.com/justyn-clark/small-protocol.git
cd small-protocol

# Install the local development build into your Go bin
go install ./cmd/small

# Verify
command -v small
small version
```

Notes:

- This installs the development build into `$(go env GOPATH)/bin`.
- Avoid copying dev binaries into `/usr/local/bin` to prevent stale installs.
