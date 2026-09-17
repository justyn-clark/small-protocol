# Durable session walkthrough

This is a real, completed **SMALL v2.0.0 session-profile workspace**. It was
generated with the released `small v1.1.0` CLI and committed so you can inspect
the exact state behind the workflow.

<p align="center">
  <a href="../../docs/media/small-v1-1-0-session-demo.mp4">
    <img src="../../docs/media/small-v1-1-0-session-demo.gif" alt="SMALL durable-session terminal demonstration" width="720">
  </a>
</p>

## Take the 60-second tour

Run these commands from the repository root:

```bash
small check --strict --dir examples/durable-session --workspace examples
small status --json --dir examples/durable-session
small session list --json --dir examples/durable-session
small reconstruct --json --dir examples/durable-session --task task-2
```

You should find:

- profile `2.0.0` in `solo` mode;
- two closed sessions: the loss-preserving v1 import and the implementation;
- `task-2` reconstructed as `completed`;
- a captured command receipt plus an explicit checkpoint receipt, both
  `verified`; and
- a narrative handoff with no unresolved conflict.

The command's visible result is
[`src/resume-ready.txt`](src/resume-ready.txt). The evidence that explains how
it got there lives in `.small/`.

## The story recorded in this workspace

1. A valid v1 workspace was migrated explicitly with a reviewed migration
   preview and a stable shared namespace.
2. SMALL created an immutable import containing every original v1 byte.
3. A new implementation session started in the default `solo` mode.
4. `small apply` ran the bounded command that created the output file and
   recorded its outcome.
5. `small checkpoint` separately accepted the task with human-readable
   evidence.
6. `small session close` preserved a narrative handoff for the next operator.

That fourth-to-fifth step matters: a successful process exit is evidence, not
automatic proof that the work meets its acceptance criteria.

## What to inspect

| Path | Meaning |
|---|---|
| [`.small/profile.json`](.small/profile.json) | Project identity, lineage, initial mode, and policy revision |
| [`.small/policy/`](.small/policy/) | Human-owned intent and constraints |
| [`.small/imports/v1/`](.small/imports/v1/) | Original v1 state plus its migration manifest |
| [`.small/sessions/`](.small/sessions/) | Immutable writer descriptors |
| [`.small/events/`](.small/events/) | Authoritative lifecycle, command, checkpoint, and handoff events |
| [`.small/receipts/`](.small/receipts/) | Typed evidence receipts verified during reconstruction |

Generated views are intentionally absent. `small status`, `small reconstruct`,
and `small session list` derive them from authoritative state.

## Continue without mutating the example

Copy the directory to a disposable location before starting another session:

```bash
cp -R examples/durable-session /tmp/small-durable-session
small session start --dir /tmp/small-durable-session --label exploration
```

The committed workspace remains a stable, strict-valid reference. For the full
operational contract, read the
[SMALL 2.0.0 session profile](../../docs/session-profile-v2.md).
