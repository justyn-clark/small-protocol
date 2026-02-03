package small

import (
	"testing"
)

func TestCheckInvariants_Version(t *testing.T) {
	tests := []struct {
		name       string
		version    interface{}
		wantErrors int
	}{
		{"correct version", ProtocolVersion, 0},
		{"wrong version", "0.1", 1},     // only v1.0.0 is supported
		{"wrong version 1.0", "1.0", 1}, // must be exactly "1.0.0"
		{"missing version", nil, 1},
		{"empty version", "", 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			artifacts := map[string]*Artifact{
				"progress": {
					Path: "test/progress.small.yml",
					Type: "progress",
					Data: map[string]interface{}{
						"small_version": tt.version,
						"owner":         "agent",
						"entries":       []interface{}{},
					},
				},
			}

			violations := CheckInvariants(artifacts, false)
			versionErrors := 0
			for _, v := range violations {
				if contains(v.Message, "small_version") {
					versionErrors++
				}
			}
			if versionErrors != tt.wantErrors {
				t.Errorf("got %d version errors, want %d", versionErrors, tt.wantErrors)
			}
		})
	}
}

func TestCheckInvariants_Owner(t *testing.T) {
	tests := []struct {
		name         string
		artifactType string
		owner        string
		wantErr      bool
	}{
		{"intent with human owner", "intent", "human", false},
		{"intent with agent owner", "intent", "agent", true},
		{"constraints with human owner", "constraints", "human", false},
		{"constraints with agent owner", "constraints", "agent", true},
		{"plan with agent owner", "plan", "agent", false},
		{"plan with human owner", "plan", "human", true},
		{"progress with agent owner", "progress", "agent", false},
		{"progress with human owner", "progress", "human", true},
		{"handoff with agent owner", "handoff", "agent", false},
		{"handoff with human owner", "handoff", "human", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := makeValidArtifact(tt.artifactType)
			data["owner"] = tt.owner

			artifacts := map[string]*Artifact{
				tt.artifactType: {
					Path: "test/" + tt.artifactType + ".small.yml",
					Type: tt.artifactType,
					Data: data,
				},
			}

			violations := CheckInvariants(artifacts, false)
			hasOwnerError := false
			for _, v := range violations {
				if contains(v.Message, "owner") {
					hasOwnerError = true
					break
				}
			}
			if hasOwnerError != tt.wantErr {
				t.Errorf("owner error = %v, want %v", hasOwnerError, tt.wantErr)
			}
		})
	}
}

func TestCheckInvariants_IntentMissingIntent(t *testing.T) {
	artifacts := map[string]*Artifact{
		"intent": {
			Path: "test/intent.small.yml",
			Type: "intent",
			Data: map[string]interface{}{
				"small_version":    ProtocolVersion,
				"owner":            "human",
				"intent":           "", // Empty intent should fail
				"scope":            map[string]interface{}{"include": []interface{}{}, "exclude": []interface{}{}},
				"success_criteria": []interface{}{},
			},
		},
	}

	violations := CheckInvariants(artifacts, false)
	hasIntentError := false
	for _, v := range violations {
		if contains(v.Message, "intent.intent") {
			hasIntentError = true
			break
		}
	}
	if !hasIntentError {
		t.Error("expected error for empty intent, got none")
	}
}

func TestCheckInvariants_ProgressWithoutEvidence(t *testing.T) {
	artifacts := map[string]*Artifact{
		"progress": {
			Path: "test/progress.small.yml",
			Type: "progress",
			Data: map[string]interface{}{
				"small_version": ProtocolVersion,
				"owner":         "agent",
				"entries": []interface{}{
					map[string]interface{}{
						"task_id":   "task-1",
						"timestamp": "2025-01-01T00:00:00.000000000Z",
						// No evidence field - should fail
					},
				},
			},
		},
	}

	violations := CheckInvariants(artifacts, false)
	hasEvidenceError := false
	for _, v := range violations {
		if contains(v.Message, "must have at least one evidence field") {
			hasEvidenceError = true
			break
		}
	}
	if !hasEvidenceError {
		t.Error("expected error for missing evidence, got none")
	}
}

func TestCheckInvariants_ProgressTimestampRequiresFractional(t *testing.T) {
	artifacts := map[string]*Artifact{
		"progress": {
			Path: "test/progress.small.yml",
			Type: "progress",
			Data: map[string]interface{}{
				"small_version": ProtocolVersion,
				"owner":         "agent",
				"entries": []interface{}{
					map[string]interface{}{
						"task_id":   "task-1",
						"timestamp": "2025-01-01T00:00:00Z",
						"evidence":  "missing fractional",
					},
				},
			},
		},
	}

	violations := CheckInvariants(artifacts, false)
	hasFractionalError := false
	for _, v := range violations {
		if contains(v.Message, "fractional seconds") {
			hasFractionalError = true
			break
		}
	}
	if !hasFractionalError {
		t.Error("expected error for missing fractional seconds, got none")
	}
}

