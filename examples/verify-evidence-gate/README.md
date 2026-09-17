# Lab: the evidence gate

**Question:** Is changing a task to `completed` enough to claim the work is
done?

No. This v1 workspace demonstrates the invariant that every completed plan task
must have a matching progress entry with evidence.

## Inspect the passing case

From the repository root:

```bash
small check --strict --dir examples/verify-evidence-gate --workspace examples
```

Compare the task in
[`plan.small.yml`](.small/plan.small.yml) with its matching entry in
[`progress.small.yml`](.small/progress.small.yml). The shared task ID is the
auditable link between the claim and its evidence.

## Reproduce the gate safely

Copy the example to a disposable directory before experimenting:

```bash
cp -R examples/verify-evidence-gate /tmp/verify-evidence-gate
small check --strict --dir /tmp/verify-evidence-gate --workspace any
```

If the completed task's progress entry is absent, strict verification fails and
names the task whose evidence is missing. Restore the copy or record evidence
through the CLI; never patch a live `.small/` history by hand.

The rule is intentionally simple: completion must be backed by durable,
task-addressable evidence.
