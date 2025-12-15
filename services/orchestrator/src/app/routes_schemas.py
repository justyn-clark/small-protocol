from __future__ import annotations
from fastapi import APIRouter, HTTPException
from .schemas import SchemaRegistryItem
from .db import tx

router = APIRouter(prefix="/schemas", tags=["schemas"])

@router.post("")
def register_schema(item: SchemaRegistryItem):
    with tx() as cur:
        cur.execute(
            """INSERT INTO schemas (schema_ref, version, schema)
               VALUES (%s,%s,%s::jsonb)
               ON CONFLICT (schema_ref) DO UPDATE SET
                 version=EXCLUDED.version,
                 schema=EXCLUDED.schema,
                 updated_at=NOW()""",
            (item.schema_ref, item.version, _json(item.schema)),
        )
    return {"ok": True, "schema_ref": item.schema_ref, "version": item.version}

@router.get("")
def list_schemas():
    with tx() as cur:
        cur.execute("SELECT schema_ref, version, created_at, updated_at FROM schemas ORDER BY schema_ref ASC")
        return {"schemas": list(cur.fetchall())}

@router.get("/{schema_ref}")
def get_schema(schema_ref: str):
    with tx() as cur:
        cur.execute("SELECT schema_ref, version, schema FROM schemas WHERE schema_ref=%s", (schema_ref,))
        row = cur.fetchone()
        if not row:
            raise HTTPException(status_code=404, detail="schema not found")
        return {"schema_ref": row["schema_ref"], "version": row["version"], "schema": row["schema"]}

def _json(obj):
    import json
    return json.dumps(obj)
