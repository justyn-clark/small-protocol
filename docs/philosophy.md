# Philosophy

## Execution as a First-Class Artifact

SMALL treats execution as a first-class artifact.

When systems fail, SMALL leaves evidence instead of speculation.

It functions as a flight recorder for agentic workflows.

## Design Rationale

Traditional AI-assisted workflows rely on ephemeral chat history that is:

- Lost when sessions end
- Difficult to audit
- Hard to resume reliably

SMALL replaces this with durable, machine-readable artifacts that:

- Persist beyond any single session
- Support audit and compliance requirements
- Enable reliable resumption from any checkpoint

## Explicit Non-Goals

| Not This              | Why                                          |
|-----------------------|----------------------------------------------|
| Task runner           | Execution is recorded, not orchestrated      |
| Multi-agent framework | Single-writer preserves determinism          |
| LLM product           | Protocol only                                |
| Permission system     | Describes constraints, does not enforce them |

## What SMALL Is

SMALL is a governance and continuity layer.

It defines what artifacts must exist, who owns them, and what invariants they must satisfy.

It does not define how work is performed, only how it is recorded.

## What SMALL Is Not

SMALL is not:

- An agent framework
- A prompt format
- A workflow engine
- A multi-agent system

These may be built on top of SMALL, but SMALL itself remains minimal and protocol-focused.
