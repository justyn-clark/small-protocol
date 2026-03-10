# SECRETS_POLICY.md

This repo follows the JCN universal secrets policy.

## Rule

Real secret env files must never live inside the repo.

Forbidden in repos:
- `.env`
- `.env.local`
- `.env.production`
- `.env.development`
- `.env.*` with real values
- any tracked file that contains live credentials intended for runtime

Allowed in repos:
- `.env.example`
- `.env.sample`
- `.env.template`
- documentation that describes required variables without real values

## Runtime source of truth

Real runtime env files must live outside the repo under:
- `~/.config/jcn/env/<repo>.env`

Apps and tools should load runtime env from that external path via wrappers, `just` recipes, Docker `--env-file`, launchd, or secret injection.

## Agent rule

Agents should never read or write real `.env` files inside the repo.
If runtime configuration is needed, use the external JCN env path convention instead.

## Commit guardrail

A shared git hook blocks committing forbidden `.env` files.
That hook is a safety net, not permission to keep secrets in the repo.
