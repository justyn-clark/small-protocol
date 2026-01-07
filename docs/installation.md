# Installation

## Requirements

- Go 1.22 or later (for source install)
- Or: download pre-built binaries (no Go required)

## Option 1: Download Pre-built Binaries

Download the latest release from [GitHub Releases](https://github.com/justyn-clark/small-protocol/releases).

```bash
# Example for macOS ARM64
curl -LO https://github.com/justyn-clark/small-protocol/releases/download/v1.0.0/small-protocol_1.0.0_Darwin_arm64.tar.gz
curl -LO https://github.com/justyn-clark/small-protocol/releases/download/v1.0.0/checksums.txt

# Verify checksum
shasum -a 256 -c checksums.txt --ignore-missing

# Extract and install
tar -xzf small-protocol_1.0.0_Darwin_arm64.tar.gz
sudo mv small /usr/local/bin/
small version
```

## Option 2 (Recommended for Go users): Install via Go

Install the `small` binary directly:

```bash
go install github.com/justyn-clark/small-protocol/cmd/small@latest
```

Verify installation:

```bash
command -v small && small version
```

Expected output:

```text
/Users/you/go/bin/small
small v1.0.0
Supported spec versions: ["1.0.0"]
```

### If `small` is not found

If `command -v small` returns nothing, your Go bin directory is not on PATH.

**zsh (macOS default):**

```bash
echo 'export PATH="$(go env GOPATH)/bin:$PATH"' >> ~/.zshrc
source ~/.zshrc
```

**bash:**

```bash
echo 'export PATH="$(go env GOPATH)/bin:$PATH"' >> ~/.bashrc
source ~/.bashrc
```

Then retry:

```bash
command -v small
small version
```

## Option 3: Build from Source

```bash
git clone https://github.com/justyn-clark/small-protocol.git
cd small-protocol
make small-build
```

Run directly:

```bash
./bin/small version
```

Optional global install:

```bash
sudo cp ./bin/small /usr/local/bin/small
small version
```
