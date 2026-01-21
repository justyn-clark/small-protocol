# DRIFT_REPORT.md

## State vs Execution Language Drift

### Issue 1
**File**: `docs/FAQ.md`
**Line**: 70
**Severity**: Major
**Category**: Language
**Snippet**: `SMALL is an **execution protocol** for agent continuity.`
**Issue**: Misleading framing. SMALL is a STATE protocol, not an execution protocol. Term "execution protocol" implies orchestration.
**Recommended Correction**: Replace with "SMALL is a **state protocol** for agent continuity" or "SMALL is a **governance and continuity layer**"
**Needs Verification**: No

### Issue 2
**File**: `docs/FAQ.md`
**Line**: 73
**Severity**: Minor
**Category**: Language
**Snippet**: `Not a task runner: SMALL does not execute tasks. small apply records execution`
**Issue**: Correct language but contradicts calling SMALL an "execution protocol" at line 70
**Recommended Correction**: After fixing Issue 1, ensure consistency throughout the section
**Needs Verification**: No

---

## CLI Command Drift

### Issue 3
**File**: `docs/cli.md`
**Line**: Table at lines 4-18
**Severity**: Critical
**Category**: CLI
**Snippet**: Core Commands table missing `small start` and `small fix`
**Issue**: `small start` exists (confirmed in source) but is undocumented. `small fix` exists but only appears in docs/cli-guide.md, not the reference table.
**Recommended Correction**: Add two rows:
```
| `small start`    | Initialize or repair run handoff state |
| `small fix`      | Normalize formatting issues in-place    |
```
**Needs Verification**: Yes - verify `small start --help` for exact short description

### Issue 4
**File**: `docs/cli-guide.md`
**Line**: N/A (missing section)
**Severity**: Critical
**Category**: CLI
**Snippet**: No dedicated `small start` section
**Issue**: `small start` command exists but has no documentation in the CLI guide
**Recommended Correction**: Add complete `### small start` section with usage, flags, when to use, common errors
**Needs Verification**: Yes - run `small start --help` to document all flags and behavior

### Issue 5
**File**: `docs/agent-operating-contract.md`
**Line**: 26-28
**Severity**: Major
**Category**: CLI
**Snippet**: `small validate` and `small lint` shown separately
**Issue**: Entry protocol shows old two-command validation instead of unified `small check`
**Recommended Correction**: Update to show `small check` as the unified command, with optional note that validate/lint can be run separately
**Needs Verification**: No

### Issue 6
**File**: `docs/getting-started.md`
**Line**: 173-186
**Severity**: Minor
**Category**: CLI
**Snippet**: Shows `small validate`, `small lint`, `small lint --strict` separately
**Issue**: Missing mention of `small check` as the unified validation command
**Recommended Correction**: Add `small check` as primary recommendation, note that validate/lint are component commands
**Needs Verification**: No

### Issue 7
**File**: `docs/quickstart.md`
**Line**: 26
**Severity**: Minor
**Category**: CLI
**Snippet**: `small validate` only
**Issue**: Only shows validate, missing lint and check commands
**Recommended Correction**: Add `small check` or at minimum `small lint` to validation workflow
**Needs Verification**: No

---

## Agent Entrypoint Drift

### Issue 8
**File**: `docs/agent-operating-contract.md`
**Line**: 9-19
**Severity**: Major
**Category**: Agent
**Snippet**: "Always Read .small/ First" section lists manual file reading
**Issue**: Conceptual instruction is correct, but should mention using CLI commands for validation after reading
**Recommended Correction**: After file reading list, add: "After reading artifacts, validate the workspace: `small check`"
**Needs Verification**: No

### Issue 9
**File**: `docs/agent-operating-contract.md`
**Line**: 26-30
**Severity**: Major
**Category**: Agent
**Snippet**: Two separate commands `small validate` and `small lint`
**Issue**: Should use unified `small check` command
**Recommended Correction**: Replace with single `small check` command, optionally note component commands
**Needs Verification**: No

### Issue 10
**File**: `templates/claude/.claude/CLAUDE.md`
**Line**: 10-14
**Severity**: Minor
**Category**: Agent
**Snippet**: "Read the current SMALL state... before proposing changes"
**Issue**: This file is repo-specific (for Claude working ON small-protocol), not a general agent guide. But it lacks modern workflow (checkpoint, check --strict)
**Recommended Correction**: If this is intended as a general template, add checkpoint and check --strict workflow. If repo-specific only, clarify in header.
**Needs Verification**: No

### Issue 11
**File**: `docs/agent-operating-contract.md`
**Line**: 122-124
**Severity**: Minor (documentation clarity)
**Category**: Agent
**Snippet**: "Agents MUST NOT use small plan --done"
**Issue**: Not drift, but potential confusion - `plan --done` exists as a command but agents should prefer checkpoint
**Recommended Correction**: Clarify: "The `small plan --done` command exists for direct plan manipulation, but agents MUST use `small checkpoint` for completion to ensure atomic plan+progress updates"
**Needs Verification**: No

---

## First-Run UX Drift

### Issue 12
**File**: `docs/getting-started.md`
**Line**: Title and structure
**Severity**: Minor
**Category**: First-run
**Snippet**: N/A
**Issue**: Guide is good but missing reference to self-healing commands (start, doctor, fix) for troubleshooting
**Recommended Correction**: Add brief section "Troubleshooting Your First Workspace" mentioning `small doctor` for diagnosis and `small fix` for common issues
**Needs Verification**: No

### Issue 13
**File**: `docs/quickstart.md`
**Line**: 23-26
**Severity**: Minor
**Category**: First-run
**Snippet**: Only shows validate command
**Issue**: Incomplete validation workflow, missing lint/check
**Recommended Correction**: Add `small check` after validate
**Needs Verification**: No

---

## Examples and Snippets Drift

### Issue 14
**File**: `docs/cli.md`
**Line**: 87-93
**Severity**: Minor
**Category**: Examples
**Snippet**: `small verify --ci` and `small verify --strict` examples
**Issue**: Examples are correct, but missing context about `small check` as the unified enforcement command
**Recommended Correction**: Add note: "Note: `small check` provides unified validation, lint, and verify. Use `small verify` directly for CI-specific gates."
**Needs Verification**: No

### Issue 15
**File**: `docs/DEVELOPMENT.md`
**Line**: 109-118
**Severity**: Minor
**Category**: Examples
**Snippet**: Shows separate validate and lint --strict commands
**Issue**: Development guide shows old workflow
**Recommended Correction**: Update to show `small check --strict` as primary command, note that validate/lint can be run separately for debugging
**Needs Verification**: No

---

## Summary Statistics

**Total Issues**: 15
**Critical**: 2
**Major**: 6
**Minor**: 7

**By Category**:
- Language: 2
- CLI: 5
- Agent: 4
- First-run: 2
- Examples: 2

**Verification Required**: 2 issues (Issue 3, Issue 4 - need local `small start --help`)
