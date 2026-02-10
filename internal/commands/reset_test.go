package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/justyn-clark/small-protocol/internal/small"
	"github.com/justyn-clark/small-protocol/internal/workspace"
	"gopkg.in/yaml.v3"
)

func TestResetCommand(t *testing.T) {
	// Create temp directory for test
	tmpDir, err := os.MkdirTemp("", "small-reset-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Save and restore baseDir
	oldBaseDir := baseDir
	baseDir = tmpDir
	defer func() { baseDir = oldBaseDir }()

	smallDir := filepath.Join(tmpDir, ".small")

	t.Run("fails when .small/ does not exist", func(t *testing.T) {
		cmd := resetCmd()
		cmd.SetArgs([]string{"--yes"})
		err := cmd.Execute()
		if err == nil {
			t.Error("expected error when .small/ does not exist")
		}
		if !strings.Contains(err.Error(), "does not exist") {
			t.Errorf("unexpected error message: %v", err)
		}
	})

	// Create .small/ directory with files
	if err := os.MkdirAll(smallDir, 0755); err != nil {
		t.Fatalf("failed to create .small dir: %v", err)
	}

	// Write test files
	testIntent := "intent: test intent content\nowner: human\n"
	testPlan := `small_version: "1.0.0"
owner: "agent"
tasks:
  - id: "task-1"
    title: "Test task"
`
	testProgress := `small_version: "1.0.0"
owner: "agent"
entries:
  - task_id: "task-0"
    status: "completed"
    timestamp: "2026-01-01T00:00:00.000000000Z"
    evidence: "Initial progress"
`
	testConstraints := "constraints: test constraints content\nowner: human\n"
	testHandoff := `small_version: "1.0.0"
owner: "agent"
summary: "Old handoff"
resume:
  current_task_id: null
  next_steps: []
links: []
replayId:
  value: "1111111111111111111111111111111111111111111111111111111111111111"
  source: "auto"
`

	files := map[string]string{
		"intent.small.yml":      testIntent,
		"plan.small.yml":        testPlan,
		"progress.small.yml":    testProgress,
		"constraints.small.yml": testConstraints,
		"handoff.small.yml":     testHandoff,
	}

	for name, content := range files {
		if err := os.WriteFile(filepath.Join(smallDir, name), []byte(content), 0644); err != nil {
			t.Fatalf("failed to write %s: %v", name, err)
		}
	}

	if err := workspace.Save(tmpDir, workspace.KindRepoRoot); err != nil {
		t.Fatalf("failed to write workspace metadata: %v", err)
	}

	t.Run("resets ephemeral files and preserves audit files", func(t *testing.T) {
		cmd := resetCmd()
		cmd.SetArgs([]string{"--yes"})
		if err := cmd.Execute(); err != nil {
			t.Fatalf("reset failed: %v", err)
		}

		// Check progress was preserved and appended
		progressContent, err := os.ReadFile(filepath.Join(smallDir, "progress.small.yml"))
		if err != nil {
			t.Fatalf("failed to read progress: %v", err)
		}
		var progress ProgressData
		if err := yaml.Unmarshal(progressContent, &progress); err != nil {
			t.Fatalf("failed to parse progress: %v", err)
		}
		if len(progress.Entries) != 2 {
			t.Fatalf("expected 2 progress entries, got %d", len(progress.Entries))
		}
		if progress.Entries[0]["task_id"] != "task-0" {
			t.Fatalf("expected original progress entry, got %v", progress.Entries[0]["task_id"])
		}
		if progress.Entries[1]["task_id"] != "reset" {
			t.Fatalf("expected reset progress entry, got %v", progress.Entries[1]["task_id"])
		}
		resetTimestamp, _ := progress.Entries[1]["timestamp"].(string)
		if _, err := small.ParseProgressTimestamp(resetTimestamp); err != nil {
			t.Fatalf("invalid reset timestamp %q: %v", resetTimestamp, err)
		}

		// Check constraints was preserved
		constraintsContent, err := os.ReadFile(filepath.Join(smallDir, "constraints.small.yml"))
		if err != nil {
			t.Fatalf("failed to read constraints: %v", err)
		}
		if string(constraintsContent) != testConstraints {
			t.Error("constraints.small.yml was modified but should have been preserved")
		}

		// Check plan was reset (should be different from test content)
		planContent, err := os.ReadFile(filepath.Join(smallDir, "plan.small.yml"))
		if err != nil {
			t.Fatalf("failed to read plan: %v", err)
		}
		if string(planContent) == testPlan {
			t.Error("plan.small.yml was not reset")
		}

		// Check intent was reset (should be different from test content)
		intentContent, err := os.ReadFile(filepath.Join(smallDir, "intent.small.yml"))
		if err != nil {
			t.Fatalf("failed to read intent: %v", err)
		}
		if string(intentContent) == testIntent {
			t.Error("intent.small.yml was not reset")
		}

		// Check handoff was reset and includes required fields
		handoffContent, err := os.ReadFile(filepath.Join(smallDir, "handoff.small.yml"))
		if err != nil {
			t.Fatalf("failed to read handoff: %v", err)
		}
		if string(handoffContent) == testHandoff {
			t.Error("handoff.small.yml was not reset")
		}
		if !strings.Contains(string(handoffContent), `small_version: "1.0.0"`) {
			t.Fatalf("expected quoted small_version in reset handoff output")
		}
		var handoff map[string]any
		if err := yaml.Unmarshal(handoffContent, &handoff); err != nil {
			t.Fatalf("failed to parse handoff: %v", err)
		}
		replayId, ok := handoff["replayId"].(map[string]any)
		if !ok {
			t.Fatalf("expected replayId in handoff after reset")
		}
		if stringVal(replayId["value"]) == "" {
			t.Fatalf("expected replayId.value to be populated after reset")
		}
		runInfo, ok := handoff["run"].(map[string]any)
		if !ok {
			t.Fatalf("expected run metadata in handoff after reset")
		}
		if stringVal(runInfo["transition_reason"]) != "reset" {
			t.Fatalf("expected transition_reason reset, got %q", stringVal(runInfo["transition_reason"]))
		}
		if stringVal(runInfo["previous_replay_id"]) != "1111111111111111111111111111111111111111111111111111111111111111" {
			t.Fatalf("expected previous_replay_id to match prior handoff replayId")
		}

	})

	// Restore files for next test
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(smallDir, name), []byte(content), 0644); err != nil {
			t.Fatalf("failed to write %s: %v", name, err)
		}
	}

	if err := workspace.Save(tmpDir, workspace.KindRepoRoot); err != nil {
		t.Fatalf("failed to write workspace metadata: %v", err)
	}

	t.Run("--keep-intent preserves intent file", func(t *testing.T) {
		cmd := resetCmd()
		cmd.SetArgs([]string{"--yes", "--keep-intent"})
		if err := cmd.Execute(); err != nil {
			t.Fatalf("reset --keep-intent failed: %v", err)
		}

		// Check intent was preserved
		intentContent, err := os.ReadFile(filepath.Join(smallDir, "intent.small.yml"))
		if err != nil {
			t.Fatalf("failed to read intent: %v", err)
		}
		if string(intentContent) != testIntent {
			t.Error("intent.small.yml was modified but should have been preserved with --keep-intent")
		}

		// Check plan was still reset
		planContent, err := os.ReadFile(filepath.Join(smallDir, "plan.small.yml"))
		if err != nil {
			t.Fatalf("failed to read plan: %v", err)
		}
		if string(planContent) == testPlan {
			t.Error("plan.small.yml was not reset")
		}

		progressContent, err := os.ReadFile(filepath.Join(smallDir, "progress.small.yml"))
		if err != nil {
			t.Fatalf("failed to read progress: %v", err)
		}
		var progress ProgressData
		if err := yaml.Unmarshal(progressContent, &progress); err != nil {
			t.Fatalf("failed to parse progress: %v", err)
		}
		if len(progress.Entries) != 2 {
			t.Fatalf("expected 2 progress entries, got %d", len(progress.Entries))
		}
		if progress.Entries[1]["task_id"] != "reset" {
			t.Fatalf("expected reset progress entry, got %v", progress.Entries[1]["task_id"])
		}
		resetTimestamp, _ := progress.Entries[1]["timestamp"].(string)
		if _, err := small.ParseProgressTimestamp(resetTimestamp); err != nil {
			t.Fatalf("invalid reset timestamp %q: %v", resetTimestamp, err)
		}
	})
}
