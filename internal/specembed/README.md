# Embedded Schemas

This directory contains the embedded copy of canonical schemas from:

- `spec/small/v1.0.0/schemas/*.schema.json`

These files are copied by tooling and embedded at build time. They are not the authoritative source and should not be edited by hand.

To refresh embedded schemas after spec changes:

```bash
make sync-schemas
```
