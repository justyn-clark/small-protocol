# Security Policy

## Scope

This repository implements the SMALL protocol and tooling. It must remain safe to clone, build, and run in untrusted environments (CI, agents, external contributors).

## Secrets and sensitive files

- Do not commit secrets, credentials, tokens, API keys, certificates, private keys, or signed URLs.
- Do not read, inspect, infer, or reference `.env` files or other secret/config files.
- Do not create, modify, overwrite, or suggest edits to `.env` files or secret files.
- If configuration examples are needed, use placeholder values in a documented example file (e.g. `.env.example`) and keep it non-sensitive.

## Reporting

If you discover a security issue:

- Do not open a public issue with exploit details.
- Contact the maintainer privately with:
  - A description of the issue and impact
  - Reproduction steps
  - Any suggested mitigation

## Disclosure

Security fixes should be:

- Minimal and targeted
- Accompanied by tests when possible
- Noted in `CHANGELOG.md` under `Security`

## Hard rule

Any change that adds or exposes secrets will be rejected.