func TestCheckInvariants_ProgressTimestampEqual(t *testing.T) {
	artifacts := map[string]*Artifact{
		"progress": {
			Path: "test/progress.small.yml",
			Type: "progress",
			Data: map[string]interface{}{
				"small_version": ProtocolVersion,
				"owner":         "agent",
				"entries": []interface{}{
					map[string]interface{}{
						"task_id":   "task-1",
						"timestamp": "2025-01-01T00:00:00.000000001Z",
						"evidence":  "first",
					},
					map[string]interface{}{
						"task_id":   "task-2",
						"timestamp": "2025-01-01T00:00:00.000000001Z",
						"evidence":  "duplicate",
					},
				},
			},
		},
	}

	violations := CheckInvariants(artifacts, false)
	hasOrderingError := false
	for _, v := range violations {
		if contains(v.Message, "must be after previous entry") {
			hasOrderingError = true
			break
		}
	}
	if !hasOrderingError {
		t.Error("expected error for equal timestamps, got none")
	}
}

func TestCheckInvariants_ProgressTimestampDecreasing(t *testing.T) {
	artifacts := map[string]*Artifact{
		"progress": {
			Path: "test/progress.small.yml",
			Type: "progress",
			Data: map[string]interface{}{
				"small_version": ProtocolVersion,
				"owner":         "agent",
				"entries": []interface{}{
					map[string]interface{}{
						"task_id":   "task-1",
						"timestamp": "2025-01-01T00:00:01.000000000Z",
						"evidence":  "later",
					},
					map[string]interface{}{
						"task_id":   "task-2",
						"timestamp": "2025-01-01T00:00:00.000000000Z",
						"evidence":  "earlier",
					},
				},
			},
		},
	}

	violations := CheckInvariants(artifacts, false)
	hasOrderingError := false
	for _, v := range violations {
		if contains(v.Message, "must be after previous entry") {
			hasOrderingError = true
			break
		}
	}
	if !hasOrderingError {
		t.Error("expected error for decreasing timestamps, got none")
	}
}

func TestCheckInvariants_ConstraintsEmpty(t *testing.T) {
	artifacts := map[string]*Artifact{
		"constraints": {
			Path: "test/constraints.small.yml",
			Type: "constraints",
			Data: map[string]interface{}{
				"small_version": ProtocolVersion,
				"owner":         "human",
				"constraints":   []interface{}{}, // Empty constraints should fail
			},
		},
	}

	violations := CheckInvariants(artifacts, false)
	hasConstraintsError := false
	for _, v := range violations {
		if contains(v.Message, "constraints.constraints") {
			hasConstraintsError = true
			break
		}
	}
	if !hasConstraintsError {
		t.Error("expected error for empty constraints, got none")
	}
}

func TestCheckInvariants_UnknownTopLevelKey(t *testing.T) {
	artifacts := map[string]*Artifact{
		"progress": {
			Path: "test/progress.small.yml",
			Type: "progress",
			Data: map[string]interface{}{
				"small_version": ProtocolVersion,
				"owner":         "agent",
				"entries":       []interface{}{},
				"typo_field":    "should fail", // Unknown key
			},
		},
	}

	violations := CheckInvariants(artifacts, false)
	hasKeyError := false
	for _, v := range violations {
		if contains(v.Message, "unknown top-level key") {
			hasKeyError = true
			break
		}
	}
	if !hasKeyError {
		t.Error("expected error for unknown top-level key, got none")
	}
}

func TestCheckInvariants_PlanTasksEmpty(t *testing.T) {
	artifacts := map[string]*Artifact{
		"plan": {
			Path: "test/plan.small.yml",
			Type: "plan",
			Data: map[string]interface{}{
				"small_version": ProtocolVersion,
				"owner":         "agent",
				"tasks":         []interface{}{}, // Empty tasks should fail
			},
		},
	}

	violations := CheckInvariants(artifacts, false)
	hasTasksError := false
	for _, v := range violations {
		if contains(v.Message, "plan.tasks") {
			hasTasksError = true
			break
		}
	}
	if !hasTasksError {
		t.Error("expected error for empty tasks, got none")
	}
}

