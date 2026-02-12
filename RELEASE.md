# Release Checklist

## Tag policy

- Keep `v1.0.0` forever (first stable contract tag).
- Keep `v1.0.1` forever (install/packaging patch).
- Never move public tags. If correction is needed, release next patch (for example `v1.0.2`).

## 1) Build release artifacts

```bash
bash scripts/build-release.sh
```

Artifacts are written to `dist/` and checksums to `dist/sha256sums.txt`.

## 2) Smoke tests

```bash
go test ./...
go run ./cmd/small version
go run ./cmd/small selftest
```

Optional local binary smoke test:

```bash
tar -xzf dist/small_X.Y.Z_darwin_arm64.tar.gz
./small version
```

## 3) Tag release

```bash
git tag -a vX.Y.Z -m "SMALL Protocol vX.Y.Z"
git push origin vX.Y.Z
```

## 4) Verify checksums

```bash
cat dist/sha256sums.txt
shasum -a 256 dist/small_X.Y.Z_*.tar.gz
```
