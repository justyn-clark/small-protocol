from __future__ import annotations
from fastapi import APIRouter, HTTPException
from .schemas import ArtifactCreate, ArtifactUpdate, TransitionRequest, RollbackRequest
from .db import tx
from .validation import validate_jsonschema
from .policy import enforce_transition, enforce_publish_gate
from .lineage import write_event, list_events, utcnow
import json

router = APIRouter(prefix="/artifacts", tags=["artifacts"])

def _json(obj):
    return json.dumps(obj)

def _get_manifest(cur, name: str):
    cur.execute("SELECT manifest FROM manifests WHERE name=%s", (name,))
    row = cur.fetchone()
    if not row:
        raise HTTPException(status_code=404, detail="manifest not found")
    return row["manifest"]

def _get_schema(cur, schema_ref: str):
    cur.execute("SELECT schema FROM schemas WHERE schema_ref=%s", (schema_ref,))
    row = cur.fetchone()
    if not row:
        raise HTTPException(status_code=404, detail="schema not found")
    return row["schema"]

def _get_artifact(cur, artifact_id: str):
    cur.execute("SELECT * FROM artifacts WHERE id=%s", (artifact_id,))
    row = cur.fetchone()
    if not row:
        raise HTTPException(status_code=404, detail="artifact not found")
    return row

def _snapshot_version(cur, row):
    cur.execute(
        """INSERT INTO artifact_versions (artifact_id, version, schema_ref, state, data, created_at)
           VALUES (%s,%s,%s,%s,%s::jsonb,NOW())""",
        (row["id"], row["version"], row["schema_ref"], row["state"], _json(row["data"]))
    )

@router.post("")
def create_artifact(req: ArtifactCreate):
    ts = utcnow()
    with tx() as cur:
        # validate schema exists and data is valid (create should be valid)
        schema = _get_schema(cur, req.schema_ref)
        errors = validate_jsonschema(schema, req.data)
        if errors:
            raise HTTPException(status_code=400, detail={"message":"schema validation failed","errors":errors})

        # create artifact at version 1
        cur.execute(
            """INSERT INTO artifacts (id, type, schema_ref, state, version, data, created_at, updated_at)
               VALUES (%s,%s,%s,%s,1,%s::jsonb,%s,%s)""",
            (req.id, req.type, req.schema_ref, req.state, _json(req.data), ts, ts),
        )
        cur.execute("SELECT * FROM artifacts WHERE id=%s", (req.id,))
        row = cur.fetchone()
        _snapshot_version(cur, row)

    write_event(
        artifact_id=req.id,
        event_type="artifact.created",
        actor_type=req.actor_type,
        actor_id=req.actor_id,
        metadata={"schema_ref": req.schema_ref, "state": req.state, "version": 1},
    )
    return row

@router.get("/{artifact_id}")
def get_artifact(artifact_id: str):
    with tx() as cur:
        row = _get_artifact(cur, artifact_id)
        return row

@router.patch("/{artifact_id}")
def update_artifact(artifact_id: str, req: ArtifactUpdate):
    ts = utcnow()
    with tx() as cur:
        row = _get_artifact(cur, artifact_id)
        schema = _get_schema(cur, row["schema_ref"])
        # allow invalid data to be stored (demonstration), but validation endpoint will catch it.
        # If you want stricter behavior: validate here and reject.
        new_version = row["version"] + 1
        cur.execute(
            """UPDATE artifacts SET data=%s::jsonb, version=%s, updated_at=%s WHERE id=%s""",
            (_json(req.data), new_version, ts, artifact_id),
        )
        cur.execute("SELECT * FROM artifacts WHERE id=%s", (artifact_id,))
        updated = cur.fetchone()
        _snapshot_version(cur, updated)

    write_event(
        artifact_id=artifact_id,
        event_type="artifact.updated",
        actor_type=req.actor_type,
        actor_id=req.actor_id,
        metadata={"version": updated["version"]},
    )
    return updated

