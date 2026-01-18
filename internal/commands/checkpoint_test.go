package commands

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/justyn-clark/small-protocol/internal/small"
	"github.com/justyn-clark/small-protocol/internal/workspace"
	"gopkg.in/yaml.v3"
)

func TestCheckpointAtomicity(t *testing.T) {
	oldNow := progressTimestampNow
	defer func() { progressTimestampNow = oldNow }()

	fixed := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	progressTimestampNow = func() time.Time { return fixed }

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
		Entries: []map[string]interface{}{
			{
				"task_id":   "seed",
				"status":    "completed",
				"timestamp": formatProgressTimestamp(fixed),
				"evidence":  "seed",
			},
		},
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

	if err := runCheckpointApply(tmpDir, "task-1", "completed", ""); err == nil {
		t.Fatal("expected checkpoint to fail with missing evidence")
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