func TestCheckInvariants_PlanTaskMissingTitle(t *testing.T) {
	artifacts := map[string]*Artifact{
		"plan": {
			Path: "test/plan.small.yml",
			Type: "plan",
			Data: map[string]interface{}{
				"small_version": ProtocolVersion,
				"owner":         "agent",
				"tasks": []interface{}{
					map[string]interface{}{
						"id":    "task-1",
						"title": "",
					},
				},
			},
		},
	}

	violations := CheckInvariants(artifacts, false)
	hasTitleError := false
	for _, v := range violations {
		if contains(v.Message, "title") {
			hasTitleError = true
			break
		}
	}
	if !hasTitleError {
		t.Error("expected error for empty task title, got none")
	}
}

func TestCheckInvariants_CompletedTaskMissingProgressEvidence(t *testing.T) {
	artifacts := map[string]*Artifact{
		"plan": {
			Path: "test/plan.small.yml",
			Type: "plan",
			Data: map[string]interface{}{
				"small_version": ProtocolVersion,
				"owner":         "agent",
				"tasks": []interface{}{
					map[string]interface{}{
						"id":     "task-evidence-rule",
						"title":  "Record progress before completing tasks",
						"status": "completed",
					},
				},
			},
		},
		"progress": {
			Path: "test/progress.small.yml",
			Type: "progress",
			Data: map[string]interface{}{
				"small_version": ProtocolVersion,
				"owner":         "agent",
				"entries":       []interface{}{},
			},
		},
	}

	violations := CheckInvariants(artifacts, true)
	hasMissingProgress := false
	for _, v := range violations {
		if contains(v.Message, "strict invariant S1 failed") {
			hasMissingProgress = true
			break
		}
	}
	if !hasMissingProgress {
		t.Error("expected strict violation for completed task without progress evidence")
	}
}

func TestCheckInvariants_CompletedTaskHasProgressEvidence(t *testing.T) {
	artifacts := map[string]*Artifact{
		"plan": {
			Path: "test/plan.small.yml",
			Type: "plan",
			Data: map[string]interface{}{
				"small_version": ProtocolVersion,
				"owner":         "agent",
				"tasks": []interface{}{
					map[string]interface{}{
						"id":     "task-evidence-rule",
						"title":  "Record progress before completing tasks",
						"status": "completed",
					},
				},
			},
		},
		"progress": {
			Path: "test/progress.small.yml",
			Type: "progress",
			Data: map[string]interface{}{
				"small_version": ProtocolVersion,
				"owner":         "agent",
				"entries": []interface{}{
					map[string]interface{}{
						"task_id":   "task-evidence-rule",
						"status":    "completed",
						"timestamp": "2025-01-01T00:00:00.000000000Z",
						"evidence":  "Linked completed tasks to progress entries",
					},
				},
			},
		},
	}

	violations := CheckInvariants(artifacts, true)
	for _, v := range violations {
		if contains(v.Message, "strict invariant S1 failed") {
			t.Fatalf("unexpected strict progress violation: %s", v.Message)
		}
	}
}

func TestCheckInvariants_CompletedTaskProgressEntryEmptyNote(t *testing.T) {
	artifacts := map[string]*Artifact{
		"plan": {
			Path: "test/plan.small.yml",
			Type: "plan",
			Data: map[string]interface{}{
				"small_version": ProtocolVersion,
				"owner":         "agent",
				"tasks": []interface{}{
					map[string]interface{}{
						"id":     "task-evidence-rule",
						"title":  "Record progress before completing tasks",
						"status": "completed",
					},
				},
			},
		},
		"progress": {
			Path: "test/progress.small.yml",
			Type: "progress",
			Data: map[string]interface{}{
				"small_version": ProtocolVersion,
				"owner":         "agent",
				"entries": []interface{}{
					map[string]interface{}{
						"task_id":   "task-evidence-rule",
						"status":    "completed",
						"timestamp": "2025-01-02T00:00:00Z",
						"notes":     "   ",
					},
				},
			},
		},
	}

	violations := CheckInvariants(artifacts, true)
	hasMissingProgress := false
	for _, v := range violations {
		if contains(v.Message, "strict invariant S1 failed") {
			hasMissingProgress = true
			break
		}
	}
	if !hasMissingProgress {
		t.Error("expected strict violation for completed task with empty note")
	}
}

