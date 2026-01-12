# Verify evidence gate

This example shows the new `small verify` rule that enforces progress entries for every completed plan task. The `.small/` contents reproduce the workspace state after this rule landed.

## Reproducing the failure

1. Leave the `verify-evidence-rule` task marked as `completed` in `plan.small.yml`.
2. Remove (or omit) the corresponding entry from `progress.small.yml`.
3. Run:

```
small verify
```

`small verify` exits with code 1 and prints:

```
Verification failed with 1 error(s):
  Invariant [progress.small.yml]: progress entries missing for completed plan tasks: verify-evidence-rule
```

## Fix

Add at least one `progress.small.yml` entry that references the completed task before rerunning `small verify`. Each completed task must include a progress entry so the verification gate passes.
