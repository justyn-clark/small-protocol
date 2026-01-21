# CANONICAL_COPY_BLOCKS.md

## Block 1: What SMALL Is

SMALL (Schema, Manifest, Artifact, Lineage, Lifecycle) is a state protocol for agent continuity. It defines five canonical artifacts that make AI-assisted work legible, auditable, and resumable by separating durable state from ephemeral execution.

SMALL is a governance and continuity layer. It defines what artifacts must exist, who owns them, and what invariants they must satisfy. It does not define how work is performed, only how it is recorded.

## Block 2: What SMALL Is Not

SMALL is not:
- An agent framework
- A prompt format
- A workflow engine
- A multi-agent system
- A task runner
- An execution orchestrator

SMALL does not execute anything. It does not store embeddings. It does not run agents. It does not orchestrate workflows. It simply makes state explicit, verifiable, and transferable.

## Block 3: Golden Path (Human Workflow)

**Human workflow**: Edit `intent.small.yml` and `constraints.small.yml` to define what work should be done and what rules must be followed. The agent handles the rest.

Humans define **what** and **why**. Agents define **how** and record **that they did it**.

Run `small init` to create the workspace, edit the two human-owned files, then let the agent work within those boundaries.

## Block 4: Golden Path (Agent Workflow)

**Agent workflow**:
1. Read `.small/` artifacts first (intent, constraints, plan, progress, handoff)
2. Validate workspace with `small check`
3. Work incrementally, using `small checkpoint` for task completion
4. Record all execution with evidence in `progress.small.yml`
5. Run `small check --strict` before handoff
6. Generate handoff with `small handoff` when stopping

Agents must respect ownership rules, never edit human-owned files, and always use `checkpoint` for completion (not `plan --done`).

## Block 5: Agent Completion Rule

Agents MUST use `small checkpoint` for task completion, not `small plan --done`.

**Why**: `checkpoint` performs an atomic state transition, updating both plan status and appending progress evidence in one operation. This prevents drift between plan state and progress records.

**Before handoff**: Agents MUST run `small check --strict` to verify all completed tasks have corresponding progress entries with evidence.

The `small plan --done` command exists for direct plan manipulation but should not be used by agents during execution.

## Block 6: State vs Execution Boundary

SMALL governs **durable state**, not **ephemeral execution**.

The CLI records what happened and validates invariants. It does not orchestrate agents, execute workflows, or manage concurrency. The orchestration layer (your tooling, your CI, your agent runtime) decides what to execute. SMALL records what happened.

This separation ensures SMALL remains minimal, stable, and framework-agnostic. You can use SMALL with any agent, any LLM, any execution environment.
