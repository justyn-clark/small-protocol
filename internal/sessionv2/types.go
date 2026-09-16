package sessionv2

import "errors"

const (
	ProfileVersion = "2.0.0"
	MaxRecordBytes = 1 << 20
)

var (
	ErrNotV2         = errors.New("workspace is not SMALL profile 2.0.0")
	ErrConflict      = errors.New("semantic conflicts remain")
	ErrCorruption    = errors.New("record identity corruption")
	ErrStaleFrontier = errors.New("stale state frontier")
	ErrAmbiguous     = errors.New("ambiguous session selection")
)

type Profile struct {
	SmallVersion      string `json:"small_version"`
	ProjectID         string `json:"project_id"`
	LineageID         string `json:"lineage_id"`
	Mode              string `json:"mode"`
	PolicyRevision    string `json:"policy_revision"`
	IntentDigest      string `json:"intent_digest,omitempty"`
	ConstraintsDigest string `json:"constraints_digest,omitempty"`
	ImportID          string `json:"import_id,omitempty"`
}

type Session struct {
	SmallVersion     string `json:"small_version"`
	ProjectID        string `json:"project_id"`
	LineageID        string `json:"lineage_id"`
	SessionID        string `json:"session_id"`
	Label            string `json:"label,omitempty"`
	ActorLabel       string `json:"actor_label,omitempty"`
	ModelLabel       string `json:"model_label,omitempty"`
	ParentSessionID  string `json:"parent_session_id,omitempty"`
	FromHandoff      string `json:"from_handoff,omitempty"`
	ObservedFrontier string `json:"observed_frontier"`
	ToolVersion      string `json:"tool_version"`
	CreatedAt        string `json:"created_at"`
	Digest           string `json:"digest,omitempty"`
}

type Event struct {
	SmallVersion   string         `json:"small_version"`
	ProjectID      string         `json:"project_id"`
	LineageID      string         `json:"lineage_id"`
	SessionID      string         `json:"session_id"`
	EventID        string         `json:"event_id"`
	Kind           string         `json:"kind"`
	Sequence       int64          `json:"sequence"`
	PreviousEvent  string         `json:"previous_event,omitempty"`
	Parents        []string       `json:"parents,omitempty"`
	OccurredAt     string         `json:"occurred_at"`
	TaskID         string         `json:"task_id,omitempty"`
	TaskRevision   int64          `json:"task_revision,omitempty"`
	PolicyRevision string         `json:"policy_revision,omitempty"`
	SourceDigest   string         `json:"source_digest,omitempty"`
	Payload        map[string]any `json:"payload"`
	Digest         string         `json:"digest,omitempty"`
}

type Receipt struct {
	SmallVersion   string         `json:"small_version"`
	ProjectID      string         `json:"project_id"`
	ReceiptID      string         `json:"receipt_id"`
	Strength       string         `json:"strength"`
	Availability   string         `json:"availability"`
	Producer       map[string]any `json:"producer,omitempty"`
	TaskID         string         `json:"task_id,omitempty"`
	TaskRevision   int64          `json:"task_revision,omitempty"`
	PolicyRevision string         `json:"policy_revision,omitempty"`
	SourceDigest   string         `json:"source_digest,omitempty"`
	Validator      string         `json:"validator,omitempty"`
	Outcome        string         `json:"outcome,omitempty"`
	ArtifactPath   string         `json:"artifact_path,omitempty"`
	ArtifactDigest string         `json:"artifact_digest,omitempty"`
	ArtifactSize   int64          `json:"artifact_size,omitempty"`
	Metadata       map[string]any `json:"metadata,omitempty"`
	Digest         string         `json:"digest,omitempty"`
}

type Conflict struct {
	ID         string   `json:"id"`
	Kind       string   `json:"kind"`
	TaskID     string   `json:"task_id,omitempty"`
	Heads      []string `json:"heads"`
	Message    string   `json:"message"`
	ResolvedBy string   `json:"resolved_by,omitempty"`
}

type TaskState struct {
	ID              string   `json:"id"`
	Alias           string   `json:"alias,omitempty"`
	Title           string   `json:"title,omitempty"`
	Revision        int64    `json:"revision"`
	Status          string   `json:"status"`
	Acceptance      []string `json:"acceptance,omitempty"`
	Dependencies    []string `json:"dependencies,omitempty"`
	PolicyRevision  string   `json:"policy_revision,omitempty"`
	SourceDigest    string   `json:"source_digest,omitempty"`
	EvidenceDigests []string `json:"evidence_digests,omitempty"`
	Attestations    []string `json:"attestations,omitempty"`
}

type State struct {
	Profile            Profile              `json:"profile"`
	Frontier           string               `json:"frontier"`
	EventCount         int                  `json:"event_count"`
	SessionCount       int                  `json:"session_count"`
	Tasks              map[string]TaskState `json:"tasks"`
	Conflicts          []Conflict           `json:"conflicts"`
	Handoffs           []Event              `json:"handoffs,omitempty"`
	ClosedSessions     map[string]bool      `json:"closed_sessions,omitempty"`
	SupersededSessions map[string]bool      `json:"superseded_sessions,omitempty"`
}

type Store struct {
	BaseDir                 string
	Profile                 Profile
	Sessions                map[string]Session
	Events                  map[string]Event
	Receipts                map[string]Receipt
	PolicyIntentDigest      string
	PolicyConstraintsDigest string
}

var AllowedEventKinds = map[string]bool{
	"session_started": true, "session_closed": true,
	"task_created": true, "task_updated": true, "task_transitioned": true,
	"command_recorded": true, "handoff_recorded": true,
	"mode_changed": true, "policy_changed": true, "ownership_claimed": true, "ownership_taken_over": true,
	"conflict_resolved": true, "v1_imported": true,
}
