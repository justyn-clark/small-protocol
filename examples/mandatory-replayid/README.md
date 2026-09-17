# Lab: mandatory replay ID

**Question:** What stops a handoff from becoming an anonymous blob of state?

This valid v1 workspace demonstrates two enforcement boundaries:

1. every completed plan task has matching progress evidence; and
2. every handoff has a schema-valid replay ID.

## Run the passing case

From the repository root:

```bash
small check --strict --dir examples/mandatory-replayid --workspace examples
```

Expected result: `Check passed`.

## Explore the failures safely

Copy the example first; do not edit committed SMALL state:

```bash
cp -R examples/mandatory-replayid /tmp/mandatory-replayid
```

In the copy, removing a completed task's progress entry makes strict checking
report the missing task evidence. Removing the `replayId` object from the
handoff makes schema validation reject the handoff.

Use the CLI to repair a handoff instead of hand-editing it:

```bash
small handoff --dir /tmp/mandatory-replayid --workspace any \
  --summary "Rebuilt from canonical run state"
small check --strict --dir /tmp/mandatory-replayid --workspace any
```

The point is not the YAML syntax. It is that a future operator can verify both
what finished and which run-defining state the handoff belongs to.
