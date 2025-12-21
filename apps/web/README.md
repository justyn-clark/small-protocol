# smallcms - SMALL-CMS Reference Demo

> **This is a reference demo (smallcms) built on SMALL-CMS v1.0.0, which demonstrates SMALL protocol principles applied to content systems.**

This application serves as a reference implementation showing how the SMALL protocol can be applied to content management systems. It demonstrates:

- Schema-driven content modeling
- Manifest-based deployment
- Immutable artifacts
- Lineage tracking
- Lifecycle management

## SMALL vs SMALL-CMS

- **SMALL v0.1**: Core protocol for agent continuity (intent, constraints, plan, progress, handoff)
- **SMALL-CMS v1.0.0**: Content system application of SMALL principles (schema, manifest, artifact, lineage, lifecycle)

This demo implements SMALL-CMS v1.0.0. For the core SMALL protocol specification, see `spec/small/v0.1/` in the repository root.

## Development

```bash
npm install
npm run dev
```

## Learn More

- Core SMALL protocol: `spec/small/v0.1/SPEC.md`
- SMALL-CMS specification: `spec/jsonschema/small-cms/v1.0.0/`
