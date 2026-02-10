# SMALL Protocol v1.0.0 Specification

**SMALL** (Schema, Manifest, Artifact, Lineage, Lifecycle) is a protocol for durable, agent-legible project state. This specification defines version 1.0.0 of the protocol.

## Versioning

- **Protocol version**: `1.0.0`
- **Version format**: `small_version: "1.0.0"` (string, not number)
- All canonical artifacts MUST include `small_version: "1.0.0"` as a required field

## Canonical Artifacts

SMALL v1.0.0 defines exactly five canonical artifacts, all stored as YAML files in the `.small/` directory:

1. **`.small/intent.small.yml`** - Human-owned project intent/goals
2. **`.small/constraints.small.yml`** - Human-owned constraints/requirements
3. **`.small/plan.small.yml`** - Agent-owned execution plan
4. **`.small/progress.small.yml`** - Agent-owned progress tracking
5. **`.small/handoff.small.yml`** - Shared resume entrypoint

### File Naming Convention

All canonical files follow the pattern: `<name>.small.yml` where `<name>` is one of: `intent`, `constraints`, `plan`, `progress`, `handoff`.

## Ownership Rules

### Human-Owned Artifacts

- **`intent.small.yml`** and **`constraints.small.yml`** are human-owned
- Agents MUST NOT modify these files
- Agents MAY read these files to understand project context
- Humans are responsible for maintaining these files

### Agent-Owned Artifacts

- **`plan.small.yml`** and **`progress.small.yml`** are agent-owned
- Agents write and update these files during execution
- Humans MAY read these files to understand agent state
- Humans SHOULD NOT manually edit these files (they may be regenerated)

### Shared Artifacts

- **`handoff.small.yml`** is shared between humans and agents
- Both humans and agents may write to this file
- This file is the primary resume entrypoint for agent sessions
- Agents MUST read `handoff.small.yml` to resume work

## Invariants

The following invariants are non-negotiable and MUST be enforced by all SMALL v1.0.0 implementations:

### 1. Secrets Never Stored

- No secrets, API keys, passwords, or sensitive tokens may be stored in any canonical artifact
- Implementations SHOULD detect and warn about potential secrets (heuristic-based)
- Secrets MUST be managed outside the `.small/` directory

### 2. Progress Entries Must Be Verifiable

- Every entry in `progress.small.yml` MUST include at least one evidence field:
  - `evidence` (string or object)
  - `verification` (string or object)
  - `command` (string - command that was executed)
  - `test` (string or object - test that was run)
  - `link` (string - URL to external evidence)
  - `commit` (string - git commit hash)
- Evidence fields enable verification of progress claims

### 3. Plan is Disposable; Progress is Not

- `plan.small.yml` may be regenerated or replaced at any time
- `progress.small.yml` is append-only and MUST preserve all historical entries
- Progress entries MUST NOT be deleted or modified after creation
- New progress entries are appended to the `entries` array

### 4. Handoff is the Only Resume Entrypoint

- Agents resuming work MUST read `handoff.small.yml` first
- `handoff.small.yml` contains the current plan and recent progress
- Agents MUST NOT attempt to reconstruct state from `plan.small.yml` and `progress.small.yml` directly
- The handoff file provides a deterministic snapshot for resumption

### 5. Version Consistency

- All canonical artifacts MUST have `small_version: "1.0.0"` (exact string match)
- Version mismatches MUST cause validation to fail
- Future protocol versions will use different version strings

### 6. Replay Semantics and Bootstrap Entries

- Bootstrap progress entries such as `meta/init` and `meta/accept-*` are allowed without `replayId`
- Once a plan exists, execution state is bound to a run replay id
- Current run identity is stored in `.small/workspace.small.yml` at `run.replay_id`
- ReplayId-scoped strict checks apply to run-bound entries; bootstrap entries remain valid historical setup records

### 7. Progress Mode Semantics

- Signal mode is the default progress behavior
- Audit mode is opt-in via `SMALL_PROGRESS_MODE=audit`
- Implementations MUST NOT require audit mode for baseline compliance

### 8. Strict Layout Boundary (S4)

- Under `.small/`, only canonical root files are allowed:
  - `intent.small.yml`
  - `constraints.small.yml`
  - `plan.small.yml`
  - `progress.small.yml`
  - `handoff.small.yml`
  - `workspace.small.yml`
- Strict mode MUST reject unexpected files or subdirectories under `.small/`
- Ephemeral operational data belongs in `.small-cache/` (outside `.small/`)

## Backwards Compatibility

- **v1.0.0 is the first stable release**
- v0.1 is deprecated and not supported by current tooling
- Future versions (v1.1.0, v2.0.0, etc.) will maintain compatibility within major version
- Breaking changes will increment the major version number

## Extension Mechanism

- Non-canonical files may be placed under `.small/ext/` or ignored by v1.0.0 implementations
- Extensions MUST NOT conflict with canonical file semantics
- Extensions MUST NOT be required for basic protocol compliance
- Implementations MAY support extensions but MUST NOT fail if extensions are absent

## AGENTS.md Semantics

SMALL tooling MAY generate an `AGENTS.md` file to provide advisory documentation for AI agents.

### Key Properties

- **AGENTS.md is advisory documentation**, not canonical state
- `.small/` artifacts always take precedence over AGENTS.md guidance
- Presence or absence of AGENTS.md does not affect protocol validity
- AGENTS.md is NOT stored in `.small/` (it lives at repository root)

### Bounded Block Format

When SMALL tooling writes to an existing AGENTS.md file, it MUST use a bounded block:

```markdown
<!-- BEGIN SMALL HARNESS v1.0.0 -->
# SMALL Execution Harness
...content...
<!-- END SMALL HARNESS v1.0.0 -->
```

Rules for bounded blocks:

- Only content within the SMALL harness block is tool-managed
- Content outside the block MUST NOT be modified by SMALL tooling
- If a block exists, it may be replaced in-place
- The version in the markers MUST match the protocol version
- Multiple SMALL blocks in a single file are an error

### Composition Modes

SMALL implementations MAY support the following modes for handling existing AGENTS.md:

- **append**: Add SMALL block after existing content
- **prepend**: Add SMALL block before existing content
- **overwrite**: Replace entire file with SMALL block only

### Non-Goals

- SMALL MUST NOT merge prose from multiple sources
- SMALL MUST NOT parse or interpret other agents' instructions
- SMALL MUST NOT infer intent from existing AGENTS.md content

## Schema Validation

All canonical artifacts MUST validate against their corresponding JSON Schema:

- `intent.small.yml` -> `intent.schema.json`
- `constraints.small.yml` -> `constraints.schema.json`
- `plan.small.yml` -> `plan.schema.json`
- `progress.small.yml` -> `progress.schema.json`
- `handoff.small.yml` -> `handoff.schema.json`

Schemas are defined using JSON Schema Draft 2020-12.

## File Format

- All canonical artifacts are YAML files (`.yml` extension)
- YAML MUST be valid and parseable
- YAML is converted to JSON in-memory for schema validation
- Implementations SHOULD preserve YAML formatting when writing files

## Implementation Requirements

A SMALL v1.0.0 implementation MUST:

1. Validate all canonical artifacts against their schemas
2. Enforce all invariants listed above
3. Support reading and writing all five canonical artifacts
4. Provide clear error messages for validation failures
5. Support the `small_version` field in all artifacts

## Reference Implementation

The reference CLI implementation (`cmd/small/`) demonstrates a complete SMALL v1.0.0 implementation and serves as the authoritative validator.
