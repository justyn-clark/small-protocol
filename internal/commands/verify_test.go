package commands

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/justyn-clark/small-protocol/internal/small"
	"github.com/justyn-clark/small-protocol/internal/workspace"
)

func TestVerifyCommand(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "small-verify-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	t.Run("returns ExitSystemError when .small/ does not exist", func(t *testing.T) {
		code := runVerify(tmpDir, false, true, workspace.ScopeRoot)
		if code != ExitSystemError {
			t.Errorf("expected exit code %d, got %d", ExitSystemError, code)
		}
	})

	smallDir := filepath.Join(tmpDir, ".small")
	if err := os.MkdirAll(smallDir, 0755); err != nil {
		t.Fatalf("failed to create .small dir: %v", err)
	}
	if err := workspace.Save(tmpDir, workspace.KindRepoRoot); err != nil {
		t.Fatalf("failed to write workspace metadata: %v", err)
	}

	t.Run("returns ExitInvalid when required files are missing", func(t *testing.T) {
		code := runVerify(tmpDir, false, true, workspace.ScopeRoot)
		if code != ExitInvalid {
			t.Errorf("expected exit code %d, got %d", ExitInvalid, code)
		}
	})

	validArtifacts := defaultArtifacts()
	writeArtifacts(t, tmpDir, validArtifacts)

	t.Run("returns ExitValid for valid artifacts", func(t *testing.T) {
		code := runVerify(tmpDir, false, true, workspace.ScopeRoot)
		if code != ExitValid {
			t.Errorf("expected exit code %d, got %d", ExitValid, code)
		}
	})

	t.Run("returns ExitValid with --strict for valid artifacts", func(t *testing.T) {
		code := runVerify(tmpDir, true, true, workspace.ScopeRoot)
		if code != ExitValid {
			t.Errorf("expected exit code %d, got %d", ExitValid, code)
		}
	})

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

		code := runVerify(tmpDir, false, true, workspace.ScopeRoot)
		if code != ExitInvalid {
			t.Errorf("expected exit code %d, got %d", ExitInvalid, code)
		}

		if err := os.WriteFile(path, []byte(validArtifacts["intent.small.yml"]), 0644); err != nil {
			t.Fatalf("failed to restore intent: %v", err)
		}
	})
}

func TestVerifyExampleWorkspaceScope(t *testing.T) {
	exampleDir, err := os.MkdirTemp("", "small-verify-example")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(exampleDir)

	artifacts := cloneArtifacts(defaultArtifacts())
	writeArtifacts(t, exampleDir, artifacts)
	mustSaveWorkspace(t, exampleDir, workspace.KindExamples)

	code := runVerify(exampleDir, false, true, workspace.ScopeRoot)
	if code != ExitInvalid {
		t.Errorf("expected exit code %d for example workspace under root scope, got %d", ExitInvalid, code)
	}

	code = runVerify(exampleDir, false, true, workspace.ScopeExamples)
	if code != ExitValid {
		t.Errorf("expected exit code %d for example workspace with examples scope, got %d", ExitValid, code)
	}
}

func TestVerifyProgressDuplicateTimestamps(t *testing.T) {
	dupDir, err := os.MkdirTemp("", "small-verify-progress-dup")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(dupDir)

	artifacts := cloneArtifacts(defaultArtifacts())
	artifacts["progress.small.yml"] = `small_version: "1.0.0"
owner: "agent"
entries:
  - task_id: "task-1"
    status: "completed"
    timestamp: "2025-01-01T00:00:00.000000000Z"
    evidence: "first entry"
  - task_id: "task-2"
    status: "completed"
    timestamp: "2025-01-01T00:00:00.000000000Z"
    evidence: "duplicate timestamp"
`
	writeArtifacts(t, dupDir, artifacts)
	mustSaveWorkspace(t, dupDir, workspace.KindRepoRoot)

	code := runVerify(dupDir, false, true, workspace.ScopeRoot)
	if code != ExitInvalid {
		t.Errorf("expected exit code %d for duplicate timestamps, got %d", ExitInvalid, code)
	}
}

