# @small-protocol/small

This package installs the native SMALL binary for your platform and exposes it as `small`.

## Install

```bash
npm i -g @small-protocol/small
small --version
```

## Behavior

- Maps npm version `X.Y.Z` to GitHub release tag `vX.Y.Z`
- Downloads `checksums.txt` and platform archive from GitHub Releases
- Verifies SHA256 before extracting
- Stores the binary in `vendor/small`
