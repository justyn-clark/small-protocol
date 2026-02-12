# Changelog

All notable changes to the SMALL protocol and tooling will be documented in this file.

This project follows a protocol-first versioning model:
- Protocol versions describe artifact contracts and semantics.
- Tooling versions may evolve independently, but always declare supported protocol versions.

---

## [v1.0.0] — 2026-01-04

### Status
- **Stable / Current**

### Changed
- SMALL v1.0.0 is now the authoritative protocol version
- v0.1 is deprecated and preserved for historical reference only
- All tooling now targets v1.0.0 exclusively
- `small_version` must be exactly `"1.0.0"` in all artifacts

### Notes
- v1.0.0 defines the **canonical artifact contract** for agent-legible project continuity.
- Implementations MUST target v1.0.0.

---

## [v0.1.0] — 2025-12-21 (DEPRECATED)

### Status
- **Deprecated** — preserved for historical reference only

### Added (historical)
- Initial SMALL protocol specification
- JSON Schemas for all artifacts
- Reference Go CLI (`small`)

### Notes
- v0.1 is no longer supported by tooling.
- See `spec/small/v0.1/DEPRECATED.md` for deprecation notice.
