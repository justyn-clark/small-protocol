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

func TestComputeHandoffResumeDeterministic(t *testing.T) {
	plan := &PlanData{
		Tasks: []PlanTask{
			{ID: "task-0", Title: "Active task", Status: "in_progress"},
			{ID: "task-1", Title: "Completed task", Status: "completed"},
			{ID: "task-2", Title: "Blocked by task-1", Status: "pending", Dependencies: []string{"task-1"}},
			{ID: "task-3", Title: "Blocked by task-2", Status: "pending", Dependencies: []string{"task-2"}},
			{ID: "task-4", Title: "Independent pending", Status: "pending"},
		},
	}

	resume := computeHandoffResume(plan, 3)
	if resume.CurrentTaskID != "task-0" {
		t.Fatalf("expected current_task_id task-0, got %q", resume.CurrentTaskID)
	}
	if len(resume.NextSteps) != 3 {
		t.Fatalf("expected 3 next steps, got %d", len(resume.NextSteps))
	}
	if resume.NextSteps[0] != "Active task" {
		t.Fatalf("expected first next step to be current task, got %q", resume.NextSteps[0])
	}
	if resume.NextSteps[1] != "Blocked by task-1" {
		t.Fatalf("expected second next step to be task-2, got %q", resume.NextSteps[1])
	}
	if resume.NextSteps[2] != "Independent pending" {
		t.Fatalf("expected third next step to be task-4, got %q", resume.NextSteps[2])
	}
}

func TestComputeHandoffResumeDefaultsWhenNoActionableTasks(t *testing.T) {
	plan := &PlanData{
		Tasks: []PlanTask{
			{ID: "task-1", Title: "Done", Status: "completed"},
		},
	}

	resume := computeHandoffResume(plan, 3)
	if len(resume.NextSteps) != 1 {
		t.Fatalf("expected default next steps when none are actionable")
	}
	if resume.NextSteps[0] != defaultNextStepMessage {
		t.Fatalf("expected default guidance next step, got %q", resume.NextSteps[0])
	}
}
