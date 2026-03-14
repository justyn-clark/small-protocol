# Changelog

All notable changes to the SMALL protocol and tooling will be documented in this file.

This project follows a protocol-first versioning model:
- Protocol versions describe artifact contracts and semantics.
- Tooling versions may evolve independently, but always declare supported protocol versions.

---

## [v1.0.9] - 2026-03-14

### Status
Stable patch release

### Changed
- Published npm package surface now includes the package-specific README, corrected Apache-2.0 license metadata, and stronger discovery metadata.
- Installation docs and maintainer release examples now point at `v1.0.9` where a pinned version example is shown.

### Notes
- This is a docs-and-distribution hygiene release. No protocol contract or runtime behavior changed from `v1.0.8`.
- The purpose of this cut is to update the public npm package page, which cannot be changed in-place for an already-published version.

---

## [v1.0.8] - 2026-03-14

### Status
Stable patch release

### Changed
- Canonical runtime lineage locations are now `.small-runs/` and `.small-archive/`.
- Legacy `.small/archive/` and `.small/runs/` layouts can be repaired with `small fix --runtime-layout`.
- Strict mode now aligns with the canonical runtime layout defaults.

### Notes
- This is behavioral hardening and migration support, not a protocol contract change.
- Existing repos with legacy runtime lineage stores should run `small fix --runtime-layout` before relying on strict layout validation.

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
