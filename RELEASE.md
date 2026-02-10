# Release Checklist

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
tar -xzf dist/small_1.0.0_darwin_arm64.tar.gz
./small version
```

## 3) Tag release

```bash
git tag -a v1.0.0 -m "SMALL Protocol v1.0.0"
git push origin v1.0.0
```

## 4) Verify checksums

```bash
cat dist/sha256sums.txt
shasum -a 256 dist/small_1.0.0_*.tar.gz
```
