# SMALL Protocol FAQ

Answers to common questions about SMALL. These reflect actual system behavior, not aspirational features.

## Deployment

### Can SMALL run on-prem / air-gapped?

Yes. SMALL is entirely local.

SMALL artifacts are plain YAML files stored in `.small/` within your project. The CLI is a single Go binary with no external dependencies. There are no network calls, no cloud services, no telemetry.

Requirements:

- A filesystem
- The `small` binary
- Git (optional, but recommended for history)

SMALL works in air-gapped environments, on-prem servers, local machines, and CI runners. If you can run a Go binary, you can run SMALL.

## Concurrency

### What happens if two agents write at the same time?

Last write wins. Data may be lost.

SMALL is single-writer by design. There is no locking, no conflict detection, no automatic merge. If two agents write to the same artifact simultaneously:

1. The second write overwrites the first
2. Progress entries from the first agent may be lost
3. Plan and progress may become inconsistent
4. No error is raised

Prevention is the responsibility of the orchestration layer. SMALL assumes you have ensured only one agent writes at a time.

See [EXECUTION_MODEL.md](./EXECUTION_MODEL.md) for details.

### Why doesn't SMALL auto-merge agent output?

Because auto-merge cannot preserve intent.

AI agents are non-deterministic. Two agents given the same task may produce different outputs. If SMALL auto-merged:

- Which output is correct?
- What if they contradict?
- What if the merge is syntactically valid but semantically wrong?

SMALL cannot answer these questions. No algorithm can.

Auto-merge creates ambiguous state. Ambiguous state leads to silent failures. Silent failures lead to corrupted projects. SMALL chooses explicit failure over implicit corruption.

### How do I scale to multiple agents safely?

Use one of these patterns:

**Sequential handoff**: Agents work one at a time. Agent A completes and generates handoff. Agent B resumes from handoff. No concurrent writes.

**Branch-per-agent**: Each agent works on a git branch. Branches are merged explicitly by a human or orchestration layer. Merge conflicts are resolved before committing.

**Task partitioning**: Tasks are assigned to specific agents. Each agent writes only to its assigned tasks. Requires careful orchestration.

All patterns require external coordination. SMALL does not provide this coordination.

## Identity

### Is SMALL a CMS, task runner, or spec format?

None of these.

SMALL is an **execution protocol** for agent continuity.

- **Not a CMS**: SMALL does not store content. It stores project state metadata.
- **Not a task runner**: SMALL does not execute tasks. `small apply` records execution; it does not orchestrate it.
- **Not a spec format**: SMALL defines enforceable artifacts, not descriptive documentation.

SMALL provides:

- Explicit state representation
- Verifiable progress tracking
- Deterministic resume points
- Append-only audit trail

The orchestration layer decides what to execute. SMALL records what happened.

### How is SMALL different from spec-only tools?

Spec-only tools help humans write specifications. SMALL enforces execution against specifications.

**Spec-only tools** (e.g., Spec Kit):

- Generate documentation
- Provide templates and structure
- Help humans describe what they want
- Output is prose for other humans to read

**SMALL**:

- Enforces artifact structure via JSON Schema
- Validates invariants (no secrets, verifiable progress)
- Tracks project state
- Provides machine-readable resume points
- Progress is append-only and auditable

Spec-only tools answer: "What do we want to build?"
SMALL answers: "What has been done, and what happens next?"

You can use spec-only tools to write the initial intent and constraints, then use SMALL to track execution.

## Implementation

### Why is the CLI coupled to the spec?

The CLI is the reference enforcer.

SMALL is not a descriptive specification that implementations may interpret loosely. The CLI defines correct behavior. If the CLI rejects an artifact, the artifact is invalid.

Benefits:

- No spec drift between documentation and implementation
- Clear answer to "is this valid?" (run the CLI)
- Single source of truth for invariants
- Other implementations can test against the CLI

Other implementations are permitted but must pass the same invariants. The test is: does the CLI accept the output?

### Can I implement SMALL in another language?

Yes, with constraints.

You may implement SMALL in any language. Your implementation must:

1. Produce artifacts the Go CLI accepts as valid
2. Enforce all invariants documented in SPEC.md
3. Pass the same validation tests

The Go CLI is authoritative. If your implementation produces output the CLI rejects, your implementation is wrong.

### Why YAML?

YAML is human-readable and machine-parseable.

- Agents can write it
- Humans can read and edit it
- Git diffs are meaningful
- Comments are preserved (unlike JSON)

JSON Schema validates the structure. YAML is the serialization format.

## Safety

### What safety defaults does SMALL enforce?

- `small apply` defaults to dry-run if no `--cmd` is provided
- Progress is append-only (never deleted)
- Secrets are rejected during lint
- All progress entries require evidence
- Handoff is the only resume entrypoint (no state reconstruction)

### Can agents delete progress?

No. Progress is append-only.

The `progress.small.yml` file may only grow. Entries cannot be deleted or modified after creation. This ensures:

- Complete audit trail
- No evidence tampering
- Ability to verify claims against history

If you need to invalidate progress, add a new entry indicating the previous entry is superseded.

### What if an agent corrupts state?

Use git.

SMALL artifacts are plain files. Git provides:

```bash
git log --oneline .small/
git diff HEAD~1 .small/
git checkout HEAD~1 -- .small/
```

Restore the last known-good state and re-run the agent.
