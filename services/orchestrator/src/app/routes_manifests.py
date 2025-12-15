from __future__ import annotations
from fastapi import APIRouter, HTTPException
from .schemas import Manifest
from .db import tx
from .validation import validate_jsonschema
import json
from pathlib import Path

router = APIRouter(prefix="/manifests", tags=["manifests"])

# validate manifests against repo JSON schema at runtime
MANIFEST_SCHEMA_PATH = Path(__file__).resolve().parents[2] / "spec" / "jsonschema" / "manifest.schema.json"
MANIFEST_SCHEMA = json.loads(MANIFEST_SCHEMA_PATH.read_text(encoding="utf-8"))

@router.post("")
def register_manifest(m: Manifest):
    errors = validate_jsonschema(MANIFEST_SCHEMA, m.model_dump())
    if errors:
        raise HTTPException(status_code=400, detail={"message":"manifest schema validation failed","errors":errors})

    with tx() as cur:
        cur.execute(
            """INSERT INTO manifests (name, version, manifest)
               VALUES (%s,%s,%s::jsonb)
               ON CONFLICT (name) DO UPDATE SET
                 version=EXCLUDED.version,
                 manifest=EXCLUDED.manifest,
                 updated_at=NOW()""",
            (m.name, m.version, _json(m.model_dump())),
        )
    return {"ok": True, "name": m.name, "version": m.version}

@router.get("")
def list_manifests():
    with tx() as cur:
        cur.execute("SELECT name, version, created_at, updated_at FROM manifests ORDER BY name ASC")
        return {"manifests": list(cur.fetchall())}

@router.get("/{name}")
def get_manifest(name: str):
    with tx() as cur:
        cur.execute("SELECT name, version, manifest FROM manifests WHERE name=%s", (name,))
        row = cur.fetchone()
        if not row:
            raise HTTPException(status_code=404, detail="manifest not found")
        return {"name": row["name"], "version": row["version"], "manifest": row["manifest"]}

def _json(obj):
    return json.dumps(obj)
