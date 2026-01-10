package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
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
	testPlan := "plan: test plan content\nowner: agent\n"
	testProgress := "progress: test progress content\nowner: agent\n"
	testConstraints := "constraints: test constraints content\nowner: human\n"
	testHandoff := "handoff: test handoff content\nowner: agent\n"

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

	t.Run("resets ephemeral files and preserves audit files", func(t *testing.T) {
		cmd := resetCmd()
		cmd.SetArgs([]string{"--yes"})
		if err := cmd.Execute(); err != nil {
			t.Fatalf("reset failed: %v", err)
		}

		// Check progress was preserved
		progressContent, err := os.ReadFile(filepath.Join(smallDir, "progress.small.yml"))
		if err != nil {
			t.Fatalf("failed to read progress: %v", err)
		}
		if string(progressContent) != testProgress {
			t.Error("progress.small.yml was modified but should have been preserved")
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

		// Check handoff was reset
		handoffContent, err := os.ReadFile(filepath.Join(smallDir, "handoff.small.yml"))
		if err != nil {
			t.Fatalf("failed to read handoff: %v", err)
		}
		if string(handoffContent) == testHandoff {
			t.Error("handoff.small.yml was not reset")
		}
	})

	// Restore files for next test
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(smallDir, name), []byte(content), 0644); err != nil {
			t.Fatalf("failed to write %s: %v", name, err)
		}
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
	})
}
