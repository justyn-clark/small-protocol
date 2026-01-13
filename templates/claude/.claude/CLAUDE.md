Title: Claude Operating Guide - SMALL Protocol Repository

Absolute Rules
- Treat the SMALL spec plus CLI repo as the single source of truth for requirements.
- Keep all runtime state inside the root `.small/` directory; do not invent additional runtime files.
- Never invent or infer behavior that is not supported by existing files or explicit instructions.
- Favor testability: every change should be verifiable through CLI or spec checks.
- Preserve deterministic outputs (ReplayId, timestamps, invariants) in every edit.

How to Work (Required Flow)
1. Read the current SMALL state (spec, CLI files, and context) before proposing changes.
2. Execute work incrementally, reporting progress rather than waiting until the final step.
3. Enforce correctness via the CLI, ensuring no Go edits violate the CLI constraint.
4. Generate a clear handoff that documents what was changed and what remains.

Repo-specific knowledge
- Spec files live under `spec/` and must stay unchanged in this task.
- CLI implementation lives under `cmd/` and `internal/`; Go edits are forbidden here.
- Templates for Claude live under `templates/claude/` and may be copied for personal state.

Things You Must Not Do
- Do not modify Go source files.
- Do not alter spec schemas or docs site content in this work.
- Do not commit `.claude/` or `.small/` from the repo root.
- Do not add speculative behavior or make undocumented assumptions.
- Do not ignore the required SMALL-first workflow.

When Unsure: Stop and ask.
