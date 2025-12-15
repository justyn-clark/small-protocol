from __future__ import annotations
from typing import Tuple
from .schemas import Manifest

def enforce_state_allowed(manifest: Manifest, state: str) -> None:
    if state not in manifest.allowed_states:
        raise PermissionError(f"state '{state}' not allowed by manifest '{manifest.name}'")

def enforce_transition(manifest: Manifest, from_state: str, to_state: str) -> None:
    enforce_state_allowed(manifest, from_state)
    enforce_state_allowed(manifest, to_state)
    allowed = manifest.transitions.get(from_state, [])
    if to_state not in allowed:
        raise PermissionError(f"transition '{from_state}' -> '{to_state}' not allowed by manifest '{manifest.name}'")

def enforce_publish_gate(manifest: Manifest, to_state: str, from_state: str) -> None:
    # Example: "published" requires coming from "approved" (already enforced by transitions),
    # but this is a hook for additional policy checks.
    if to_state == "published" and from_state != "approved":
        raise PermissionError("publish gate: must be in 'approved' before publishing")
