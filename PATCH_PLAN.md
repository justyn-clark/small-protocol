# PATCH_PLAN.md

## Sequencing Strategy

Patches are grouped by type to minimize cross-file conflicts and enable parallel review.

**Group 1**: Canonical Language Pass (state vs execution framing)
**Group 2**: CLI Reference Pass (tables, command docs)
**Group 3**: Agent Entrypoints Pass (operating contract, templates)
**Group 4**: First-Run Guide Pass (getting-started, quickstart)

---

## Group 1: Canonical Language Pass

**Objective**: Eliminate "execution protocol" framing, reinforce state/governance framing

### Files to Edit
1. `docs/FAQ.md`

### Changes

#### `docs/FAQ.md`
- **Line 70**: Replace `SMALL is an **execution protocol** for agent continuity` with `SMALL is a **state protocol** for agent continuity`
- **Line 66-84**: Review entire "Is SMALL a CMS, task runner, or spec format?" section for consistency after language change
- Ensure all language aligns with: SMALL is a governance and continuity layer, not an execution/orchestration engine

**Verification needed**: No

---

## Group 2: CLI Reference Pass

**Objective**: Document missing commands (start, fix), unify validation commands (check), ensure CLI reference completeness

### Files to Edit
1. `docs/cli.md`
2. `docs/cli-guide.md`

### Changes

#### `docs/cli.md`
- **Table at lines 4-18**: Add missing commands:
  - After `small check` row: `| small start | Initialize or repair run handoff state |`
  - After `small doctor` row: `| small fix | Normalize formatting issues in-place |`
- **Verification needed**: YES - run `small start --help` locally to verify exact short description

#### `docs/cli-guide.md`
- **After line 110** (after `small doctor` section): Add new `### small start` section
- Section structure:
  ```markdown
  ### small start

  Initialize or repair run handoff state. This is a self-healing command that ensures handoff.small.yml has a valid replayId.

  ```bash
  small start
  ```

  **Flags:**

  | Flag | Description |
  |------|-------------|
  | `--summary <string>` | Custom summary text for handoff |
  | `--strict` | Enable strict validation |
  | `--dir <path>` | Directory containing .small/ |
  | `--workspace <scope>` | Workspace scope (root, any) |

  **What gets initialized/repaired:**

  - Creates .small/ directory if missing
  - Ensures all canonical artifacts exist
  - Generates or preserves replayId in handoff
  - Self-heals missing replayId automatically

  **When to use start:**

  - First-time workspace setup (alternative to init)
  - Repairing corrupted handoff state
  - Ensuring replayId exists before operations

  **Common errors:**

  | Error | Cause | Resolution |
  |-------|-------|------------|
  | `--workspace examples is not supported` | Cannot start in examples workspace | Use --workspace any or run from root |
  ```

- **Verification needed**: YES - run `small start --help` locally to verify all flags, descriptions, and behavior details

---

## Group 3: Agent Entrypoints Pass

**Objective**: Update agent operating contract to use unified `small check`, ensure checkpoint workflow is clear

### Files to Edit
1. `docs/agent-operating-contract.md`
2. `templates/claude/.claude/CLAUDE.md` (optional, if used as general template)

### Changes

#### `docs/agent-operating-contract.md`

- **Lines 9-19** (Always Read .small/ First section):
  - After line 18 (last artifact in numbered list), add:
    `After reading, validate the workspace state.`

- **Lines 21-30** (Verify the Workspace section):
  - Replace:
    ```bash
    small validate
    small lint
    ```
  - With:
    ```bash
    small check
    ```
  - Add note: "(Alternatively, run `small validate` and `small lint` separately for debugging)"

- **Line 149** (Handoff gate section):
  - Current text is correct, no change needed
  - Confirm `small check --strict` is clearly stated as required

- **Lines 122-124** (Completion Rule heading):
  - Clarify plan --done vs checkpoint:
  - Add after line 123:
    `The \`small plan --done\` command exists for direct plan manipulation, but agents MUST use \`small checkpoint\` to ensure atomic plan+progress state transitions.`

**Verification needed**: No

#### `templates/claude/.claude/CLAUDE.md` (OPTIONAL)

- **Header** (before line 3): Add clarification comment:
  ```
  # Note: This file is REPO-SPECIFIC for working on the small-protocol codebase itself.
  # For a general SMALL agent template, see docs/agent-operating-contract.md.
  ```
- **Lines 10-14** (How to Work section):
  - After "Read the current SMALL state" add:
    `Use \`small check\` to validate workspace before and after changes.`
  - Add bullet for checkpoint:
    `Use \`small checkpoint\` to complete tasks, not manual plan status toggles.`

**Verification needed**: No

---

## Group 4: First-Run Guide Pass

**Objective**: Ensure first-run guides mention unified `small check` and self-healing commands

### Files to Edit
1. `docs/getting-started.md`
2. `docs/quickstart.md`
3. `docs/DEVELOPMENT.md`

### Changes

#### `docs/getting-started.md`

- **Lines 173-186** (Validating the Workspace section):
  - Add before the existing bash block:
    ```markdown
    Use the unified validation command:

    ```bash
    small check
    small check --strict
    ```
    ```
  - After the existing bash block, add note:
    ```markdown
    You can also run validation steps separately for debugging:

    ```bash
    small validate  # Schema validation only
    small lint      # Invariant checks only
    ```
    ```

- **After line 222** (after Next Steps section):
  - Add new section:
    ```markdown
    ## Troubleshooting Your First Workspace

    If you encounter issues:

    ```bash
    small doctor  # Diagnose workspace issues (read-only)
    small fix --versions  # Fix small_version formatting
    small start  # Repair handoff state
    ```
    ```

**Verification needed**: No

#### `docs/quickstart.md`

- **Line 26** (after `small validate`):
  - Add:
    ```bash
    small check
    ```

**Verification needed**: No

#### `docs/DEVELOPMENT.md`

- **Lines 109-118** (Validation During Development section):
  - Replace the three separate commands with:
    ```bash
    # Unified validation and enforcement
    small check --strict

    # Or run validation steps separately:
    small validate  # Schema validation
    small lint --strict  # Invariant checks
    small verify --ci  # CI gate
    ```

**Verification needed**: No

---

## Items Requiring Local Verification

The following changes require running the CLI locally to verify exact help text and flags:

1. **`small start --help`** - Document exact short description for cli.md table (Issue 3)
2. **`small start --help`** - Document all flags and behavior for cli-guide.md section (Issue 4)

**Verification Command**:
```bash
small start --help
```

Expected info to capture:
- Exact short description (for cli.md table)
- All available flags with descriptions
- Default behavior
- Common error messages
- When to use vs `small init` or `small doctor`

---

## Verification Checklist

After applying patches:

- [ ] All 15 drift issues addressed
- [ ] Local `small start --help` verification completed
- [ ] `small start` documented in cli.md table
- [ ] `small start` documented in cli-guide.md with full section
- [ ] "Execution protocol" language removed from FAQ.md
- [ ] `small check` mentioned as unified command in agent-operating-contract.md
- [ ] `small check` mentioned in getting-started.md
- [ ] `small check` mentioned in quickstart.md
- [ ] `small check` mentioned in DEVELOPMENT.md
- [ ] All changes are deterministic and copy-pastable
