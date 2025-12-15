# Agent-Legible CMS Spec

A canonical specification and reference implementation for **agent-legible content systems**.

This project defines how content should be modeled, validated, versioned, and operated in an AI-driven environment — using **schemas, manifests, and deterministic workflows**, not opaque CMS abstractions.

---

## Why This Exists

Traditional CMS platforms optimize for:

- human editors
- WYSIWYG interfaces
- mutable state
- implicit workflows

AI systems require the opposite:

- explicit contracts
- machine-verifiable state
- deterministic lineage
- observable lifecycle events

This repo is a response to that mismatch.

---

## Core Idea

> Content must be **agent-legible** before it can be agent-operated.

That means:

- schemas before UI
- manifests before publishing
- validation before mutation
- lineage before automation

---

## Project Structure

### Phase 0 — Canonical Spec

- Define primitives
- Establish terminology
- Publish durable contracts

### Phase 1 — Reference Implementation

- Render specs using MDX
- Show canonical relationships
- Provide executable examples

### Phase 2 — Reference Workflow (Run #3)

- Validate manifests
- Generate lineage
- Emit lifecycle events
- Demonstrate determinism end-to-end

---

## Core Primitives

| Primitive  | Purpose |
|----------|---------|
| Artifact | Immutable, versioned content unit |
| Schema | Structural + semantic contract |
| Manifest | Deployment-time intent |
| Lifecycle | State transitions over time |
| Lineage | Provenance + derivation |
| Agent Action | Explicit, validated mutation |

Each primitive is:

- schema-defined
- versioned
- serializable
- deterministic

---

## What This Is Not

- Not a CMS replacement (yet)
- Not a UI-first product
- Not an AI wrapper on legacy tools
- Not prompt engineering

This is **infrastructure**.

---

## Current State

- ✅ Protocol-styled documentation site
- ✅ MDX rendering with syntax highlighting
- ✅ Mermaid diagrams
- ✅ Primitive Spec v1 published
- ✅ Reference Workflow (interactive validation)

---

## Where This Goes

Short term:

- CLI reference runner
- OpenAPI surface
- JSON Schema registry

Medium term:

- Agent execution engine
- Manifest-driven orchestration
- Durable artifact storage

Long term:

- Replace CMS abstractions with agent-legible primitives
- Become the canonical standard others integrate against

---

## Philosophy

- Schemas are law
- Manifests are intent
- Agents are tools, not magic
- Determinism beats "AI vibes"
- Infrastructure first, products follow

---

## Status

This is an active, opinionated system under construction.

Breaking changes are expected until v1.0.

Follow along. Contribute deliberately.