func TestCheckInvariants_CompletedTaskProgressEntryInvalidTimestamp(t *testing.T) {
	artifacts := map[string]*Artifact{
		"plan": {
			Path: "test/plan.small.yml",
			Type: "plan",
			Data: map[string]interface{}{
				"small_version": ProtocolVersion,
				"owner":         "agent",
				"tasks": []interface{}{
					map[string]interface{}{
						"id":     "task-evidence-rule",
						"title":  "Record progress before completing tasks",
						"status": "completed",
					},
				},
			},
		},
		"progress": {
			Path: "test/progress.small.yml",
			Type: "progress",
			Data: map[string]interface{}{
				"small_version": ProtocolVersion,
				"owner":         "agent",
				"entries": []interface{}{
					map[string]interface{}{
						"task_id":   "task-evidence-rule",
						"status":    "completed",
						"timestamp": "not-a-time",
						"notes":     "completed",
					},
				},
			},
		},
	}

	violations := CheckInvariants(artifacts, true)
	hasInvalidTimestamp := false
	for _, v := range violations {
		if contains(v.Message, "timestamp") && contains(v.Message, "RFC3339Nano") {
			hasInvalidTimestamp = true
			break
		}
	}
	if !hasInvalidTimestamp {
		t.Error("expected violation for completed task with invalid timestamp")
	}
}

func TestCheckInvariants_StrictModePlanProgressEvidence(t *testing.T) {
	artifacts := map[string]*Artifact{
		"plan": {
			Path: "test/plan.small.yml",
			Type: "plan",
			Data: map[string]interface{}{
				"small_version": ProtocolVersion,
				"owner":         "agent",
				"tasks": []interface{}{
					map[string]interface{}{
						"id":     "task-completed",
						"title":  "Done task",
						"status": "completed",
					},
					map[string]interface{}{
						"id":     "task-blocked",
						"title":  "Blocked task",
						"status": "blocked",
					},
				},
			},
		},
		"progress": {
			Path: "test/progress.small.yml",
			Type: "progress",
			Data: map[string]interface{}{
				"small_version": ProtocolVersion,
				"owner":         "agent",
				"entries": []interface{}{
					map[string]interface{}{
						"task_id":   "task-blocked",
						"status":    "blocked",
						"timestamp": "2025-01-01T00:00:00.000000001Z",
						"evidence":  " ",
						"notes":     "",
					},
				},
			},
		},
	}

	violations := CheckInvariants(artifacts, true)
	var s1Violation string
	for _, v := range violations {
		if contains(v.Message, "strict invariant S1 failed") {
			s1Violation = v.Message
			break
		}
	}
	if s1Violation == "" {
		t.Fatal("expected strict mode S1 violation")
	}
	if !contains(s1Violation, "task-completed") || !contains(s1Violation, "Done task") || !contains(s1Violation, "no progress entry") {
		t.Fatalf("expected completed task details in violation: %s", s1Violation)
	}
	if !contains(s1Violation, "task-blocked") || !contains(s1Violation, "Blocked task") || !contains(s1Violation, "empty evidence/notes") {
		t.Fatalf("expected blocked task details in violation: %s", s1Violation)
	}
}

func TestCheckInvariants_StrictModeProgressTaskIDs(t *testing.T) {
	currentReplayID := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	artifacts := map[string]*Artifact{
		"plan": {
			Path: "test/plan.small.yml",
			Type: "plan",
			Data: map[string]interface{}{
				"small_version": ProtocolVersion,
				"owner":         "agent",
				"tasks": []interface{}{
					map[string]interface{}{
						"id":    "task-1",
						"title": "Known task",
					},
				},
			},
		},
		"progress": {
			Path: "test/progress.small.yml",
			Type: "progress",
			Data: map[string]interface{}{
				"small_version": ProtocolVersion,
				"owner":         "agent",
				"entries": []interface{}{
					map[string]interface{}{
						"task_id":   "task-unknown",
						"status":    "completed",
						"timestamp": "2025-01-01T00:00:00.000000001Z",
						"evidence":  "unexpected",
						"replayId":  currentReplayID,
					},
					map[string]interface{}{
						"task_id":   "meta/reconcile-plan",
						"status":    "completed",
						"timestamp": "2025-01-01T00:00:00.000000002Z",
						"evidence":  "Reconciled plan to match completed work",
						"replayId":  currentReplayID,
					},
				},
			},
		},
		"handoff": {
			Path: "test/handoff.small.yml",
			Type: "handoff",
			Data: map[string]interface{}{
				"small_version": ProtocolVersion,
				"owner":         "agent",
				"summary":       "Test handoff",
				"resume": map[string]interface{}{
					"current_task_id": "",
					"next_steps":      []interface{}{},
				},
				"links": []interface{}{},
				"replayId": map[string]interface{}{
					"value":  currentReplayID,
					"source": "auto",
				},
			},
		},
	}

	violations := CheckInvariants(artifacts, true)
	hasS2 := false
	for _, v := range violations {
		if contains(v.Message, "strict invariant S2 failed") {
			if contains(v.Message, "task-unknown") && contains(v.Message, currentReplayID) {
				hasS2 = true
			}
		}
	}
	if !hasS2 {
		t.Error("expected strict mode S2 violation for unknown task id in replay scope")
	}
}

