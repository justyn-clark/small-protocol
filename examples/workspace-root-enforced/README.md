# Workspace Root Enforcement Example

This example demonstrates `small verify` workspace scope enforcement.

## Workspace Metadata

The `.small/workspace.small.yml` file contains:

```yaml
small_version: "1.0.0"
kind: "examples"  # Marks this as an example workspace
```

## Workspace Scopes

The `--workspace` flag controls which workspaces pass verification:

| Scope | Description |
|-------|-------------|
| `root` | Only repo-root workspaces (default) |
| `examples` | Only example workspaces |
| `any` | Any workspace kind |

## Verifying the Example

```bash
# From repo root - passes because kind is "examples"
small verify --dir examples/workspace-root-enforced --workspace examples

# This would fail with --workspace root
small verify --dir examples/workspace-root-enforced --workspace root
# Error: Workspace validation failed: invalid workspace kind "examples"
```

## When to Use Workspace Scopes

- **CI pipelines**: Use `--workspace root` to verify the main workspace
- **Example verification**: Use `--workspace examples` to verify committed examples
- **Development**: Use `--workspace any` to bypass scope checks

## Key Behavior

The workspace scope ensures that:
1. Example workspaces are not accidentally treated as the main workspace
2. CI pipelines can distinguish between different workspace types
3. Commands like `plan`, `apply`, `reset` are restricted to appropriate workspace types
