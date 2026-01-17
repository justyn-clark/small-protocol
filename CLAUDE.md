# Role & Repository Context

You are working in small-protocol spec/cli repo.

# MANDATORY: SMALL Protocol Enforcement

**This project uses SMALL Protocol for all AI agent work. These rules are NON-NEGOTIABLE.**

## Required CLI Commands

All work MUST use the `small` CLI. NEVER use TodoWrite or other task tracking - use SMALL exclusively.

### At Session Start
```bash
small status                    # Check current project state
small plan --show               # Review existing plan (if any)
```

### Before Starting Work
```bash
small plan --add "Task title"   # Add task to plan.small.yml
# Manually edit .small/intent.small.yml to set intent and scope
# Manually edit .small/progress.small.yml to add progress entries
```

### During Work
```bash
small apply "<command>"         # Execute commands bounded by intent/constraints
small plan --done "<task-id>"   # Mark task completed in plan
# Manually add entries to .small/progress.small.yml for audit trail
```

### Progress Entry Format (manual edit)
```yaml
- task_id: "task-name"
  status: "completed"  # or "in_progress"
  timestamp: "2026-01-16T15:22:13.000000001Z"  # RFC3339Nano with fractional seconds
  evidence: "What was done"
  notes: "Additional context"
```

### At Session End or Handoff
```bash
small handoff                   # Generate handoff.small.yml for continuity
small archive                   # Archive run state if completing a milestone
```

### For Verification
```bash
small lint                      # Check for invariant violations
small validate                  # Validate all SMALL artifacts
small verify                    # CI/local enforcement gate
```

## Enforcement Rules

1. **NEVER skip SMALL commands** - Even after context compaction, these rules persist in CLAUDE.md
2. **NEVER use TodoWrite** - All task tracking goes through `small progress`
3. **ALWAYS run `small status`** at the start of any new response involving work
4. **ALWAYS run `small handoff`** before ending a session or when context is running low
5. **ALWAYS use `small apply`** for bounded command execution when available
6. **UPDATE progress.small.yml** after completing each discrete task, not in batches
7. **ALWAYS run `small lint`** before `small handoff` - lint checks invariants that validate misses
8. **ALWAYS run `small validate && small lint`** after manually editing any .small/*.yml files

## Timestamp Format

When manually editing progress.small.yml, timestamps MUST be RFC3339Nano format with fractional seconds:
- **Correct**: `2026-01-16T15:22:13.000000001Z`
- **Wrong**: `2026-01-16T15:22:13Z` (missing fractional seconds)

Timestamps must also be strictly increasing (each entry later than the previous).

## File Locations

- `.small/plan.small.yml` - Intent, approach, constraints, success criteria
- `.small/progress.small.yml` - Task list with status tracking
- `.small/handoff.small.yml` - Continuation context for next session
- `.small/runs/` - Archived run history

## Why This Matters

SMALL Protocol ensures:
- Durable state that survives context window limits
- Agent-legible project continuity across sessions
- Auditable history of AI work
- Clear intent/constraint boundaries for safe execution

**If you find yourself using TodoWrite or tracking tasks any other way, STOP and use `small progress` instead.**

This file provides guidance to AI agents when working with code in this repository.

## Relationship to SMALL Protocol

This repository uses SMALL for execution tracking and handoff.
Claude should treat `.small/` as authoritative execution state.

- Goals live in `intent.small.yml`
- Constraints live in `constraints.small.yml`
- Work happens through `plan.small.yml`
- Evidence is written to `progress.small.yml`
- Resume state is in `handoff.small.yml`
