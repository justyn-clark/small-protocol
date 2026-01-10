# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

You are operating under the SMALL Protocol.

SMALL is an execution model with five artifacts:
- intent.small.yml (human-defined goal)
- constraints.small.yml (human-defined limits)
- plan.small.yml (your proposed approach)
- progress.small.yml (your execution log, append-only)
- handoff.small.yml (resume point for continuity)

Your operating rules:

1. READ intent and constraints before proposing any plan.
2. PROPOSE a plan in plan.small.yml. Do not execute until the plan exists.
3. EXECUTE work step by step, recording each action in progress.small.yml.
4. NEVER modify intent or constraints without explicit human instruction.
5. APPEND to progress.small.yml after each completed step. Include evidence (commits, test results, file changes).
6. GENERATE handoff.small.yml at checkpoints or when stopping.
7. VALIDATE artifacts with `small validate` if the CLI is available.
8. STOP if you encounter a constraint violation. Do not work around constraints.
9. ASK for clarification if intent is ambiguous. Do not guess.

If a .small/ directory exists, read handoff.small.yml first to resume prior work.

Your execution must be:
- Explicit (no hidden actions)
- Auditable (all changes evidenced)
- Resumable (handoff enables any agent to continue)

Begin by reading the .small/ directory contents.

## Project Overview

**SMALL Protocol** is a formal execution protocol for making AI-assisted work legible, deterministic, and resumable. It defines five canonical YAML artifacts that replace ephemeral chat history with durable execution state.

The protocol itself is implemented as a Go CLI tool with validation, linting, and execution tracking capabilities.

## Essential Commands

```bash
# Build the CLI tool
make small-build

# Run all tests
make small-test

# Run a single test
go test -run TestName ./internal/commands

# Validate examples directory
make small-validate

# Format code
make small-format

# Check formatting
make small-format-check

# Lint examples
make small-lint

# Sync schemas from spec to embedded location (required before build)
make sync-schemas

# Verify with script
make verify
```

## Code Architecture

### Key Directories

- **`cmd/small/`** - CLI entry point that delegates to command handlers
- **`internal/commands/`** - Command implementations (init, validate, lint, plan, apply, status, handoff)
- **`internal/small/`** - Core protocol logic (validator, loader, invariants checking)
- **`internal/specembed/`** - Embedded JSON schemas for artifact validation
- **`spec/small/v1.0.0/`** - Specification files (SPEC.md), schemas, and examples

### Core Packages

**`internal/small/`** provides the protocol implementation:
- **`validator.go`** - Validates artifacts against JSON schemas
- **`invariants.go`** - Enforces non-negotiable protocol rules (ownership, status transitions, etc.)
- **`loader.go`** - Loads artifacts from disk
- **`version.go`** - Version tracking

**`internal/commands/`** implements CLI behavior:
- **`init.go`** - Initializes .small/ directory with canonical artifacts
- **`validate.go`** - Validates artifacts against schemas
- **`lint.go`** - Checks for protocol violations
- **`plan.go`** - Creates or modifies plan.small.yml
- **`apply.go`** - Records command executions in progress.small.yml
- **`status.go`** - Shows current workspace state
- **`handoff.go`** - Generates handoff.small.yml for session boundaries
- **`root.go`** - Cobra root command setup

### Artifact Validation Flow

1. Artifacts are loaded from `.small/` directory
2. Schemas are embedded in the binary from `internal/specembed/schemas/`
3. JSON schema validation is performed via `validator.go`
4. Protocol invariants are checked via `invariants.go`
5. Human-owned artifacts (intent, constraints) cannot be modified by agents

### Testing

Tests cover:
- Invariant enforcement (apply_test.go, status_test.go, plan_test.go)
- Protocol rules (invariants_test.go)

Run tests frequently to ensure schema changes don't break validation logic.

## Important Design Decisions

1. **Schema Embedding**: Schemas are embedded at build time via `make sync-schemas`, allowing the CLI to validate offline without external dependencies.

2. **Artifact Ownership**: Protocol enforces strict ownership:
   - Human-owned: `intent.small.yml`, `constraints.small.yml`
   - Agent-owned: `plan.small.yml`, `progress.small.yml`
   - System-owned: `handoff.small.yml`

3. **Validation Before Execution**: Commands like `apply` validate artifacts before recording progress.

4. **Invariant Checking**: `invariants.go` checks non-negotiable protocol rules beyond JSON schema validation (e.g., progress entries are append-only, status transitions are valid).

## Common Development Tasks

### Adding a New Command

1. Create a new file in `internal/commands/` following existing patterns
2. Implement a function that returns `*cobra.Command`
3. Register it in `root.go` with `rootCmd.AddCommand()`
4. Add tests in a corresponding `*_test.go` file

### Modifying Schemas

1. Edit schemas in `spec/small/v1.0.0/schemas/`
2. Run `make sync-schemas` to copy them to `internal/specembed/schemas/`
3. Update `validator.go` if validation logic needs to change
4. Run `make small-validate` to test against examples
5. Update examples in `spec/small/v1.0.0/examples/` if needed

### Fixing Invariant Violations

1. Check `internal/small/invariants.go` for the rule being violated
2. Look at `invariants_test.go` for test cases that document expected behavior
3. Fix the violation in the command or add proper error handling
4. Add test coverage for the edge case

## Files to Understand First

- `README.md` - High-level protocol overview
- `spec/small/v1.0.0/SPEC.md` - Authoritative specification
- `docs/agent-operating-contract.md` - Rules agents must follow
- `internal/small/invariants.go` - Non-negotiable protocol rules

## Key Invariants (From Protocol)

- Progress entries are append-only (never modify or delete)
- Status transitions must follow valid state machine (pending → in_progress → completed/blocked)
- Human-owned artifacts cannot be edited by agents
- All commands must be recorded via `small apply` before claiming execution
- Validation must pass before marking tasks complete
