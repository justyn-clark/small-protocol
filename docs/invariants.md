# Invariants (v1.0.0)

The following are non-negotiable:

- `intent` and `constraints` MUST be human-owned
- `plan` and `progress` MUST be agent-owned
- `progress` MUST be append-only and evidence-backed
- Schema validation MUST pass
- `small_version` MUST equal `"1.0.0"`

Violations are hard errors.

## Ownership Rules

| Artifact    | Owner  |
|-------------|--------|
| intent      | Human  |
| constraints | Human  |
| plan        | Agent  |
| progress    | Agent  |
| handoff     | System |

Human-owned artifacts define the work and its boundaries.
Agent-owned artifacts record proposed and executed work.
System-owned artifacts capture state for resumption.

## Append-Only Requirement

The `progress.small.yml` artifact is append-only.

- Entries may not be modified after creation
- Entries may not be deleted
- Each entry must include evidence of execution

## Version Constraint

All artifacts must declare:

```yaml
small_version: "1.0.0"
```

Artifacts with missing or mismatched versions will fail validation.

## Schema Validation

All artifacts must validate against the authoritative schemas located at:

```text
spec/small/v1.0.0/schemas/
```

Schema violations are hard errors and will block execution.
