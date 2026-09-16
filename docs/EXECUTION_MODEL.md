# SMALL Execution Model

This document defines the historical execution model for SMALL v1.0.0 and the
separate SMALL v2.0.0 session profile.

## Single-Writer Design

SMALL v1 is single-writer by design. Only one agent may write to v1 artifacts at a time.

Mutating CLI operations are serialized by a checkout-local exclusive lock and
fail busy when another cooperative writer holds it. There is no automatic merge
or distributed lock.

### Why Single-Writer?

AI agents are non-deterministic. Given the same input, two agents may produce different outputs. If two agents write to the same artifact simultaneously:

- Silent merges corrupt intent
- Automatic resolution cannot determine which output is correct
- The resulting state may satisfy neither agent's goal

Failure is safer than ambiguity. SMALL chooses to fail loudly rather than produce corrupt state.

### What Happens on Concurrent Writes

If two processes attempt to modify SMALL artifacts simultaneously:

1. The first cooperative CLI writer holds the checkout-local lock
2. The second cooperative writer is refused as busy
3. Multi-file publications use a recoverable transaction journal
4. A later command recovers a prepared publication before accepting new writes

The lock cannot coordinate disconnected clones, non-cooperating direct file
writes, or network filesystems with incompatible locking semantics. V1 histories
that diverge across clones may still produce ordinary Git conflicts in shared
YAML files; resolve those explicitly and re-run strict validation.

## Current v1 Guarantees

### Append-Only Progress

`progress.small.yml` is append-only. Progress entries are never deleted or modified after creation. This provides:

- Complete audit trail of all work
- Ability to reconstruct state at any point
- Evidence preservation for verification

### Explicit Resume Points

For v1, `handoff.small.yml` is the explicit resume entrypoint. Agents do not
attempt to reconstruct v1 state from raw artifacts. The handoff provides:

- Stable snapshot of current state
- Recent progress for context
- Clear next actions

### Git-Based Time Travel

SMALL artifacts are plain files. Git provides:

- Full history of all changes
- Ability to revert to any previous state
- Diff-based inspection of changes
- Branch-based parallel work (with explicit merge)

If an agent corrupts state, `git checkout` restores the previous known-good state.

## Out of Scope for v1.0.0

The following features are explicitly not part of SMALL v1.0.0:

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

## Historical design directions

The following v1-era ideas are retained as design history, not current v2
command documentation. V2 implemented attributed sessions and semantic
reconciliation without implementing task leases, branch automation, or
distributed consensus.

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

SMALL v1.0.0 is single-writer. This is intentional. The complexity cost of automatic concurrency exceeds the benefit for the target use case: durable, verifiable agent continuity.

## SMALL v2.0.0 session profile

v1 remains exactly single-writer. v2 introduces immutable session descriptors,
per-session event records, explicit causal parents, typed receipts, and a
deterministic reducer. New-format workspaces default to solo: one active local
writer, with explicit close/resume/takeover. Collaborative mode is opt-in and
permits multiple attributed sessions; it does not provide a distributed lock.

The reducer preserves independent events and distinguishes compatible history
from semantic decisions. It never chooses incompatible task outcomes, policy or
mode changes by timestamp. Unresolved conflicts block strict validation,
snapshots, and authoritative handoff until a full-head, expected-frontier
resolution event is recorded.

`small apply` executes one command and records its outcome. It does not retry,
schedule, spawn, select models, or accept a task. `small checkpoint` records the
separate acceptance decision and evidence. External orchestration remains
outside SMALL.

Tracked v2 state is immutable history plus human policy and import archives.
Locks, active-session pointers, transaction journals, and indexes are local
`.small-cache/` data and may be discarded/rebuilt. See
[session-profile-v2.md](session-profile-v2.md) for the operational contract.

If you need concurrency, use git branches and merge explicitly.