func TestCheckInvariants_StrictModeProgressTaskIDsIgnoresOtherReplayID(t *testing.T) {
	currentReplayID := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	otherReplayID := "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	artifacts := map[string]*Artifact{
		"plan": {
			Path: "test/plan.small.yml",
			Type: "plan",
			Data: map[string]interface{}{
				"small_version": ProtocolVersion,
				"owner":         "agent",
				"tasks": []interface{}{
					map[string]interface{}{
						"id":    "task-1",
						"title": "Known task",
					},
				},
			},
		},
		"progress": {
			Path: "test/progress.small.yml",
			Type: "progress",
			Data: map[string]interface{}{
				"small_version": ProtocolVersion,
				"owner":         "agent",
				"entries": []interface{}{
					map[string]interface{}{
						"task_id":   "task-unknown",
						"status":    "completed",
						"timestamp": "2025-01-01T00:00:00.000000001Z",
						"evidence":  "unexpected",
						"replayId":  otherReplayID,
					},
				},
			},
		},
		"handoff": {
			Path: "test/handoff.small.yml",
			Type: "handoff",
			Data: map[string]interface{}{
				"small_version": ProtocolVersion,
				"owner":         "agent",
				"summary":       "Test handoff",
				"resume": map[string]interface{}{
					"current_task_id": "",
					"next_steps":      []interface{}{},
				},
				"links": []interface{}{},
				"replayId": map[string]interface{}{
					"value":  currentReplayID,
					"source": "auto",
				},
			},
		},
	}

	violations := CheckInvariants(artifacts, true)
	for _, v := range violations {
		if contains(v.Message, "strict invariant S2 failed") {
			t.Fatalf("unexpected strict mode S2 violation: %s", v.Message)
		}
	}
}

func TestCheckInvariants_StrictModeHandoffTaskReference(t *testing.T) {
	artifacts := map[string]*Artifact{
		"plan": {
			Path: "test/plan.small.yml",
			Type: "plan",
			Data: map[string]interface{}{
				"small_version": ProtocolVersion,
				"owner":         "agent",
				"tasks": []interface{}{
					map[string]interface{}{
						"id":    "task-1",
						"title": "Known task",
					},
				},
			},
		},
		"handoff": {
			Path: "test/handoff.small.yml",
			Type: "handoff",
			Data: map[string]interface{}{
				"small_version": ProtocolVersion,
				"owner":         "agent",
				"summary":       "Test handoff",
				"resume": map[string]interface{}{
					"current_task_id": "task-missing",
					"next_steps":      []interface{}{},
				},
				"links": []interface{}{},
			},
		},
	}

	violations := CheckInvariants(artifacts, true)
	hasS3 := false
	for _, v := range violations {
		if contains(v.Message, "strict invariant S3 failed") {
			if contains(v.Message, "resume.current_task_id") && contains(v.Message, "task-missing") {
				hasS3 = true
			}
		}
	}
	if !hasS3 {
		t.Error("expected strict mode S3 violation for missing current task")
	}
}

func TestCheckInvariants_StrictModeSecrets(t *testing.T) {

	artifacts := map[string]*Artifact{
		"progress": {
			Path: "test/progress.small.yml",
			Type: "progress",
			Data: map[string]interface{}{
				"small_version": ProtocolVersion,
				"owner":         "agent",
				"entries":       []interface{}{},
				"api_key":       "secret123", // Should be caught in strict mode
			},
		},
	}

	// Non-strict mode should not catch secrets (but will catch unknown key)
	violationsNonStrict := CheckInvariants(artifacts, false)
	hasSecretErrorNonStrict := false
	for _, v := range violationsNonStrict {
		if contains(v.Message, "secret") {
			hasSecretErrorNonStrict = true
			break
		}
	}
	if hasSecretErrorNonStrict {
		t.Error("non-strict mode should not detect secrets")
	}

	// Strict mode should catch secrets
	violationsStrict := CheckInvariants(artifacts, true)
	hasSecretErrorStrict := false
	for _, v := range violationsStrict {
		if contains(v.Message, "secret") {
			hasSecretErrorStrict = true
			break
		}
	}
	if !hasSecretErrorStrict {
		t.Error("strict mode should detect secrets")
	}
}

