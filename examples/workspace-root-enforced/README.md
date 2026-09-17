# Lab: workspace scope

**Question:** How can CI distinguish a published example from the live SMALL
state at a repository root?

This v1 fixture declares `kind: examples` in
[`workspace.small.yml`](.small/workspace.small.yml). The scope flag makes that
boundary explicit.

## See both outcomes

From the repository root:

```bash
# Accepted: this command expects an example workspace.
small check --strict --dir examples/workspace-root-enforced --workspace examples

# Rejected: this command expects live repository-root state.
small check --strict --dir examples/workspace-root-enforced --workspace root
```

The second command should fail with a workspace-kind mismatch. That is the
feature: sample state cannot silently masquerade as the repository's active
run.

## Scope guide

| Scope | Intended use |
|---|---|
| `root` | The live SMALL workspace at a repository root |
| `examples` | Committed reference and regression workspaces |
| `any` | Explicitly bypass the kind distinction for controlled tooling |

Use the narrowest scope that matches the operation. In CI for this gallery,
[`scripts/verify-examples.sh`](../../scripts/verify-examples.sh) selects
`examples` and requires strict validation for every discovered workspace.
