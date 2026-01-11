package commands

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/justyn-clark/small-protocol/internal/small"
)

func TestVerifyCommand(t *testing.T) {
	// Create temp directory for tests
	tmpDir, err := os.MkdirTemp("", "small-verify-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	t.Run("returns ExitSystemError when .small/ does not exist", func(t *testing.T) {
		code := runVerify(tmpDir, false, true)
		if code != ExitSystemError {
			t.Errorf("expected exit code %d, got %d", ExitSystemError, code)
		}
	})

	// Create .small/ directory
	smallDir := filepath.Join(tmpDir, ".small")
	if err := os.MkdirAll(smallDir, 0755); err != nil {
		t.Fatalf("failed to create .small dir: %v", err)
	}

	t.Run("returns ExitInvalid when required files are missing", func(t *testing.T) {
		code := runVerify(tmpDir, false, true)
		if code != ExitInvalid {
			t.Errorf("expected exit code %d, got %d", ExitInvalid, code)
		}
	})

	// Create valid artifacts
	validArtifacts := map[string]string{
		"intent.small.yml": `small_version: "1.0.0"
owner: "human"
intent: "Test intent"
scope:
  include: []
  exclude: []
success_criteria: []
`,
		"constraints.small.yml": `small_version: "1.0.0"
owner: "human"
constraints:
  - id: "test"
    rule: "Test rule"
    severity: "error"
`,
		"plan.small.yml": `small_version: "1.0.0"
owner: "agent"
tasks:
  - id: "task-1"
    title: "Test task"
`,
		"progress.small.yml": `small_version: "1.0.0"
owner: "agent"
entries: []
`,
		"handoff.small.yml": `small_version: "1.0.0"
owner: "agent"
summary: "Test handoff"
resume:
  current_task_id: ""
  next_steps: []
links: []
`,
	}

	for filename, content := range validArtifacts {
		path := filepath.Join(smallDir, filename)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("failed to write %s: %v", filename, err)
		}
	}

	t.Run("returns ExitValid for valid artifacts", func(t *testing.T) {
		code := runVerify(tmpDir, false, true)
		if code != ExitValid {
			t.Errorf("expected exit code %d, got %d", ExitValid, code)
		}
	})

	t.Run("returns ExitValid with --strict for valid artifacts", func(t *testing.T) {
		code := runVerify(tmpDir, true, true)
		if code != ExitValid {
			t.Errorf("expected exit code %d, got %d", ExitValid, code)
		}
	})

	// Create invalid intent (wrong owner)
	t.Run("returns ExitInvalid for invalid artifacts", func(t *testing.T) {
		invalidIntent := `small_version: "1.0.0"
owner: "agent"
intent: "Test"
scope:
  include: []
  exclude: []
success_criteria: []
`
		path := filepath.Join(smallDir, "intent.small.yml")
		if err := os.WriteFile(path, []byte(invalidIntent), 0644); err != nil {
			t.Fatalf("failed to write intent: %v", err)
		}

		code := runVerify(tmpDir, false, true)
		if code != ExitInvalid {
			t.Errorf("expected exit code %d, got %d", ExitInvalid, code)
		}

		// Restore valid intent
		if err := os.WriteFile(path, []byte(validArtifacts["intent.small.yml"]), 0644); err != nil {
			t.Fatalf("failed to restore intent: %v", err)
		}
	})
}

func TestValidateReplayId(t *testing.T) {
	tests := []struct {
		name       string
		replayId   map[string]interface{}
		wantErrors int
	}{
		{
			name:       "nil replayId - no errors",
			replayId:   nil,
			wantErrors: 0,
		},
		{
			name: "valid replayId with auto source",
			replayId: map[string]interface{}{
				"value":  "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2",
				"source": "auto",
			},
			wantErrors: 0,
		},
		{
			name: "valid replayId with manual source",
			replayId: map[string]interface{}{
				"value":  "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2",
				"source": "manual",
			},
			wantErrors: 0,
		},
		{
			name: "invalid SHA256 format",
			replayId: map[string]interface{}{
				"value":  "invalid",
				"source": "auto",
			},
			wantErrors: 1,
		},
		{
			name: "invalid source",
			replayId: map[string]interface{}{
				"value":  "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2",
				"source": "cli",
			},
			wantErrors: 1,
		},
		{
			name: "missing value",
			replayId: map[string]interface{}{
				"source": "auto",
			},
			wantErrors: 1,
		},
		{
			name: "missing source",
			replayId: map[string]interface{}{
				"value": "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2",
			},
			wantErrors: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock artifact with replayId
			artifact := &small.Artifact{
				Type: "handoff",
				Path: "test/handoff.small.yml",
				Data: map[string]interface{}{
					"small_version": "1.0.0",
					"owner":         "agent",
				},
			}

			if tt.replayId != nil {
				artifact.Data["replayId"] = tt.replayId
			}

			errors := validateReplayId(artifact)
			if len(errors) != tt.wantErrors {
				t.Errorf("expected %d errors, got %d: %v", tt.wantErrors, len(errors), errors)
			}
		})
	}
}
