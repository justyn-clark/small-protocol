Understood. Here is the entire file, corrected, with no stray text outside, returned as one single Markdown code block and with the typo fixed.

You can copy-paste this verbatim into docs/installation.md.

# Installation

## Requirements

- Go 1.22 or later (for Go-based install or source development)
- Or: download pre-built binaries (no Go required)

---
ssasasaaswnload Pre-built Binaries (No Go Required)

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

# Verify
small version

This installs small system-wide in /usr/local/bin.

⸻

Option 2 (Recommended for Go Users): Install via Go

Install the small binary directly using Go.
This is the preferred method for developers and provides a single source of truth.

go install github.com/justyn-clark/small-protocol/cmd/small@latest

This installs small into your Go bin directory (usually ~/go/bin).

Verify installation:

command -v small
small version

Expected output:

/Users/you/go/bin/small
small v1.0.0
Supported spec versions: ["1.0.0"]

If small is not found

If command -v small returns nothing, your Go bin directory is not on PATH.

zsh (macOS default):

echo 'export PATH="$(go env GOPATH)/bin:$PATH"' >> ~/.zshrc
source ~/.zshrc

bash:

echo 'export PATH="$(go env GOPATH)/bin:$PATH"' >> ~/.bashrc
source ~/.bashrc

Then retry:

command -v small
small version


⸻

Option 3: Build from Source (Development Only)

For contributors working on SMALL itself.

git clone https://github.com/justyn-clark/small-protocol.git
cd small-protocol

# Install the local development build into your Go bin
go install ./cmd/small

Verify:

command -v small
small version

Notes:
	•	This installs the development build into $(go env GOPATH)/bin.
	•	Avoid copying binaries into /usr/local/bin during development to prevent stale installs.
	•	make small-build may produce ./bin/small for inspection or CI use, but it is not the canonical install path.
