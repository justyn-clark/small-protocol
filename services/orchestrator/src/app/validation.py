from __future__ import annotations
from typing import Any, Dict, List
from jsonschema import Draft202012Validator

def validate_jsonschema(schema: Dict[str, Any], data: Dict[str, Any]) -> List[Dict[str, Any]]:
    v = Draft202012Validator(schema)
    errors = []
    for e in sorted(v.iter_errors(data), key=lambda x: x.path):
        errors.append({
            "message": e.message,
            "path": list(e.path),
            "schema_path": list(e.schema_path),
        })
    return errors
