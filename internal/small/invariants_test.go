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
						"task_id": "task-1",
						// No evidence field - should fail
					},
				},
			},
		},
	}

	violations := CheckInvariants(artifacts, false)
	hasEvidenceError := false
	for _, v := range violations {
		if contains(v.Message, "evidence") {
			hasEvidenceError = true
			break
		}
	}
	if !hasEvidenceError {
		t.Error("expected error for missing evidence, got none")
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

	violations := CheckInvariants(artifacts, false)
	hasMissingProgress := false
	for _, v := range violations {
		if contains(v.Message, "progress entries missing or invalid for completed plan tasks") {
			hasMissingProgress = true
			break
		}
	}
	if !hasMissingProgress {
		t.Error("expected violation for completed task without progress evidence")
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
						"timestamp": "2025-01-01T00:00:00Z",
						"evidence":  "Linked completed tasks to progress entries",
					},
				},
			},
		},
	}

	violations := CheckInvariants(artifacts, false)
	for _, v := range violations {
		if contains(v.Message, "progress entries missing or invalid for completed plan tasks") {
			t.Fatalf("unexpected missing progress violation: %s", v.Message)
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

	violations := CheckInvariants(artifacts, false)
	hasMissingProgress := false
	for _, v := range violations {
		if contains(v.Message, "progress entries missing or invalid for completed plan tasks") {
			hasMissingProgress = true
			break
		}
	}
	if !hasMissingProgress {
		t.Error("expected violation for completed task with empty note")
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

	violations := CheckInvariants(artifacts, false)
	hasMissingProgress := false
	for _, v := range violations {
		if contains(v.Message, "progress entries missing or invalid for completed plan tasks") {
			hasMissingProgress = true
			break
		}
	}
	if !hasMissingProgress {
		t.Error("expected violation for completed task with invalid timestamp")
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
