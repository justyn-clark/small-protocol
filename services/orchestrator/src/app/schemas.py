from __future__ import annotations
from typing import Any, Dict, List, Optional, Literal
from pydantic import BaseModel, Field

ActorType = Literal["human","agent","system"]

class SchemaRegistryItem(BaseModel):
    schema_ref: str
    version: int = Field(ge=1)
    schema: Dict[str, Any]

class ManifestValidator(BaseModel):
    id: str
    type: Literal["jsonschema"]
    schema_ref: Optional[str] = None

class Manifest(BaseModel):
    name: str
    version: int = Field(ge=1)
    artifact_types: List[str]
    allowed_states: List[str]
    transitions: Dict[str, List[str]]
    validators: List[ManifestValidator] = Field(default_factory=list)
    agent_permissions: Dict[str, List[str]] = Field(default_factory=dict)
    publish_targets: Optional[List[Dict[str, Any]]] = None

class Artifact(BaseModel):
    id: str
    type: str
    schema_ref: str
    state: str
    version: int
    data: Dict[str, Any]
    blob_ref: Optional[str] = None
    created_at: str
    updated_at: str

class ArtifactCreate(BaseModel):
    id: str
    type: str
    schema_ref: str
    state: str
    data: Dict[str, Any]
    actor_type: ActorType = "human"
    actor_id: str = "unknown"

class ArtifactUpdate(BaseModel):
    data: Dict[str, Any]
    actor_type: ActorType = "human"
    actor_id: str = "unknown"

class TransitionRequest(BaseModel):
    manifest_name: str
    to_state: str
    actor_type: ActorType = "human"
    actor_id: str = "unknown"

class RollbackRequest(BaseModel):
    target_version: int = Field(ge=1)
    actor_type: ActorType = "human"
    actor_id: str = "unknown"

class ValidationResult(BaseModel):
    ok: bool
    schema_ref: str
    errors: List[Dict[str, Any]] = Field(default_factory=list)

class Event(BaseModel):
    event_id: str
    artifact_id: str
    event_type: str
    actor_type: ActorType
    actor_id: str
    timestamp: str
    metadata: Dict[str, Any]
