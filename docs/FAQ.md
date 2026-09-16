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

It depends on the profile and whether the writers share a checkout.

V1 remains single-writer. Mutating CLI operations in one checkout use a local
exclusive lock and fail busy rather than silently interleave state-file writes.
The lock is not distributed: disconnected clones can still diverge, and their
shared YAML tails may conflict when Git merges them.

V2 also uses a checkout-local lock, but records each writer in unique immutable
session and event paths. A v2 workspace defaults to solo mode, so another active
writer is refused. Collaborative mode must be selected explicitly. After Git
combines independent histories, the deterministic reducer detects incompatible
task outcomes, policy changes, mode changes, stale evidence, and incomplete
resolutions. Unresolved conflicts make strict validation and authoritative
handoff fail until an explicit full-head resolution is recorded.

Neither profile provides a distributed lease, source-code conflict resolution,
automatic Git merge, or execution orchestration.

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

For v1, use one of these externally coordinated patterns:

**Sequential handoff**: Agents work one at a time. Agent A completes and generates handoff. Agent B resumes from handoff. No concurrent writes.

**Branch-per-agent**: Each agent works on a git branch. Branches are merged explicitly by a human or orchestration layer. Merge conflicts are resolved before committing.

**Task partitioning**: Tasks are assigned to specific agents. Each agent writes only to its assigned tasks. Requires careful orchestration.

For v2, keep solo mode for ordinary work. Opt into collaborative mode only when
you need multiple attributed sessions, give each writer a distinct session, let
Git transport the immutable records, then run `small reconcile` and
`small check --strict` after histories meet. SMALL records and validates the
coordination state; an external tool or human still schedules and executes work.

## Identity

### Is SMALL a CMS, task runner, or spec format?

None of these.

SMALL is an **execution protocol** for agent continuity.

- **Not a CMS**: SMALL does not store content. It stores project state metadata.
- **Not a task runner**: SMALL does not decide or schedule tasks. `small apply` runs one supplied child command and records its outcome; it does not orchestrate a workflow.
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
- In v1, handoff is the explicit resume entrypoint; v2 uses bounded reducer-backed session resume

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