func TestVerifyProgressTimestampRequiresFractional(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "small-verify-fractional")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	artifacts := cloneArtifacts(defaultArtifacts())
	artifacts["progress.small.yml"] = `small_version: "1.0.0"
owner: "agent"
entries:
  - task_id: "task-1"
    status: "completed"
    timestamp: "2025-01-01T00:00:00Z"
    evidence: "missing fractional"
`
	writeArtifacts(t, tmpDir, artifacts)
	mustSaveWorkspace(t, tmpDir, workspace.KindRepoRoot)

	code := runVerify(tmpDir, false, true, workspace.ScopeRoot)
	if code != ExitInvalid {
		t.Errorf("expected exit code %d for missing fractional seconds, got %d", ExitInvalid, code)
	}
}

func TestVerifyProgressIncreasingTimestamps(t *testing.T) {
	nanoDir, err := os.MkdirTemp("", "small-verify-nano")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(nanoDir)

	artifacts := cloneArtifacts(defaultArtifacts())
	artifacts["progress.small.yml"] = `small_version: "1.0.0"
owner: "agent"
entries:
  - task_id: "task-1"
    status: "completed"
    timestamp: "2025-01-01T00:00:00.000000001Z"
    evidence: "first entry"
  - task_id: "task-2"
    status: "completed"
    timestamp: "2025-01-01T00:00:00.000000002Z"
    evidence: "second entry"
`
	writeArtifacts(t, nanoDir, artifacts)
	mustSaveWorkspace(t, nanoDir, workspace.KindRepoRoot)

	code := runVerify(nanoDir, false, true, workspace.ScopeRoot)
	if code != ExitValid {
		t.Errorf("expected exit code %d for RFC3339Nano timestamps, got %d", ExitValid, code)
	}
}

func TestVerifyCompletedTaskWithoutProgressEvidence(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "small-verify-evidence-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	artifacts := map[string]string{
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
  - id: "task-completed-no-evidence"
    title: "A completed task with no progress entry"
    status: "completed"
`,
		"progress.small.yml": `small_version: "1.0.0"
owner: "agent"
entries: []
`,
		"handoff.small.yml": `small_version: "1.0.0"
owner: "agent"
summary: "Test handoff"
resume:
  current_task_id: null
  next_steps: []
links: []
replayId:
  value: "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2"
  source: "auto"
`,
	}

	writeArtifacts(t, tmpDir, artifacts)
	mustSaveWorkspace(t, tmpDir, workspace.KindRepoRoot)

	code := runVerify(tmpDir, true, true, workspace.ScopeRoot)
	if code != ExitInvalid {
		t.Errorf("expected exit code %d (ExitInvalid) for completed task without progress evidence, got %d", ExitInvalid, code)
	}
}

func TestVerifyHandoffWithoutReplayId(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "small-verify-replayid-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	artifacts := map[string]string{
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
summary: "Test handoff WITHOUT replayId"
resume:
  current_task_id: null
  next_steps: []
links: []
`,
	}

	writeArtifacts(t, tmpDir, artifacts)
	mustSaveWorkspace(t, tmpDir, workspace.KindRepoRoot)

	code := runVerify(tmpDir, true, true, workspace.ScopeRoot)
	if code != ExitInvalid {
		t.Errorf("expected exit code %d (ExitInvalid) for completed task without progress evidence, got %d", ExitInvalid, code)
	}

}

