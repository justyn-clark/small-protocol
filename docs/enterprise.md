# Enterprise Integration

## Stateless by Design

SMALL is stateless by design.

It runs no servers, daemons, or background processes.
Lineage and audit are captured by exporting artifacts into existing systems.

## Integration Patterns

| Environment | Pattern                                                   |
|-------------|-----------------------------------------------------------|
| Git         | Commit `.small/` for immutable history                    |
| CI/CD       | Archive progress and status output                        |
| Audit       | Ingest artifacts into Splunk, ELK, Datadog, or a database |

## Git Integration

Commit the `.small/` directory to preserve state history:

```bash
git add .small/
git commit -m "Record SMALL state"
```

Each commit provides an immutable snapshot of project state at that point in time.

## CI/CD Integration

Archive SMALL artifacts as build outputs:

```bash
small status --json > small-status.json
cp -r .small/ artifacts/
```

This enables post-hoc audit and debugging of automated workflows.

## Audit Integration

SMALL artifacts are machine-readable YAML (v1) or JSON (v2) files suitable for ingestion into log aggregation and observability systems.

V1 export patterns:

- Stream `progress.small.yml` entries to a log pipeline
- Index `handoff.small.yml` for checkpoint recovery
- Store full `.small/` snapshots in long-term storage

For v2, export the complete authoritative `.small` tree rather than selecting
only familiar v1 filenames. The tree includes immutable sessions, events,
resolutions, policy revisions, imports, and evidence receipts. `small run
snapshot` and `small archive` preserve that tree and its integrity metadata.

## What SMALL Does Not Replace

SMALL does not replace:

- IAM (Identity and Access Management)
- RBAC (Role-Based Access Control)
- Enforcement systems

SMALL makes execution legible and provable.
It describes constraints; it does not enforce them.