@router.post("/{artifact_id}/validate")
def validate_artifact(artifact_id: str):
    with tx() as cur:
        row = _get_artifact(cur, artifact_id)
        schema = _get_schema(cur, row["schema_ref"])
        errors = validate_jsonschema(schema, row["data"])
        ok = len(errors) == 0

    write_event(
        artifact_id=artifact_id,
        event_type="artifact.validated",
        actor_type="system",
        actor_id="validator",
        metadata={"ok": ok, "errors": errors, "schema_ref": row["schema_ref"], "version": row["version"]},
    )
    return {"ok": ok, "schema_ref": row["schema_ref"], "errors": errors}

@router.post("/{artifact_id}/transition")
def transition_artifact(artifact_id: str, req: TransitionRequest):
    ts = utcnow()
    with tx() as cur:
        row = _get_artifact(cur, artifact_id)
        manifest = _get_manifest(cur, req.manifest_name)

        # enforce artifact type + allowed states
        if row["type"] not in manifest["artifact_types"]:
            raise HTTPException(status_code=403, detail="artifact type not governed by manifest")

        from_state = row["state"]
        to_state = req.to_state

        try:
            enforce_transition(_manifest_to_obj(manifest), from_state, to_state)
            enforce_publish_gate(_manifest_to_obj(manifest), to_state, from_state)
        except PermissionError as e:
            raise HTTPException(status_code=403, detail=str(e))

        # validation gate on transition: run validators that reference this artifact schema_ref
        schema = _get_schema(cur, row["schema_ref"])
        errors = validate_jsonschema(schema, row["data"])
        if errors:
            raise HTTPException(status_code=400, detail={"message":"cannot transition: schema validation failed","errors":errors})

        new_version = row["version"] + 1
        cur.execute(
            """UPDATE artifacts SET state=%s, version=%s, updated_at=%s WHERE id=%s""",
            (to_state, new_version, ts, artifact_id),
        )
        cur.execute("SELECT * FROM artifacts WHERE id=%s", (artifact_id,))
        updated = cur.fetchone()
        _snapshot_version(cur, updated)

    write_event(
        artifact_id=artifact_id,
        event_type="artifact.transitioned",
        actor_type=req.actor_type,
        actor_id=req.actor_id,
        metadata={"from": from_state, "to": to_state, "version": updated["version"], "manifest": req.manifest_name},
    )
    return updated

@router.get("/{artifact_id}/versions")
def list_versions(artifact_id: str):
    with tx() as cur:
        cur.execute(
            """SELECT artifact_id, version, schema_ref, state, created_at
               FROM artifact_versions WHERE artifact_id=%s ORDER BY version ASC""",
            (artifact_id,),
        )
        return {"versions": list(cur.fetchall())}

@router.post("/{artifact_id}/rollback")
def rollback_artifact(artifact_id: str, req: RollbackRequest):
    ts = utcnow()
    with tx() as cur:
        row = _get_artifact(cur, artifact_id)
        cur.execute(
            """SELECT artifact_id, version, schema_ref, state, data
               FROM artifact_versions WHERE artifact_id=%s AND version=%s""",
            (artifact_id, req.target_version),
        )
        snap = cur.fetchone()
        if not snap:
            raise HTTPException(status_code=404, detail="target version not found")

        new_version = row["version"] + 1
        cur.execute(
            """UPDATE artifacts SET schema_ref=%s, state=%s, data=%s::jsonb, version=%s, updated_at=%s WHERE id=%s""",
            (snap["schema_ref"], snap["state"], _json(snap["data"]), new_version, ts, artifact_id),
        )
        cur.execute("SELECT * FROM artifacts WHERE id=%s", (artifact_id,))
        updated = cur.fetchone()
        _snapshot_version(cur, updated)

    write_event(
        artifact_id=artifact_id,
        event_type="artifact.rolled_back",
        actor_type=req.actor_type,
        actor_id=req.actor_id,
        metadata={"target_version": req.target_version, "new_version": updated["version"]},
    )
    return updated

@router.get("/{artifact_id}/events")
def get_events(artifact_id: str):
    return {"events": list_events(artifact_id)}

def _manifest_to_obj(manifest: dict):
    from .schemas import Manifest, ManifestValidator
    return Manifest(**manifest)
