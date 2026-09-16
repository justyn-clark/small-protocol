# Release checklist

## Protocol compatibility gate

- Confirm `small version` advertises `1.0.0` and `2.0.0`.
- Run the complete v1 suite and verify an unmigrated fixture has no diff.
- Run migration determinism, stale-input, idempotence, and every fault boundary.
- Run the local two-clone session proof and 10k/100k reducer measurements.
- Confirm an older supported v1 CLI only fails read-only against a migrated
  fixture and does not change its authoritative digest.
- Run `go test -race ./...`, `go vet ./...`, format/schema gates, built selftest,
  and `scripts/verify.sh` in a clean candidate containing the full patch.

## 1) Prepare and tag

1. Ensure CI is green on main.
2. Confirm version bump and changelog updates.
3. Create and push the tag:

```bash
git tag -a vX.Y.Z -m "SMALL vX.Y.Z"
git push origin vX.Y.Z
```

## 2) Verify GitHub release assets

1. Confirm release has:
   - `small-vX.Y.Z-darwin-amd64.tar.gz`
   - `small-vX.Y.Z-darwin-arm64.tar.gz`
   - `small-vX.Y.Z-linux-amd64.tar.gz`
   - `small-vX.Y.Z-linux-arm64.tar.gz`
   - `small-vX.Y.Z-windows-amd64.zip`
   - `small-vX.Y.Z-windows-arm64.zip`
   - `checksums.txt`
2. Verify checksum file includes each platform archive.

## 3) Verify installer paths

1. Curl installer:

```bash
curl -fsSL https://smallprotocol.dev/install.sh | bash -s -- --version vX.Y.Z --dir /tmp/smallbin
PATH=/tmp/smallbin:$PATH small version
```

2. npm package:

```bash
npm i -g @small-protocol/small@X.Y.Z
small version
```

## 4) npm publish (OIDC-only)

1. Set `packages/npm/package.json` version to `X.Y.Z`.
2. Ensure tag is `vX.Y.Z`.
3. Configure npm Trusted Publishing (OIDC) for `@small-protocol/small` with this GitHub repo/workflow as trusted publisher.
4. Publish workflow must run with Node 24 and npm 11.10.1.
5. Publish with provenance:

```bash
cd packages/npm
npm publish --provenance --access public
```