func TestValidateReplayId(t *testing.T) {
	tests := []struct {
		name       string
		replayId   map[string]any
		wantErrors int
	}{
		{
			name:       "nil replayId - error (replayId is required)",
			replayId:   nil,
			wantErrors: 1,
		},
		{
			name: "valid replayId with auto source",
			replayId: map[string]any{
				"value":  "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2",
				"source": "auto",
			},
			wantErrors: 0,
		},
		{
			name: "valid replayId with manual source",
			replayId: map[string]any{
				"value":  "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2",
				"source": "manual",
			},
			wantErrors: 0,
		},
		{
			name: "invalid SHA256 format",
			replayId: map[string]any{
				"value":  "invalid",
				"source": "auto",
			},
			wantErrors: 1,
		},
		{
			name: "invalid source",
			replayId: map[string]any{
				"value":  "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2",
				"source": "cli",
			},
			wantErrors: 1,
		},
		{
			name: "missing value",
			replayId: map[string]any{
				"source": "auto",
			},
			wantErrors: 1,
		},
		{
			name: "missing source",
			replayId: map[string]any{
				"value": "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2",
			},
			wantErrors: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			artifact := &small.Artifact{
				Type: "handoff",
				Path: "test/handoff.small.yml",
				Data: map[string]any{
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

func TestVerifyInvalidWorkspaceKindIncludesValidKinds(t *testing.T) {
	tmpDir := t.TempDir()
	writeArtifacts(t, tmpDir, defaultArtifacts())
	smallDir := filepath.Join(tmpDir, ".small")
	metadata := fmt.Sprintf("small_version: %q\nkind: invalid-kind\n", small.ProtocolVersion)
	if err := os.WriteFile(filepath.Join(smallDir, "workspace.small.yml"), []byte(metadata), 0644); err != nil {
		t.Fatalf("failed to write workspace metadata: %v", err)
	}

	oldStderr := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to capture stderr: %v", err)
	}
	os.Stderr = w
	defer func() {
		os.Stderr = oldStderr
	}()

	var buf bytes.Buffer
	done := make(chan struct{})
	go func() {
		_, _ = io.Copy(&buf, r)
		r.Close()
		close(done)
	}()

	code := runVerify(tmpDir, false, true, workspace.ScopeAny)
	if err := w.Close(); err != nil {
		t.Fatalf("failed to close stderr pipe: %v", err)
	}
	<-done

	if code != ExitInvalid {
		t.Fatalf("expected exit code %d, got %d", ExitInvalid, code)
	}

	output := buf.String()
	if !strings.Contains(output, "valid kinds") {
		t.Fatalf("expected output to mention valid kinds, got %q", output)
	}
	for _, kind := range []string{string(workspace.KindRepoRoot), string(workspace.KindExamples)} {
		if !strings.Contains(output, kind) {
			t.Fatalf("expected output to mention %q, got %q", kind, output)
		}
	}
}

func defaultArtifacts() map[string]string {
	return map[string]string{
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
  current_task_id: null
  next_steps: []
links: []
replayId:
  value: "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2"
  source: "auto"
`,
	}
}

func cloneArtifacts(src map[string]string) map[string]string {
	dst := make(map[string]string, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func writeArtifacts(t *testing.T, baseDir string, artifacts map[string]string) {
	smallDir := filepath.Join(baseDir, ".small")
	if err := os.MkdirAll(smallDir, 0755); err != nil {
		t.Fatalf("failed to create .small dir: %v", err)
	}
	for filename, content := range artifacts {
		if err := os.WriteFile(filepath.Join(smallDir, filename), []byte(content), 0644); err != nil {
			t.Fatalf("failed to write %s: %v", filename, err)
		}
	}
}

func mustSaveWorkspace(t *testing.T, baseDir string, kind workspace.Kind) {
	if err := workspace.Save(baseDir, kind); err != nil {
		t.Fatalf("failed to write workspace metadata: %v", err)
	}
}

func TestVerifyActionableFixes(t *testing.T) {
	t.Run("missing replayId suggests small handoff", func(t *testing.T) {
		tmpDir := t.TempDir()
		artifacts := cloneArtifacts(defaultArtifacts())
		artifacts["handoff.small.yml"] = `small_version: "1.0.0"
owner: "agent"
summary: "Test handoff WITHOUT replayId"
resume:
  current_task_id: null
  next_steps: []
links: []
`
		writeArtifacts(t, tmpDir, artifacts)
		mustSaveWorkspace(t, tmpDir, workspace.KindRepoRoot)

		// Capture stderr
		oldStderr := os.Stderr
		r, w, err := os.Pipe()
		if err != nil {
			t.Fatalf("failed to capture stderr: %v", err)
		}
		os.Stderr = w
		defer func() {
			os.Stderr = oldStderr
		}()

		var buf bytes.Buffer
		done := make(chan struct{})
		go func() {
			_, _ = io.Copy(&buf, r)
			r.Close()
			close(done)
		}()

		code := runVerify(tmpDir, false, false, workspace.ScopeRoot)
		w.Close()
		<-done

		if code != ExitInvalid {
			t.Errorf("expected exit code %d, got %d", ExitInvalid, code)
		}

		output := buf.String()
		if !strings.Contains(output, "small handoff") {
			t.Errorf("expected fix message to mention 'small handoff', got: %s", output)
		}
	})

	t.Run("progress timestamp issues suggest small progress migrate", func(t *testing.T) {
		tmpDir := t.TempDir()
		artifacts := cloneArtifacts(defaultArtifacts())
		artifacts["progress.small.yml"] = `small_version: "1.0.0"
owner: "agent"
entries:
  - task_id: "task-1"
    status: "completed"
    timestamp: "2025-01-01T00:00:00Z"
    evidence: "missing fractional"
`
		writeArtifacts(t, tmpDir, artifacts)
		mustSaveWorkspace(t, tmpDir, workspace.KindRepoRoot)

		oldStderr := os.Stderr
		r, w, err := os.Pipe()
		if err != nil {
			t.Fatalf("failed to capture stderr: %v", err)
		}
		os.Stderr = w
		defer func() {
			os.Stderr = oldStderr
		}()

		var buf bytes.Buffer
		done := make(chan struct{})
		go func() {
			_, _ = io.Copy(&buf, r)
			r.Close()
			close(done)
		}()

		code := runVerify(tmpDir, false, false, workspace.ScopeRoot)
		w.Close()
		<-done

		if code != ExitInvalid {
			t.Errorf("expected exit code %d, got %d", ExitInvalid, code)
		}

		output := buf.String()
		if !strings.Contains(output, "small progress migrate") {
			t.Errorf("expected fix message to mention 'small progress migrate', got: %s", output)
		}
	})

	t.Run("missing .small directory suggests small init", func(t *testing.T) {
		tmpDir := t.TempDir()

		oldStderr := os.Stderr
		r, w, err := os.Pipe()
		if err != nil {
			t.Fatalf("failed to capture stderr: %v", err)
		}
		os.Stderr = w
		defer func() {
			os.Stderr = oldStderr
		}()

		var buf bytes.Buffer
		done := make(chan struct{})
		go func() {
			_, _ = io.Copy(&buf, r)
			r.Close()
			close(done)
		}()

		code := runVerify(tmpDir, false, false, workspace.ScopeRoot)
		w.Close()
		<-done

		if code != ExitSystemError {
			t.Errorf("expected exit code %d, got %d", ExitSystemError, code)
		}

		output := buf.String()
		if !strings.Contains(output, "small init") {
			t.Errorf("expected fix message to mention 'small init', got: %s", output)
		}
	})
}

func TestVerifyRejectsEmptyHandoffCurrentTaskID(t *testing.T) {
	tmpDir := t.TempDir()
	artifacts := cloneArtifacts(defaultArtifacts())
	artifacts["handoff.small.yml"] = `small_version: "1.0.0"
owner: "agent"
summary: "Test handoff"
resume:
  current_task_id: ""
  next_steps: []
links: []
replayId:
  value: "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2"
  source: "auto"
`
	writeArtifacts(t, tmpDir, artifacts)
	mustSaveWorkspace(t, tmpDir, workspace.KindRepoRoot)

	code := runVerify(tmpDir, false, true, workspace.ScopeRoot)
	if code != ExitInvalid {
		t.Fatalf("expected ExitInvalid for empty current_task_id, got %d", code)
	}
}

func TestVerifyAcceptsHandoffWithoutCurrentTaskID(t *testing.T) {
	tmpDir := t.TempDir()
	artifacts := cloneArtifacts(defaultArtifacts())
	artifacts["handoff.small.yml"] = `small_version: "1.0.0"
owner: "agent"
summary: "Run complete. strict check passed. All plan tasks completed."
resume:
  next_steps: []
links: []
replayId:
  value: "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2"
  source: "auto"
`
	writeArtifacts(t, tmpDir, artifacts)
	mustSaveWorkspace(t, tmpDir, workspace.KindRepoRoot)

	code := runVerify(tmpDir, true, true, workspace.ScopeRoot)
	if code != ExitValid {
		t.Fatalf("expected ExitValid when current_task_id is omitted, got %d", code)
	}
}
