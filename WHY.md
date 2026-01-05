# Why SMALL Exists

Modern AI agents are powerful, but fragile.

They forget.
They restart.
They hallucinate state.
They lose context between sessions, tools, and environments.

SMALL exists to solve that problem — **without frameworks, platforms, or lock-in**.

## The Core Problem

Agents today operate inside transient contexts:

- chat windows
- editor sessions
- ephemeral tools
- partial memory stores

When an agent stops, crashes, or is replaced:

- intent is lost
- plans drift
- progress becomes unverifiable
- humans are forced to restate everything

Most solutions try to fix this by adding:

- agent runtimes
- orchestration layers
- memory databases
- proprietary formats

These approaches increase complexity and brittleness.

## The SMALL Approach

SMALL is deliberately minimal.

Instead of building a system, SMALL defines a **contract**:

- a small set of human-readable artifacts
- with strict semantics
- validated by schemas
- resumable by any agent or human

SMALL answers one question clearly:

> “If this agent stopped right now, how would the next one continue?”

## What SMALL Is

SMALL is:

- a **protocol**, not a platform
- **file-based**, not service-based
- **language-agnostic**
- **tool-agnostic**
- **human-legible and agent-legible**

At its core are five artifacts:

- `intent` — why this work exists
- `constraints` — what must not be violated
- `plan` — what is intended next (disposable)
- `progress` — what has actually happened (append-only)
- `handoff` — the sole resume entrypoint

Together, they form a durable continuity surface.

## What SMALL Is Not

SMALL is **not**:

- a CMS
- an agent framework
- a memory database
- an orchestration engine
- a replacement for your tools

SMALL does not execute anything.
It does not store embeddings.
It does not run agents.

It simply makes state **explicit, verifiable, and transferable**.

---

## Why Files?

Files are:

- inspectable
- diffable
- portable
- version-controlled
- understandable without infrastructure

If a human can read it, an agent can reason about it.
If an agent can write it, a human can audit it.

This symmetry is intentional.

## Why v1.0.0 Is Stable

SMALL v1.0.0 defines the **minimum viable contract** for continuity.

Nothing more.
Nothing less.

v1.0.0 ensures:

- long-term stability
- low cognitive overhead
- confidence for implementers
- freedom to build extensions without breaking the core

Future versions may add capabilities, but v1.0.0 provides a stable foundation.

## The Goal

SMALL is not trying to be clever.

It is trying to be **reliable**.

When an agent fails, restarts, or is replaced, SMALL ensures that:

- work can continue
- intent is preserved
- progress is accountable
- context is never lost to hand-waving

That’s it.

That’s the whole point.
