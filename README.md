# SMALL Protocol

**SMALL is a protocol for durable, agent-legible project state.**

SMALL defines five canonical artifacts that enable agents to understand, execute, and resume work across sessions. It is not a CMS, not an agent framework, and not MCP.

---

## What is SMALL?

SMALL (State, Manifest, Artifact, Lineage, Lifecycle) provides a minimal, strict protocol for agent continuity. Agents can read project state, execute plans, track progress, and resume work after interruptions.

### Core Principles

- **Explicit State**: All project state is stored in canonical YAML files
- **Ownership Rules**: Clear separation between human-owned and agent-owned artifacts
- **Verifiable Progress**: Every progress entry must include evidence
- **Deterministic Handoff**: Agents resume from a single entrypoint

---

## Quickstart

### Build the CLI

```bash
make small-build
```

### Initialize a Project

```bash
./bin/small init --name myproject
```

This creates `.small/` with all five canonical files:
- `intent.small.yml` - Project goals (human-owned)
- `constraints.small.yml` - Requirements (human-owned)
- `plan.small.yml` - Execution plan (agent-owned)
- `progress.small.yml` - Progress tracking (agent-owned)
- `handoff.small.yml` - Resume entrypoint (shared)

### Validate

```bash
./bin/small validate
```

### Lint

```bash
./bin/small lint
```

### Generate Handoff

```bash
./bin/small handoff
```

---

## Repository Structure

This repository contains:

- **`spec/small/v0.1/`** - SMALL protocol specification
  - `SPEC.md` - Normative specification document
  - `schemas/` - JSON Schema definitions for all artifacts
  - `examples/.small/` - Example YAML files

- **`cmd/small/`** - Reference CLI implementation
  - Go-based tool for validating and managing SMALL artifacts
  - Commands: `init`, `validate`, `lint`, `handoff`, `version`

- **`apps/web/`** - Demo/reference applications
  - Includes `smallcms`, which demonstrates SMALL-CMS v1.0.0
  - SMALL-CMS v1.0.0 is a reference demo showing how SMALL principles apply to content systems

- **`spec/jsonschema/small-cms/v1.0.0/`** - SMALL-CMS specification
  - Content system application of SMALL principles
  - Defines Schema, Manifest, Artifact, Lineage, Lifecycle primitives for content

---

## SMALL vs SMALL-CMS

**SMALL v0.1** is the core protocol for agent continuity:
- Focus: Project state, plans, progress, handoffs
- Artifacts: intent, constraints, plan, progress, handoff
- Use case: Any project where agents need to resume work

**SMALL-CMS v1.0.0** is a demo application of SMALL principles:
- Focus: Content systems (articles, schemas, manifests)
- Artifacts: schema, manifest, artifact, lineage, lifecycle
- Use case: Demonstrates how SMALL principles apply to content management

---

## Protocol Specification

See [`spec/small/v0.1/SPEC.md`](spec/small/v0.1/SPEC.md) for the complete specification.

### Canonical Artifacts

1. **`.small/intent.small.yml`** - Human-owned project intent/goals
2. **`.small/constraints.small.yml`** - Human-owned constraints/requirements
3. **`.small/plan.small.yml`** - Agent-owned execution plan
4. **`.small/progress.small.yml`** - Agent-owned progress tracking
5. **`.small/handoff.small.yml`** - Shared resume entrypoint

### Invariants

- Secrets never stored in any artifact
- Progress entries must include verifiable evidence
- Plan is disposable; progress is not (append-only)
- Handoff is the only resume entrypoint
- `small_version` must be `"0.1"` in all files

---

## Development

### Build CLI

```bash
make small-build
```

### Validate Examples

```bash
cd spec/small/v0.1/examples
../../bin/small validate
```

### Run Tests

```bash
make small-test
```

---

## Status

SMALL v0.1 is the current stable protocol version. The specification is complete and the reference CLI implementation is available.

Breaking changes are expected until v1.0.

---

## Philosophy

- Schemas are law
- Explicit state beats implicit state
- Agents are tools, not magic
- Determinism beats "AI vibes"
- Infrastructure first, products follow
