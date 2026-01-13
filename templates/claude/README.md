# Claude Agent Template (SMALL Protocol)

This directory contains a template Claude agent configuration for working
on the SMALL Protocol spec and CLI repository.

## How to use

If you use Claude tooling:

cp -r templates/claude/.claude .claude

Do NOT commit .claude/ at repo root. It is personal execution state.

## What this template enforces

- SMALL-first workflow (intent, constraints, plan, progress, handoff)
- Correct handling of root .small/ vs example .small/
- CLI-safe Go changes
- Deterministic behavior (ReplayId, timestamps, invariants)
- No speculative edits or undocumented behavior

This template is tailored specifically for this repository.