func TestCheckInvariants_StrictModeInsecureLinks(t *testing.T) {
	artifacts := map[string]*Artifact{
		"handoff": {
			Path: "test/handoff.small.yml",
			Type: "handoff",
			Data: map[string]interface{}{
				"small_version": ProtocolVersion,
				"owner":         "agent",
				"summary":       "Test handoff",
				"resume": map[string]interface{}{
					"current_task_id": "",
					"next_steps":      []interface{}{},
				},
				"links": []interface{}{
					map[string]interface{}{
						"url": "http://insecure.example.com", // Should fail in strict mode
					},
				},
			},
		},
	}

	// Non-strict mode should not catch insecure links
	violationsNonStrict := CheckInvariants(artifacts, false)
	hasInsecureErrorNonStrict := false
	for _, v := range violationsNonStrict {
		if contains(v.Message, "insecure") {
			hasInsecureErrorNonStrict = true
			break
		}
	}
	if hasInsecureErrorNonStrict {
		t.Error("non-strict mode should not detect insecure links")
	}

	// Strict mode should catch insecure links
	violationsStrict := CheckInvariants(artifacts, true)
	hasInsecureErrorStrict := false
	for _, v := range violationsStrict {
		if contains(v.Message, "insecure") {
			hasInsecureErrorStrict = true
			break
		}
	}
	if !hasInsecureErrorStrict {
		t.Error("strict mode should detect insecure links")
	}
}

// Helper functions

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func makeValidArtifact(artifactType string) map[string]interface{} {
	base := map[string]interface{}{
		"small_version": ProtocolVersion,
	}

	switch artifactType {
	case "intent":
		base["owner"] = "human"
		base["intent"] = "Test intent"
		base["scope"] = map[string]interface{}{
			"include": []interface{}{},
			"exclude": []interface{}{},
		}
		base["success_criteria"] = []interface{}{}
	case "constraints":
		base["owner"] = "human"
		base["constraints"] = []interface{}{
			map[string]interface{}{
				"id":       "test-1",
				"rule":     "Test rule",
				"severity": "error",
			},
		}
	case "plan":
		base["owner"] = "agent"
		base["tasks"] = []interface{}{
			map[string]interface{}{
				"id":    "task-1",
				"title": "Test task",
			},
		}
	case "progress":
		base["owner"] = "agent"
		base["entries"] = []interface{}{}
	case "handoff":
		base["owner"] = "agent"
		base["summary"] = "Test summary"
		base["resume"] = map[string]interface{}{
			"current_task_id": "",
			"next_steps":      []interface{}{},
		}
		base["links"] = []interface{}{}
	}

	return base
}

func TestCheckDanglingTasks_NoDanglingTasks(t *testing.T) {
	plan := &Artifact{
		Path: "test/plan.small.yml",
		Type: "plan",
		Data: map[string]interface{}{
			"small_version": ProtocolVersion,
			"owner":         "agent",
			"tasks": []interface{}{
				map[string]interface{}{
					"id":     "task-1",
					"title":  "Completed task",
					"status": "completed",
				},
			},
		},
	}
	progress := &Artifact{
		Path: "test/progress.small.yml",
		Type: "progress",
		Data: map[string]interface{}{
			"small_version": ProtocolVersion,
			"owner":         "agent",
			"entries": []interface{}{
				map[string]interface{}{
					"task_id":   "task-1",
					"timestamp": "2025-01-01T00:00:00.000000000Z",
					"evidence":  "Done",
				},
			},
		},
	}

	dangling := CheckDanglingTasks(plan, progress)
	if len(dangling) != 0 {
		t.Errorf("expected 0 dangling tasks, got %d", len(dangling))
	}
}

func TestCheckDanglingTasks_TaskWithProgressButPending(t *testing.T) {
	plan := &Artifact{
		Path: "test/plan.small.yml",
		Type: "plan",
		Data: map[string]interface{}{
			"small_version": ProtocolVersion,
			"owner":         "agent",
			"tasks": []interface{}{
				map[string]interface{}{
					"id":     "task-1",
					"title":  "Started but not finished",
					"status": "pending",
				},
			},
		},
	}
	progress := &Artifact{
		Path: "test/progress.small.yml",
		Type: "progress",
		Data: map[string]interface{}{
			"small_version": ProtocolVersion,
			"owner":         "agent",
			"entries": []interface{}{
				map[string]interface{}{
					"task_id":   "task-1",
					"timestamp": "2025-01-01T00:00:00.000000000Z",
					"evidence":  "Started work",
				},
			},
		},
	}

	dangling := CheckDanglingTasks(plan, progress)
	if len(dangling) != 1 {
		t.Fatalf("expected 1 dangling task, got %d", len(dangling))
	}
	if dangling[0].ID != "task-1" {
		t.Errorf("expected task-1, got %s", dangling[0].ID)
	}
}

