package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateReplayId(t *testing.T) {
	// Create temp directory for tests
	tmpDir, err := os.MkdirTemp("", "small-handoff-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	smallDir := filepath.Join(tmpDir, ".small")
	if err := os.MkdirAll(smallDir, 0755); err != nil {
		t.Fatalf("failed to create .small dir: %v", err)
	}

	// Create test artifacts
	intentContent := `small_version: "1.0.0"
owner: human
intent: "Test intent for deterministic hashing"
scope:
  include: []
  exclude: []
success_criteria: []
`
	planContent := `small_version: "1.0.0"
owner: agent
tasks:
  - id: task-1
    title: "Test task"
`
	constraintsContent := `small_version: "1.0.0"
owner: human
constraints:
  - id: no-secrets
    rule: "No secrets"
    severity: error
`

	// Write intent and plan (required)
	if err := os.WriteFile(filepath.Join(smallDir, "intent.small.yml"), []byte(intentContent), 0644); err != nil {
		t.Fatalf("failed to write intent: %v", err)
	}
	if err := os.WriteFile(filepath.Join(smallDir, "plan.small.yml"), []byte(planContent), 0644); err != nil {
		t.Fatalf("failed to write plan: %v", err)
	}

	t.Run("auto mode generates deterministic hash", func(t *testing.T) {
		// Generate replayId twice with same input
		result1, err := generateReplayId(smallDir, "")
		if err != nil {
			t.Fatalf("generateReplayId failed: %v", err)
		}
		result2, err := generateReplayId(smallDir, "")
		if err != nil {
			t.Fatalf("generateReplayId failed: %v", err)
		}

		// Should produce identical values
		if result1.Value != result2.Value {
			t.Errorf("determinism failed: got different values %s vs %s", result1.Value, result2.Value)
		}

		// Source should be "auto"
		if result1.Source != "auto" {
			t.Errorf("expected source 'auto', got %s", result1.Source)
		}

		// Value should be 64 lowercase hex chars
		if len(result1.Value) != 64 {
			t.Errorf("expected 64 char hash, got %d chars", len(result1.Value))
		}
		if !replayIdPattern.MatchString(result1.Value) {
			t.Errorf("hash should match lowercase hex pattern, got %s", result1.Value)
		}
	})

	t.Run("constraints affect hash when present", func(t *testing.T) {
		// Get hash without constraints
		hashWithout, err := generateReplayId(smallDir, "")
		if err != nil {
			t.Fatalf("generateReplayId failed: %v", err)
		}

		// Add constraints file
		constraintsPath := filepath.Join(smallDir, "constraints.small.yml")
		if err := os.WriteFile(constraintsPath, []byte(constraintsContent), 0644); err != nil {
			t.Fatalf("failed to write constraints: %v", err)
		}
		t.Cleanup(func() { _ = os.Remove(constraintsPath) })

		// Get hash with constraints
		hashWith, err := generateReplayId(smallDir, "")
		if err != nil {
			t.Fatalf("generateReplayId failed: %v", err)
		}

		// Hashes should be different
		if hashWithout.Value == hashWith.Value {
			t.Error("adding constraints should change the hash")
		}
	})

	t.Run("manual mode validates and uses provided value", func(t *testing.T) {
		validHash := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
		result, err := generateReplayId(smallDir, validHash)
		if err != nil {
			t.Fatalf("generateReplayId with valid manual hash failed: %v", err)
		}

		if result.Value != validHash {
			t.Errorf("expected value %s, got %s", validHash, result.Value)
		}
		if result.Source != "manual" {
			t.Errorf("expected source 'manual', got %s", result.Source)
		}
	})

	t.Run("manual mode accepts uppercase and normalizes to lowercase", func(t *testing.T) {
		uppercaseHash := "0123456789ABCDEF0123456789ABCDEF0123456789ABCDEF0123456789ABCDEF"
		expectedLower := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
		result, err := generateReplayId(smallDir, uppercaseHash)
		if err != nil {
			t.Fatalf("generateReplayId with uppercase hash failed: %v", err)
		}

		if result.Value != expectedLower {
			t.Errorf("expected normalized value %s, got %s", expectedLower, result.Value)
		}
		if result.Source != "manual" {
			t.Errorf("expected source 'manual', got %s", result.Source)
		}
	})

	t.Run("manual mode rejects invalid format", func(t *testing.T) {
		invalidHashes := []string{
			"invalid", // too short
			"a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b",  // 63 chars
			"g1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2", // invalid char 'g'
		}

		for _, invalidHash := range invalidHashes {
			_, err := generateReplayId(smallDir, invalidHash)
			if err == nil {
				t.Errorf("expected error for invalid hash %q, got nil", invalidHash)
			}
		}
	})

	t.Run("fails when intent missing", func(t *testing.T) {
		caseDir := filepath.Join(tmpDir, "empty")
		caseSmallDir := filepath.Join(caseDir, ".small")
		if err := os.MkdirAll(caseSmallDir, 0755); err != nil {
			t.Fatalf("failed to create case .small dir: %v", err)
		}

		_, err := generateReplayId(caseSmallDir, "")
		if err == nil {
			t.Error("expected error when intent.small.yml missing")
		}
	})

	t.Run("fails when plan missing", func(t *testing.T) {
		caseDir := filepath.Join(tmpDir, "no-plan")
		caseSmallDir := filepath.Join(caseDir, ".small")
		if err := os.MkdirAll(caseSmallDir, 0755); err != nil {
			t.Fatalf("failed to create case .small dir: %v", err)
		}
		if err := os.WriteFile(filepath.Join(caseSmallDir, "intent.small.yml"), []byte(intentContent), 0644); err != nil {
			t.Fatalf("failed to write intent: %v", err)
		}

		_, err := generateReplayId(caseSmallDir, "")
		if err == nil {
			t.Error("expected error when plan.small.yml missing")
		}
	})

	t.Run("line ending normalization produces consistent hash", func(t *testing.T) {
		// Create a directory with CRLF line endings
		crlfCaseDir := filepath.Join(tmpDir, "crlf")
		crlfSmallDir := filepath.Join(crlfCaseDir, ".small")
		if err := os.MkdirAll(crlfSmallDir, 0755); err != nil {
			t.Fatalf("failed to create crlf .small dir: %v", err)
		}

		// Write files with CRLF
		crlfIntent := "small_version: \"1.0.0\"\r\nowner: human\r\nintent: \"Test\"\r\nscope:\r\n  include: []\r\n  exclude: []\r\nsuccess_criteria: []\r\n"
		crlfPlan := "small_version: \"1.0.0\"\r\nowner: agent\r\ntasks:\r\n  - id: task-1\r\n    title: \"Test\"\r\n"

		if err := os.WriteFile(filepath.Join(crlfSmallDir, "intent.small.yml"), []byte(crlfIntent), 0644); err != nil {
			t.Fatalf("failed to write crlf intent: %v", err)
		}
		if err := os.WriteFile(filepath.Join(crlfSmallDir, "plan.small.yml"), []byte(crlfPlan), 0644); err != nil {
			t.Fatalf("failed to write crlf plan: %v", err)
		}

		// Create same content with LF
		lfCaseDir := filepath.Join(tmpDir, "lf")
		lfSmallDir := filepath.Join(lfCaseDir, ".small")
		if err := os.MkdirAll(lfSmallDir, 0755); err != nil {
			t.Fatalf("failed to create lf .small dir: %v", err)
		}

		lfIntent := "small_version: \"1.0.0\"\nowner: human\nintent: \"Test\"\nscope:\n  include: []\n  exclude: []\nsuccess_criteria: []\n"
		lfPlan := "small_version: \"1.0.0\"\nowner: agent\ntasks:\n  - id: task-1\n    title: \"Test\"\n"

		if err := os.WriteFile(filepath.Join(lfSmallDir, "intent.small.yml"), []byte(lfIntent), 0644); err != nil {
			t.Fatalf("failed to write lf intent: %v", err)
		}
		if err := os.WriteFile(filepath.Join(lfSmallDir, "plan.small.yml"), []byte(lfPlan), 0644); err != nil {
			t.Fatalf("failed to write lf plan: %v", err)
		}

		// Hashes should be identical
		crlfResult, err := generateReplayId(crlfSmallDir, "")
		if err != nil {
			t.Fatalf("generateReplayId for crlf failed: %v", err)
		}
		lfResult, err := generateReplayId(lfSmallDir, "")
		if err != nil {
			t.Fatalf("generateReplayId for lf failed: %v", err)
		}

		if crlfResult.Value != lfResult.Value {
			t.Errorf("line ending normalization failed: CRLF=%s, LF=%s", crlfResult.Value, lfResult.Value)
		}
	})
}

func TestHandoffBlocksDanglingTasks(t *testing.T) {
	// Create temp directory for tests
	tmpDir, err := os.MkdirTemp("", "small-handoff-dangling-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	smallDir := filepath.Join(tmpDir, ".small")
	if err := os.MkdirAll(smallDir, 0755); err != nil {
		t.Fatalf("failed to create .small dir: %v", err)
	}

	// Create test artifacts with a dangling task (has progress but status is pending)
	intentContent := `small_version: "1.0.0"
owner: human
intent: "Test intent"
scope:
  include: []
  exclude: []
success_criteria: []
`
	planContent := `small_version: "1.0.0"
owner: agent
tasks:
  - id: task-1
    title: "Started but not finished"
    status: pending
`
	constraintsContent := `small_version: "1.0.0"
owner: human
constraints:
  - id: no-secrets
    rule: "No secrets"
    severity: error
`
	progressContent := `small_version: "1.0.0"
owner: agent
entries:
  - task_id: task-1
    timestamp: "2025-01-01T00:00:00.000000000Z"
    evidence: "Started working on this task"
`

	if err := os.WriteFile(filepath.Join(smallDir, "intent.small.yml"), []byte(intentContent), 0644); err != nil {
		t.Fatalf("failed to write intent: %v", err)
	}
	if err := os.WriteFile(filepath.Join(smallDir, "plan.small.yml"), []byte(planContent), 0644); err != nil {
		t.Fatalf("failed to write plan: %v", err)
	}
	if err := os.WriteFile(filepath.Join(smallDir, "constraints.small.yml"), []byte(constraintsContent), 0644); err != nil {
		t.Fatalf("failed to write constraints: %v", err)
	}
	if err := os.WriteFile(filepath.Join(smallDir, "progress.small.yml"), []byte(progressContent), 0644); err != nil {
		t.Fatalf("failed to write progress: %v", err)
	}

	// Create handoff command and execute it
	cmd := handoffCmd()
	cmd.SetArgs([]string{"--dir", tmpDir, "--workspace", "any"})

	err = cmd.Execute()
	if err == nil {
		t.Error("expected error for dangling tasks, got nil")
	}
	if err != nil && !contains(err.Error(), "dangling tasks") {
		t.Errorf("expected error about dangling tasks, got: %v", err)
	}

	// Verify handoff.small.yml was NOT created
	handoffPath := filepath.Join(smallDir, "handoff.small.yml")
	if _, err := os.Stat(handoffPath); !os.IsNotExist(err) {
		t.Error("handoff.small.yml should not be created when dangling tasks exist")
	}
}

func TestHandoffAllowsCompletedTasks(t *testing.T) {
	// Create temp directory for tests
	tmpDir, err := os.MkdirTemp("", "small-handoff-completed-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	smallDir := filepath.Join(tmpDir, ".small")
	if err := os.MkdirAll(smallDir, 0755); err != nil {
		t.Fatalf("failed to create .small dir: %v", err)
	}

	// Create test artifacts with a completed task
	intentContent := `small_version: "1.0.0"
owner: human
intent: "Test intent"
scope:
  include: []
  exclude: []
success_criteria: []
`
	planContent := `small_version: "1.0.0"
owner: agent
tasks:
  - id: task-1
    title: "Properly completed task"
    status: completed
`
	constraintsContent := `small_version: "1.0.0"
owner: human
constraints:
  - id: no-secrets
    rule: "No secrets"
    severity: error
`
	progressContent := `small_version: "1.0.0"
owner: agent
entries:
  - task_id: task-1
    timestamp: "2025-01-01T00:00:00.000000000Z"
    evidence: "Completed the work"
`

	if err := os.WriteFile(filepath.Join(smallDir, "intent.small.yml"), []byte(intentContent), 0644); err != nil {
		t.Fatalf("failed to write intent: %v", err)
	}
	if err := os.WriteFile(filepath.Join(smallDir, "plan.small.yml"), []byte(planContent), 0644); err != nil {
		t.Fatalf("failed to write plan: %v", err)
	}
	if err := os.WriteFile(filepath.Join(smallDir, "constraints.small.yml"), []byte(constraintsContent), 0644); err != nil {
		t.Fatalf("failed to write constraints: %v", err)
	}
	if err := os.WriteFile(filepath.Join(smallDir, "progress.small.yml"), []byte(progressContent), 0644); err != nil {
		t.Fatalf("failed to write progress: %v", err)
	}

	// Create handoff command and execute it
	cmd := handoffCmd()
	cmd.SetArgs([]string{"--dir", tmpDir, "--workspace", "any"})

	err = cmd.Execute()
	if err != nil {
		t.Errorf("expected no error for completed tasks, got: %v", err)
	}

	// Verify handoff.small.yml was created
	handoffPath := filepath.Join(smallDir, "handoff.small.yml")
	if _, err := os.Stat(handoffPath); os.IsNotExist(err) {
		t.Error("handoff.small.yml should be created when all tasks are properly completed")
	}
	handoffContent, err := os.ReadFile(handoffPath)
	if err != nil {
		t.Fatalf("failed to read handoff: %v", err)
	}
	if !strings.Contains(string(handoffContent), `small_version: "1.0.0"`) {
		t.Fatalf("expected quoted small_version in handoff output")
	}
	if strings.Contains(string(handoffContent), "current_task_id") {
		t.Fatalf("expected completed handoff to omit current_task_id, got:\n%s", string(handoffContent))
	}
	if !strings.Contains(string(handoffContent), "summary: Run complete. strict check passed. All plan tasks completed.") {
		t.Fatalf("expected canonical completed summary, got:\n%s", string(handoffContent))
	}
	if !strings.Contains(string(handoffContent), "next_steps: []") {
		t.Fatalf("expected completed handoff next_steps to be empty, got:\n%s", string(handoffContent))
	}
}

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

func TestBuildHandoffCompleteState(t *testing.T) {
	tmpDir := t.TempDir()
	smallDir := filepath.Join(tmpDir, ".small")
	if err := os.MkdirAll(smallDir, 0o755); err != nil {
		t.Fatalf("failed to create .small dir: %v", err)
	}

	artifacts := map[string]string{
		"intent.small.yml": `small_version: "1.0.0"
owner: "human"
intent: "Test"
scope:
  include: []
  exclude: []
success_criteria: []
`,
		"constraints.small.yml": `small_version: "1.0.0"
owner: "human"
constraints:
  - id: "no-secrets"
    rule: "No secrets"
    severity: "error"
`,
		"plan.small.yml": `small_version: "1.0.0"
owner: "agent"
tasks:
  - id: "task-1"
    title: "done"
    status: "completed"
`,
		"progress.small.yml": `small_version: "1.0.0"
owner: "agent"
entries:
  - task_id: "task-1"
    status: "completed"
    timestamp: "2026-01-01T00:00:00.000000000Z"
    evidence: "done"
`,
	}
	for name, content := range artifacts {
		if err := os.WriteFile(filepath.Join(smallDir, name), []byte(content), 0o644); err != nil {
			t.Fatalf("failed to write %s: %v", name, err)
		}
	}

	h, err := buildHandoff(tmpDir, "", "", nil, nil, nil, defaultNextStepsLimit)
	if err != nil {
		t.Fatalf("buildHandoff failed: %v", err)
	}
	if h.Summary != "Run complete. strict check passed. All plan tasks completed." {
		t.Fatalf("unexpected summary: %q", h.Summary)
	}
	if h.Resume.CurrentTaskID != nil {
		t.Fatalf("expected nil current_task_id, got %v", *h.Resume.CurrentTaskID)
	}
	if len(h.Resume.NextSteps) != 0 {
		t.Fatalf("expected no next steps, got %d", len(h.Resume.NextSteps))
	}
}

func TestBuildHandoffBlockedOnStrictFailure(t *testing.T) {
	tmpDir := t.TempDir()
	smallDir := filepath.Join(tmpDir, ".small")
	if err := os.MkdirAll(smallDir, 0o755); err != nil {
		t.Fatalf("failed to create .small dir: %v", err)
	}

	artifacts := map[string]string{
		"intent.small.yml": `small_version: "1.0.0"
owner: "human"
intent: "Test"
scope:
  include: []
  exclude: []
success_criteria: []
`,
		"constraints.small.yml": `small_version: "1.0.0"
owner: "human"
constraints:
  - id: "no-secrets"
    rule: "No secrets"
    severity: "error"
`,
		"plan.small.yml": `small_version: "1.0.0"
owner: "agent"
tasks:
  - id: "task-1"
    title: "todo"
    status: "pending"
`,
		"progress.small.yml": `small_version: "1.0.0"
owner: "agent"
entries: []
`,
	}
	for name, content := range artifacts {
		if err := os.WriteFile(filepath.Join(smallDir, name), []byte(content), 0o644); err != nil {
			t.Fatalf("failed to write %s: %v", name, err)
		}
	}
	if err := os.MkdirAll(filepath.Join(smallDir, "logs"), 0o755); err != nil {
		t.Fatalf("failed to create legacy logs dir: %v", err)
	}

	h, err := buildHandoff(tmpDir, "", "", nil, nil, nil, defaultNextStepsLimit)
	if err != nil {
		t.Fatalf("buildHandoff failed: %v", err)
	}
	if h.Summary != "Run blocked on task-1." {
		t.Fatalf("unexpected summary: %q", h.Summary)
	}
	if h.Resume.CurrentTaskID == nil || *h.Resume.CurrentTaskID != "task-1" {
		t.Fatalf("expected current_task_id task-1, got %v", h.Resume.CurrentTaskID)
	}
	if len(h.Resume.NextSteps) == 0 || !strings.Contains(strings.Join(h.Resume.NextSteps, " "), "small check --strict") {
		t.Fatalf("expected unblock steps to mention strict check, got %v", h.Resume.NextSteps)
	}
}

func TestBuildHandoffBlockedOnStrictFailureWithoutTasks(t *testing.T) {
	tmpDir := t.TempDir()
	smallDir := filepath.Join(tmpDir, ".small")
	if err := os.MkdirAll(smallDir, 0o755); err != nil {
		t.Fatalf("failed to create .small dir: %v", err)
	}

	artifacts := map[string]string{
		"intent.small.yml": `small_version: "1.0.0"
owner: "human"
intent: "Test"
scope:
  include: []
  exclude: []
success_criteria: []
`,
		"constraints.small.yml": `small_version: "1.0.0"
owner: "human"
constraints:
  - id: "no-secrets"
    rule: "No secrets"
    severity: "error"
`,
		"plan.small.yml": `small_version: "1.0.0"
owner: "agent"
tasks: []
`,
		"progress.small.yml": `small_version: "1.0.0"
owner: "agent"
entries: []
`,
	}
	for name, content := range artifacts {
		if err := os.WriteFile(filepath.Join(smallDir, name), []byte(content), 0o644); err != nil {
			t.Fatalf("failed to write %s: %v", name, err)
		}
	}
	if err := os.MkdirAll(filepath.Join(smallDir, "logs"), 0o755); err != nil {
		t.Fatalf("failed to create legacy logs dir: %v", err)
	}

	h, err := buildHandoff(tmpDir, "", "", nil, nil, nil, defaultNextStepsLimit)
	if err != nil {
		t.Fatalf("buildHandoff failed: %v", err)
	}
	if h.Summary != "Run blocked. strict check failed." {
		t.Fatalf("unexpected summary: %q", h.Summary)
	}
	if h.Resume.CurrentTaskID == nil || *h.Resume.CurrentTaskID != "meta/blocker" {
		t.Fatalf("expected current_task_id meta/blocker, got %v", h.Resume.CurrentTaskID)
	}
}

func TestBuildHandoffInProgressState(t *testing.T) {
	tmpDir := t.TempDir()
	smallDir := filepath.Join(tmpDir, ".small")
	if err := os.MkdirAll(smallDir, 0o755); err != nil {
		t.Fatalf("failed to create .small dir: %v", err)
	}

	artifacts := map[string]string{
		"intent.small.yml": `small_version: "1.0.0"
owner: "human"
intent: "Test"
scope:
  include: []
  exclude: []
success_criteria: []
`,
		"constraints.small.yml": `small_version: "1.0.0"
owner: "human"
constraints:
  - id: "no-secrets"
    rule: "No secrets"
    severity: "error"
`,
		"plan.small.yml": `small_version: "1.0.0"
owner: "agent"
tasks:
  - id: "task-9"
    title: "todo"
    status: "pending"
`,
		"progress.small.yml": `small_version: "1.0.0"
owner: "agent"
entries: []
`,
	}
	for name, content := range artifacts {
		if err := os.WriteFile(filepath.Join(smallDir, name), []byte(content), 0o644); err != nil {
			t.Fatalf("failed to write %s: %v", name, err)
		}
	}

	h, err := buildHandoff(tmpDir, "", "", nil, nil, nil, defaultNextStepsLimit)
	if err != nil {
		t.Fatalf("buildHandoff failed: %v", err)
	}
	if h.Summary != "Run in progress. Next task: task-9." {
		t.Fatalf("unexpected summary: %q", h.Summary)
	}
	if h.Resume.CurrentTaskID == nil || *h.Resume.CurrentTaskID != "task-9" {
		t.Fatalf("expected current_task_id task-9, got %v", h.Resume.CurrentTaskID)
	}
	if len(h.Resume.NextSteps) != 1 || h.Resume.NextSteps[0] != "Continue with task-9" {
		t.Fatalf("unexpected next steps: %v", h.Resume.NextSteps)
	}
}
