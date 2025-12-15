from __future__ import annotations
from typing import Any, Dict, List
import uuid
from datetime import datetime, timezone
from .db import tx

def utcnow() -> str:
    return datetime.now(timezone.utc).isoformat()

def write_event(*, artifact_id: str, event_type: str, actor_type: str, actor_id: str, metadata: Dict[str, Any]) -> str:
    event_id = str(uuid.uuid4())
    ts = utcnow()
    with tx() as cur:
        cur.execute(
            """INSERT INTO artifact_events (event_id, artifact_id, event_type, actor_type, actor_id, timestamp, metadata)
               VALUES (%s,%s,%s,%s,%s,%s,%s::jsonb)""",
            (event_id, artifact_id, event_type, actor_type, actor_id, ts, _json(metadata)),
        )
    return event_id

def list_events(artifact_id: str) -> List[Dict[str, Any]]:
    with tx() as cur:
        cur.execute(
            """SELECT event_id, artifact_id, event_type, actor_type, actor_id, timestamp, metadata
               FROM artifact_events WHERE artifact_id=%s ORDER BY timestamp ASC""",
            (artifact_id,),
        )
        return list(cur.fetchall())

def _json(obj: Any) -> str:
    import json
    return json.dumps(obj)