func TestCheckDanglingTasks_TaskWithProgressButInProgress(t *testing.T) {
	plan := &Artifact{
		Path: "test/plan.small.yml",
		Type: "plan",
		Data: map[string]interface{}{
			"small_version": ProtocolVersion,
			"owner":         "agent",
			"tasks": []interface{}{
				map[string]interface{}{
					"id":     "task-1",
					"title":  "Work in progress",
					"status": "in_progress",
				},
			},
		},
	}
	progress := &Artifact{
		Path: "test/progress.small.yml",
		Type: "progress",
		Data: map[string]interface{}{
			"small_version": ProtocolVersion,
			"owner":         "agent",
			"entries": []interface{}{
				map[string]interface{}{
					"task_id":   "task-1",
					"timestamp": "2025-01-01T00:00:00.000000000Z",
					"evidence":  "Working on it",
				},
			},
		},
	}

	dangling := CheckDanglingTasks(plan, progress)
	if len(dangling) != 1 {
		t.Fatalf("expected 1 dangling task, got %d", len(dangling))
	}
	if dangling[0].Status != "in_progress" {
		t.Errorf("expected in_progress status, got %s", dangling[0].Status)
	}
}

func TestCheckDanglingTasks_BlockedTaskIsNotDangling(t *testing.T) {
	plan := &Artifact{
		Path: "test/plan.small.yml",
		Type: "plan",
		Data: map[string]interface{}{
			"small_version": ProtocolVersion,
			"owner":         "agent",
			"tasks": []interface{}{
				map[string]interface{}{
					"id":     "task-1",
					"title":  "Blocked task",
					"status": "blocked",
				},
			},
		},
	}
	progress := &Artifact{
		Path: "test/progress.small.yml",
		Type: "progress",
		Data: map[string]interface{}{
			"small_version": ProtocolVersion,
			"owner":         "agent",
			"entries": []interface{}{
				map[string]interface{}{
					"task_id":   "task-1",
					"timestamp": "2025-01-01T00:00:00.000000000Z",
					"evidence":  "Blocked on dependency",
				},
			},
		},
	}

	dangling := CheckDanglingTasks(plan, progress)
	if len(dangling) != 0 {
		t.Errorf("expected 0 dangling tasks (blocked is terminal), got %d", len(dangling))
	}
}

func TestCheckDanglingTasks_TaskWithoutProgressIsNotDangling(t *testing.T) {
	plan := &Artifact{
		Path: "test/plan.small.yml",
		Type: "plan",
		Data: map[string]interface{}{
			"small_version": ProtocolVersion,
			"owner":         "agent",
			"tasks": []interface{}{
				map[string]interface{}{
					"id":     "task-1",
					"title":  "Not yet started",
					"status": "pending",
				},
			},
		},
	}
	progress := &Artifact{
		Path: "test/progress.small.yml",
		Type: "progress",
		Data: map[string]interface{}{
			"small_version": ProtocolVersion,
			"owner":         "agent",
			"entries":       []interface{}{},
		},
	}

	dangling := CheckDanglingTasks(plan, progress)
	if len(dangling) != 0 {
		t.Errorf("expected 0 dangling tasks (no progress means not started), got %d", len(dangling))
	}
}

func TestCheckDanglingTasks_MetaTasksIgnored(t *testing.T) {
	plan := &Artifact{
		Path: "test/plan.small.yml",
		Type: "plan",
		Data: map[string]interface{}{
			"small_version": ProtocolVersion,
			"owner":         "agent",
			"tasks": []interface{}{
				map[string]interface{}{
					"id":     "task-1",
					"title":  "Regular task",
					"status": "pending",
				},
			},
		},
	}
	progress := &Artifact{
		Path: "test/progress.small.yml",
		Type: "progress",
		Data: map[string]interface{}{
			"small_version": ProtocolVersion,
			"owner":         "agent",
			"entries": []interface{}{
				map[string]interface{}{
					"task_id":   "meta/reconcile-plan",
					"timestamp": "2025-01-01T00:00:00.000000000Z",
					"evidence":  "Reconciled plan",
				},
			},
		},
	}

	dangling := CheckDanglingTasks(plan, progress)
	if len(dangling) != 0 {
		t.Errorf("expected 0 dangling tasks (meta/ tasks should be ignored), got %d", len(dangling))
	}
}

