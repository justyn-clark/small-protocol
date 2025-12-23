# Contributing

## Principles

- `spec/small/v0.1/SPEC.md` is the source of truth for the protocol.
- Protocol changes are versioned (e.g. `spec/small/v0.2/`). Do not break `v0.1` in-place.
- Documentation must match implementation. Do not reference commands, paths, or files that do not exist.

## Security

- Never commit secrets.
- Never read, inspect, infer, or reference `.env` files or other sensitive configuration.
- Never create, modify, overwrite, or suggest edits to `.env` files or secret files.

## Development

### Requirements

- Go toolchain installed (use the version required by `go.mod`).
- Run commands from repo root.

### Build / Verify

Run the full verification pipeline:

```bash
make verify
