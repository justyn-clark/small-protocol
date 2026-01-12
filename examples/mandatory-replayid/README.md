# Mandatory ReplayId Example

This example demonstrates two key `small verify` invariants:

1. **Evidence Gate**: Completed tasks in `plan.small.yml` must have corresponding entries in `progress.small.yml`
2. **ReplayId Requirement**: `handoff.small.yml` must include a valid `replayId`

## Verifying the Example

```bash
# From repo root
small verify --dir examples/mandatory-replayid
# Expected: Verification passed
```

## Reproducing Failures

### Evidence Gate Failure

1. Edit `.small/progress.small.yml` and remove all entries
2. Run `small verify --dir examples/mandatory-replayid`

Expected output:
```
Verification failed with 1 error(s):
  Invariant [progress.small.yml]: progress entries missing or invalid for completed plan tasks: demonstrate-evidence-gate, demonstrate-replayid
```

### ReplayId Failure

1. Edit `.small/handoff.small.yml` and remove the `replayId` block
2. Run `small verify --dir examples/mandatory-replayid`

Expected output:
```
Verification failed with 1 error(s):
  handoff.small.yml must include replayId (use 'small handoff' to generate)
```

## CI Integration

Add to your CI workflow:

```yaml
- name: Verify SMALL artifacts
  run: small verify --ci
```

This ensures that:
- Completed tasks have auditable progress evidence
- Session continuity is tracked via replayId
- Schema validation passes
- Ownership rules are enforced