func TestCheckInvariants_StrictModeLocalhostHTTPAllowedInProgress(t *testing.T) {
	tests := []struct {
		name      string
		url       string
		wantError bool
	}{
		{"localhost passes", "http://localhost:3001/api", false},
		{"localhost no port passes", "http://localhost/test", false},
		{"127.0.0.1 passes", "http://127.0.0.1:8080/health", false},
		{"0.0.0.0 passes", "http://0.0.0.0:3000", false},
		{"[::1] passes", "http://[::1]:3000/api", false},
		{"external http fails", "http://example.com", true},
		{"external http fails 2", "http://api.example.com/v1", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			artifacts := map[string]*Artifact{
				"progress": {
					Path: "test/progress.small.yml",
					Type: "progress",
					Data: map[string]interface{}{
						"small_version": ProtocolVersion,
						"owner":         "agent",
						"entries": []interface{}{
							map[string]interface{}{
								"task_id":   "task-1",
								"timestamp": "2025-01-01T00:00:00.000000000Z",
								"evidence":  "Started server at " + tt.url,
							},
						},
					},
				},
			}

			violations := CheckInvariants(artifacts, true)
			hasInsecureError := false
			for _, v := range violations {
				if contains(v.Message, "insecure link detected") {
					hasInsecureError = true
					break
				}
			}
			if hasInsecureError != tt.wantError {
				t.Errorf("insecure error = %v, want %v", hasInsecureError, tt.wantError)
			}
		})
	}
}

func TestCheckInvariants_StrictModeLocalhostHTTPBlockedInHandoff(t *testing.T) {
	// Even localhost http should be blocked in non-progress files
	artifacts := map[string]*Artifact{
		"handoff": {
			Path: "test/handoff.small.yml",
			Type: "handoff",
			Data: map[string]interface{}{
				"small_version": ProtocolVersion,
				"owner":         "agent",
				"summary":       "Test handoff with http://localhost:3001",
				"resume": map[string]interface{}{
					"current_task_id": "",
					"next_steps":      []interface{}{},
				},
				"links": []interface{}{},
			},
		},
	}

	violations := CheckInvariants(artifacts, true)
	hasInsecureError := false
	for _, v := range violations {
		if contains(v.Message, "insecure link detected") {
			hasInsecureError = true
			break
		}
	}
	if !hasInsecureError {
		t.Error("expected insecure link error for localhost in handoff (allowlist only applies to progress)")
	}
}

func TestCheckInvariants_StrictModeInsecureLinksErrorMessage(t *testing.T) {
	// Test that the error message includes the localhost allowlist note for progress files
	artifacts := map[string]*Artifact{
		"progress": {
			Path: "test/progress.small.yml",
			Type: "progress",
			Data: map[string]interface{}{
				"small_version": ProtocolVersion,
				"owner":         "agent",
				"entries": []interface{}{
					map[string]interface{}{
						"task_id":   "task-1",
						"timestamp": "2025-01-01T00:00:00.000000000Z",
						"evidence":  "Connected to http://example.com",
					},
				},
			},
		},
	}

	violations := CheckInvariants(artifacts, true)
	var insecureMsg string
	for _, v := range violations {
		if contains(v.Message, "insecure link detected") {
			insecureMsg = v.Message
			break
		}
	}
	if insecureMsg == "" {
		t.Fatal("expected insecure link violation")
	}
	if !contains(insecureMsg, "localhost") || !contains(insecureMsg, "127.0.0.1") {
		t.Errorf("expected error message to mention localhost allowlist, got: %s", insecureMsg)
	}
}

func TestCheckInvariants_StrictModeIgnoresPartialHTTPInText(t *testing.T) {
	// Test that "http://" appearing in descriptive text (not as a real URL) is ignored
	// This handles cases like: "error: insecure link http:// in file"
	artifacts := map[string]*Artifact{
		"progress": {
			Path: "test/progress.small.yml",
			Type: "progress",
			Data: map[string]interface{}{
				"small_version": ProtocolVersion,
				"owner":         "agent",
				"entries": []interface{}{
					map[string]interface{}{
						"task_id":   "task-1",
						"timestamp": "2025-01-01T00:00:00.000000000Z",
						"evidence":  "small check --strict fails with insecure link http:// in .small/progress.small.yml",
					},
					map[string]interface{}{
						"task_id":   "task-2",
						"timestamp": "2025-01-01T00:00:00.000000002Z",
						"evidence":  "Started server at http://localhost:3001 successfully",
					},
				},
			},
		},
	}

	violations := CheckInvariants(artifacts, true)
	for _, v := range violations {
		if contains(v.Message, "insecure link detected") {
			t.Errorf("should not flag partial 'http://' in descriptive text or localhost URLs, got: %s", v.Message)
		}
	}
}
