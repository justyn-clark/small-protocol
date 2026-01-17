package commands

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/justyn-clark/small-protocol/internal/small"
	"github.com/justyn-clark/small-protocol/internal/workspace"
	"gopkg.in/yaml.v3"
)

func TestCheckpointAtomicity(t *testing.T) {
	tmpDir := t.TempDir()
	smallDir := filepath.Join(tmpDir, ".small")
	if err := os.MkdirAll(smallDir, 0o755); err != nil {
		t.Fatalf("failed to create .small dir: %v", err)
	}
	mustSaveWorkspace(t, tmpDir, workspace.KindRepoRoot)

	plan := PlanData{
		SmallVersion: small.ProtocolVersion,
		Owner:        "agent",
		Tasks: []PlanTask{{
			ID:     "task-1",
			Title:  "Test task",
			Status: "pending",
		}},
	}
	progress := ProgressData{
		SmallVersion: small.ProtocolVersion,
		Owner:        "agent",
		Entries:      []map[string]interface{}{},
	}
	planData, err := yaml.Marshal(&plan)
	if err != nil {
		t.Fatalf("failed to marshal plan: %v", err)
	}
	progressData, err := yaml.Marshal(&progress)
	if err != nil {
		t.Fatalf("failed to marshal progress: %v", err)
	}
	planPath := filepath.Join(smallDir, "plan.small.yml")
	progressPath := filepath.Join(smallDir, "progress.small.yml")
	if err := os.WriteFile(planPath, planData, 0o644); err != nil {
		t.Fatalf("failed to write plan: %v", err)
	}
	if err := os.WriteFile(progressPath, progressData, 0o644); err != nil {
		t.Fatalf("failed to write progress: %v", err)
	}

	originalPlanData, err := os.ReadFile(planPath)
	if err != nil {
		t.Fatalf("failed to read plan: %v", err)
	}
	originalProgressData, err := os.ReadFile(progressPath)
	if err != nil {
		t.Fatalf("failed to read progress: %v", err)
	}

	invalidEntry := map[string]interface{}{
		"task_id": "",
		"status":  "completed",
	}
	if err := validateProgressEntry(invalidEntry); err == nil {
		t.Fatal("expected invalid progress entry")
	}

	if err := runCheckpointApply(tmpDir, "task-1", "completed", ""); err == nil {
		t.Fatal("expected checkpoint to fail with invalid progress entry")
	}

	finalPlan, err := os.ReadFile(planPath)
	if err != nil {
		t.Fatalf("failed to read plan after failure: %v", err)
	}
	finalProgress, err := os.ReadFile(progressPath)
	if err != nil {
		t.Fatalf("failed to read progress after failure: %v", err)
	}

	if string(finalPlan) != string(originalPlanData) {
		t.Fatal("plan.small.yml should be unchanged after failed checkpoint")
	}
	if string(finalProgress) != string(originalProgressData) {
		t.Fatal("progress.small.yml should be unchanged after failed checkpoint")
	}
}
