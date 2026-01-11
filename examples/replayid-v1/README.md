# ReplayId v1 — Deterministic Handoff Example

This example demonstrates a complete SMALL execution resulting in a
deterministic `replayId`.

What it shows:

- Automatic replayId generation (`source: auto`)
- Deterministic hashing of intent + plan + constraints
- A full execution trace captured via `.small/` artifacts
- A valid handoff suitable for audit, resume, or verification

Notes:

- This `.small/` directory is a **committed example**, not a live workspace.
- The root `.small/` directory remains ignored and is used only during active runs.
- This pattern is recommended for documenting protocol behavior in spec repos.
