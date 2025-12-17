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

## The SMALL Model

This project is built around SMALL:

Schema → Manifest → Artifact → Lineage → Lifecycle

SMALL defines the minimal execution primitives required to build agent-legible systems.

---

## SMALL as a Protocol

SMALL is not just a concept—it's a versioned, machine-consumable protocol. Agents can discover and validate against it programmatically.

### Discovery

The protocol is discoverable via a REST endpoint:

```http
GET /protocol/small/v1
```

Returns a JSON contract specifying:

- Protocol name and version
- Available primitives (Schema, Manifest, Artifact, Lineage, Lifecycle)
- Protocol rules and guarantees
- Schema locations for validation

### Versioning

SMALL follows semantic versioning (Major.Minor.Patch):

- **Major**: Breaking changes that require agent updates
- **Minor**: Backward-compatible additions
- **Patch**: Bug fixes and clarifications

The current version is **1.0.0**.

### Guarantees

SMALL enforces these guarantees at the protocol level:

- **Deterministic**: Same inputs produce same outputs
- **Immutable Artifacts**: Once published, artifacts cannot be modified
- **Explicit Contracts Only**: No hidden state or implicit behavior
- **No Hidden State**: All system state is observable and addressable

### Determinism Promise

All SMALL operations are deterministic. Given the same manifest and schema, validation, lineage generation, and lifecycle emission produce identical results. This enables reproducible agent workflows and reliable automation.

### Replay IDs

SMALL generates deterministic replay IDs for all operations, enabling exact replay of workflows. The replay ID is computed as:

```bash
sha256("SMALL|" + protocolVersion + "|" + canonicalJson(manifest))
```

This ensures:

- Same manifest + same protocol version → same replay ID
- Different protocol version → different replay ID  
- Different manifest → different replay ID

Replay IDs are included in all lineage and lifecycle records, providing a cryptographic guarantee of determinism.

### Agent Integration

Agents integrate with SMALL by:

1. **Discovering** the protocol version via `/protocol/small/v1`
2. **Fetching** referenced schemas from `/schemas/*.schema.json`
3. **Validating** artifacts against schemas before materialization
4. **Generating** lineage and lifecycle records for all operations
5. **Operating** within the protocol's deterministic guarantees

The protocol is the source of truth. Documentation is validated against it at build time, preventing drift between docs and implementation.

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
- ✅ SMALL Protocol v1.0.0 (Run #5) — Protocol discovery endpoint, build-time doc validation, SMALL Playground
- ✅ Executable SMALL (Run #6) — Deterministic replay IDs, strict offline validation, OpenAPI surface

---

## Where This Goes

Short term:

- ✅ OpenAPI surface (`/openapi/small.v1.yaml`)
- ✅ JSON Schema registry (local meta-schema, offline validation)
- CLI reference runner

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

## Run #6: Executable SMALL

Run #6 transforms SMALL from "docs + demo" into an executable protocol with:

### Deterministic Replay

- **Replay IDs**: Cryptographic identifiers computed from protocol version and manifest
- **Protocol Version Validation**: Enforces supported protocol versions (currently `["1.0.0"]`)
- **Canonical JSON**: Stable serialization ensures identical inputs produce identical outputs

### Strict Offline Validation

- **Bundled Meta-Schema**: Draft 2020-12 meta-schema included locally (no remote fetching)
- **Schema Registry**: AJV registry loads all schemas from local filesystem
- **Build-Time Verification**: Schema registry verified during docs build to catch drift early

### OpenAPI Surface

- **API Contract**: [`/openapi/small.v1.yaml`](/openapi/small.v1.yaml) documents all endpoints
- **Endpoints**: Protocol discovery, schema access, manifest validation, replay workflow
- **Documentation**: [`/docs/api`](/docs/api) provides integration guide and examples

### Schema Strategy

- **Canonical Source**: `spec/jsonschema/` contains authoritative schemas
- **Runtime Sync**: `scripts/sync-schemas.ts` syncs schemas to `apps/web/public/schemas/`
- **Portable IDs**: Schemas use internet-portable `$id` URLs (rewritten at sync-time for local dev)

### Acceptance Checklist

- ✅ Schema validation works offline (no AJV meta-schema errors)
- ✅ Deterministic replay (same inputs → same replay ID)
- ✅ Static assets load (`/schemas/meta/draft2020-12.schema.json`, `/schemas/small/v1/*.schema.json`, `/openapi/small.v1.yaml`)
- ✅ No hydration warnings in dev console

---

## Status

This is an active, opinionated system under construction.

Breaking changes are expected until v1.0.

Follow along. Contribute deliberately.
