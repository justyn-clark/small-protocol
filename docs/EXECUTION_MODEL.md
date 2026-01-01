# SMALL Execution Model

This document defines the execution and concurrency model for SMALL v0.x.

## Single-Writer Design

SMALL is single-writer by design. Only one agent may write to SMALL artifacts at a time.

Concurrent writes are treated as errors, not merged. There is no automatic conflict resolution.

### Why Single-Writer?

AI agents are non-deterministic. Given the same input, two agents may produce different outputs. If two agents write to the same artifact simultaneously:

- Silent merges corrupt intent
- Automatic resolution cannot determine which output is correct
- The resulting state may satisfy neither agent's goal

Failure is safer than ambiguity. SMALL chooses to fail loudly rather than produce corrupt state.

### What Happens on Concurrent Writes

If two processes attempt to modify SMALL artifacts simultaneously:

1. The second write will overwrite the first (last-write-wins at the filesystem level)
2. Progress entries from the first write may be lost
3. Plan state may become inconsistent with progress
4. Handoff will reflect only the last writer's state

SMALL does not detect or prevent this at runtime. Prevention is the responsibility of the orchestration layer.

## Current Guarantees

### Append-Only Progress

`progress.small.yml` is append-only. Progress entries are never deleted or modified after creation. This provides:

- Complete audit trail of all work
- Ability to reconstruct state at any point
- Evidence preservation for verification

### Explicit Resume Points

`handoff.small.yml` is the only resume entrypoint. Agents do not attempt to reconstruct state from raw artifacts. The handoff provides:

- Deterministic snapshot of current state
- Recent progress for context
- Clear next actions

### Git-Based Time Travel

SMALL artifacts are plain files. Git provides:

- Full history of all changes
- Ability to revert to any previous state
- Diff-based inspection of changes
- Branch-based parallel work (with explicit merge)

If an agent corrupts state, `git checkout` restores the previous known-good state.

## Out of Scope for v0.x

The following features are explicitly not part of SMALL v0.x:

### CRDTs

Conflict-free replicated data types would enable automatic merge. SMALL does not use CRDTs because:

- They add complexity
- They obscure intent
- They assume conflicts can be resolved automatically
- They are not necessary for single-writer workflows

### Automatic Merge

SMALL does not automatically merge concurrent changes. If you need concurrent agents:

- Use separate branches per agent
- Merge explicitly via git
- Validate after merge

### Distributed Consensus

SMALL does not implement distributed consensus protocols. There is no leader election, no quorum, no Paxos, no Raft. The orchestration layer is responsible for ensuring single-writer semantics.

## Safe Multi-Agent Patterns

If you need multiple agents working on the same project:

### Sequential Handoff

Agents work sequentially. Agent A completes, generates handoff, Agent B resumes from handoff. No concurrent writes.

### Branch-Per-Agent

Each agent works on a separate git branch. Branches are merged explicitly by a human or orchestration layer. Merge conflicts are resolved before committing.

### Task Partitioning

Tasks are partitioned across agents. Each agent owns distinct tasks and writes to distinct parts of the plan. Requires careful orchestration to avoid overlap.

## Future Directions

The following are conceptual directions for future versions. They are not commitments.

### Agent Identity Tagging

Progress entries could include agent identity:

```yaml
entries:
  - timestamp: "2025-01-15T10:00:00Z"
    agent_id: "agent-alpha"
    task_ref: "task-1"
    ...
```

This would enable:

- Audit of which agent performed which work
- Filtering progress by agent
- Debugging multi-agent issues

### Task Leases

Tasks could be leased to agents with expiration:

```yaml
tasks:
  - id: "task-1"
    lease:
      agent_id: "agent-alpha"
      expires: "2025-01-15T11:00:00Z"
```

This would enable:

- Explicit ownership during execution
- Automatic release on timeout
- Detection of concurrent claims

### Branch-Per-Agent Workflows

Tooling could automate branch creation and merge:

```bash
small branch --agent alpha
# Creates branch small/agent-alpha
# Agent works on branch
small merge --agent alpha
# Merges branch back to main after validation
```

### Explicit Merge Validation

A hypothetical `small validate --merge` could:

- Compare two branches
- Detect conflicting progress entries
- Require explicit resolution
- Validate merged state

These are conceptual only. Implementation depends on demonstrated need.

## Summary

SMALL v0.x is single-writer. This is intentional. The complexity cost of automatic concurrency exceeds the benefit for the target use case: durable, verifiable agent continuity.

If you need concurrency, use git branches and merge explicitly.
