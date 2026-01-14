package commands

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/justyn-clark/small-protocol/internal/workspace"
)

func TestSelftestCommand(t *testing.T) {
	t.Run("selftest succeeds in temp directory", func(t *testing.T) {
		// Run selftest with default settings (OS temp, auto cleanup)
		err := runSelftest("", false, workspace.ScopeAny)
		if err != nil {
			t.Errorf("selftest failed: %v", err)
		}
	})

	t.Run("selftest succeeds with --keep flag", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "small-selftest-keep-test")
		if err != nil {
			t.Fatalf("failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		err = runSelftest(tmpDir, true, workspace.ScopeAny)
		if err != nil {
			t.Errorf("selftest with --keep failed: %v", err)
		}

		// Verify workspace was created and preserved
		smallDir := filepath.Join(tmpDir, ".small")
		if _, err := os.Stat(smallDir); os.IsNotExist(err) {
			t.Error("expected .small directory to be preserved with --keep")
		}

		// Verify key artifacts exist
		requiredFiles := []string{
			"intent.small.yml",
			"constraints.small.yml",
			"plan.small.yml",
			"progress.small.yml",
			"handoff.small.yml",
			"workspace.small.yml",
		}
		for _, f := range requiredFiles {
			path := filepath.Join(smallDir, f)
			if _, err := os.Stat(path); os.IsNotExist(err) {
				t.Errorf("expected %s to exist after selftest", f)
			}
		}
	})

	t.Run("selftest with explicit directory", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "small-selftest-explicit")
		if err != nil {
			t.Fatalf("failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		testDir := filepath.Join(tmpDir, "subdir")
		err = runSelftest(testDir, true, workspace.ScopeAny)
		if err != nil {
			t.Errorf("selftest with explicit dir failed: %v", err)
		}

		// Verify .small was created in the explicit directory
		smallDir := filepath.Join(testDir, ".small")
		if _, err := os.Stat(smallDir); os.IsNotExist(err) {
			t.Error("expected .small directory in explicit directory")
		}
	})
}

func TestSelftestStepFunctions(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "small-selftest-steps")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	t.Run("init step creates valid workspace", func(t *testing.T) {
		err := runSelftestInit(tmpDir)
		if err != nil {
			t.Fatalf("runSelftestInit failed: %v", err)
		}

		// Verify all files exist
		smallDir := filepath.Join(tmpDir, ".small")
		files := []string{
			"intent.small.yml",
			"constraints.small.yml",
			"plan.small.yml",
			"progress.small.yml",
			"handoff.small.yml",
			"workspace.small.yml",
		}
		for _, f := range files {
			if _, err := os.Stat(filepath.Join(smallDir, f)); os.IsNotExist(err) {
				t.Errorf("expected %s to exist after init", f)
			}
		}
	})

	t.Run("plan add step adds task", func(t *testing.T) {
		err := runSelftestPlanAdd(tmpDir)
		if err != nil {
			t.Fatalf("runSelftestPlanAdd failed: %v", err)
		}

		planPath := filepath.Join(tmpDir, ".small", "plan.small.yml")
		plan, err := loadPlan(planPath)
		if err != nil {
			t.Fatalf("failed to load plan: %v", err)
		}

		if len(plan.Tasks) < 2 {
			t.Error("expected at least 2 tasks after plan add")
		}
	})

	t.Run("plan done step marks task complete", func(t *testing.T) {
		err := runSelftestPlanDone(tmpDir)
		if err != nil {
			t.Fatalf("runSelftestPlanDone failed: %v", err)
		}

		planPath := filepath.Join(tmpDir, ".small", "plan.small.yml")
		plan, err := loadPlan(planPath)
		if err != nil {
			t.Fatalf("failed to load plan: %v", err)
		}

		// Find the task-2 (or highest) and check it's completed
		foundCompleted := false
		for _, task := range plan.Tasks {
			if task.Status == "completed" {
				foundCompleted = true
				break
			}
		}
		if !foundCompleted {
			t.Error("expected at least one completed task after plan done")
		}
	})

	t.Run("apply dry-run step records progress", func(t *testing.T) {
		err := runSelftestApplyDryRun(tmpDir)
		if err != nil {
			t.Fatalf("runSelftestApplyDryRun failed: %v", err)
		}
	})

	t.Run("handoff step generates valid handoff", func(t *testing.T) {
		err := runSelftestHandoff(tmpDir)
		if err != nil {
			t.Fatalf("runSelftestHandoff failed: %v", err)
		}

		handoffPath := filepath.Join(tmpDir, ".small", "handoff.small.yml")
		if _, err := os.Stat(handoffPath); os.IsNotExist(err) {
			t.Error("expected handoff.small.yml to exist after handoff step")
		}
	})

	t.Run("verify step passes", func(t *testing.T) {
		err := runSelftestVerify(tmpDir, workspace.ScopeAny)
		if err != nil {
			t.Errorf("runSelftestVerify failed: %v", err)
		}
	})
}

func TestFormatYAMLStringArray(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected string
	}{
		{
			name:     "empty array",
			input:    []string{},
			expected: "[]",
		},
		{
			name:     "single item",
			input:    []string{"item"},
			expected: `["item"]`,
		},
		{
			name:     "multiple items",
			input:    []string{"item1", "item2"},
			expected: `["item1", "item2"]`,
		},
		{
			name:     "item with quotes",
			input:    []string{`item "with" quotes`},
			expected: `["item \"with\" quotes"]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatYAMLStringArray(tt.input)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}
