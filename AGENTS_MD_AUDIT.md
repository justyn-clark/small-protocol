# AGENTS.md Audit Report

## Overview
Auditing the provided AGENTS.md against current SMALL CLI doctrine and workflows.

## Strengths (Correct Patterns)
✅ "SMALL governs state, not execution" - correct framing
✅ "Never mark tasks complete with plan toggles" - correct
✅ "Completion is only via `small checkpoint`" - correct
✅ Enforces `small check --strict` before handoff - correct
✅ "Never assume chat memory; always read state from disk and CLI" - correct
✅ Proper use of evidence in checkpoint commands
✅ Correct forbidden list (no timestamp overrides, no plan --done for agents)

## Issues Found

### Issue 1: Missing Self-Healing Commands
**Severity**: Major
**Category**: CLI completeness
**Location**: "Agent-safe command set" section
**Issue**: Missing `small start` and `small fix` commands
**Fix**: Add to agent-safe command set:
```
### Self-healing

- `small start` (repair handoff state)
- `small fix --versions` (normalize small_version formatting)
- `small doctor` (diagnose workspace issues)
```

### Issue 2: Confusing Section Name
**Severity**: Minor
**Category**: Clarity
**Location**: "Ralph loop" heading
**Issue**: "Ralph loop" is unclear and non-standard terminology
**Fix**: Rename to "Work Loop" or "Execution Loop"

### Issue 3: Session Start Missing Self-Healing
**Severity**: Major
**Category**: Workflow completeness
**Location**: "Session start" section
**Issue**: Doesn't mention using `small start` to repair corrupted workspace state
**Fix**: Add guidance:
```
If the workspace is corrupted or missing replayId:

```bash
small start  # Self-heal handoff state
small check --strict
```
```

### Issue 4: Missing Explicit Init vs Start Guidance
**Severity**: Minor
**Category**: Clarity
**Location**: Document scope
**Issue**: Doesn't clarify when to use `small init` vs `small start`
**Fix**: Add to artifact roles or session start:
- `small init` - Create new workspace from scratch
- `small start` - Repair existing workspace or ensure handoff state

### Issue 5: Doctor Command Context
**Severity**: Minor
**Category**: Clarity
**Location**: "Agent-safe command set" under Read-only
**Issue**: `small doctor` is listed but doesn't explain its diagnostic purpose
**Fix**: Add context: `small doctor` (diagnose workspace issues - read-only)

## Recommended Changes Summary

1. **Add self-healing section** to agent-safe command set with `start`, `fix`, `doctor`
2. **Rename "Ralph loop"** to "Work Loop" or "Execution Loop"
3. **Add self-healing guidance** to session start section
4. **Add init vs start clarification** to artifact roles or early in document
5. **Enhance doctor description** to emphasize diagnostic use

## Overall Assessment

The AGENTS.md is **fundamentally sound** with correct state governance framing and proper checkpoint/check --strict workflows. The issues are primarily about **completeness** (missing self-healing commands) and **clarity** (naming, context). No major doctrinal drift detected.

**Recommended Action**: Apply fixes above to create v2 with complete CLI command coverage.
