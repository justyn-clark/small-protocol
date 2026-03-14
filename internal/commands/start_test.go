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

func TestStartCommandCreatesCompleteHandoff(t *testing.T) {
	tmpDir := t.TempDir()

	cmd := startCmd()
	cmd.SetArgs([]string{"--dir", tmpDir, "--workspace", "any"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("start failed: %v", err)
	}

	handoffPath := filepath.Join(tmpDir, small.SmallDir, "handoff.small.yml")
	handoffData, err := os.ReadFile(handoffPath)
	if err != nil {
		t.Fatalf("failed to read handoff: %v", err)
	}
	if !strings.Contains(string(handoffData), `small_version: "1.0.0"`) {
		t.Fatalf("expected quoted small_version in handoff output")
	}

	var handoff map[string]any
	if err := yaml.Unmarshal(handoffData, &handoff); err != nil {
		t.Fatalf("failed to parse handoff: %v", err)
	}

	replayId, ok := handoff["replayId"].(map[string]any)
	if !ok || stringVal(replayId["value"]) == "" || stringVal(replayId["source"]) == "" {
		t.Fatalf("expected replayId.value and replayId.source to be populated")
	}

	resume, ok := handoff["resume"].(map[string]any)
	if !ok {
		t.Fatalf("expected resume object in handoff")
	}
	if _, ok := resume["current_task_id"].(string); !ok {
		t.Fatalf("expected resume.current_task_id to be present")
	}
	nextSteps, ok := resume["next_steps"].([]any)
	if !ok || len(nextSteps) == 0 {
		t.Fatalf("expected resume.next_steps to be populated")
	}

	runInfo, ok := handoff["run"].(map[string]any)
	if !ok {
		t.Fatalf("expected run metadata to be present")
	}
	if stringVal(runInfo["transition_reason"]) != "self_heal" {
		t.Fatalf("expected transition_reason self_heal, got %q", stringVal(runInfo["transition_reason"]))
	}
	if stringVal(runInfo["created_at"]) == "" {
		t.Fatalf("expected run.created_at to be populated")
	}

	progressPath := filepath.Join(tmpDir, small.SmallDir, "progress.small.yml")
	progressData, err := os.ReadFile(progressPath)
	if err != nil {
		t.Fatalf("failed to read progress: %v", err)
	}
	var progress ProgressData
	if err := yaml.Unmarshal(progressData, &progress); err != nil {
		t.Fatalf("failed to parse progress: %v", err)
	}
	if len(progress.Entries) == 0 {
		t.Fatalf("expected progress entry for self-heal")
	}
	if progress.Entries[len(progress.Entries)-1]["task_id"] != "meta/replayid-self-heal" {
		t.Fatalf("expected replayId self-heal progress entry")
	}
}

func TestStartFixRepairsLegacyRuntimeLayout(t *testing.T) {
	tmpDir := t.TempDir()
	if err := runSelftestInit(tmpDir); err != nil {
		t.Fatalf("runSelftestInit failed: %v", err)
	}
	if err := workspace.Save(tmpDir, workspace.KindRepoRoot); err != nil {
		t.Fatalf("workspace save failed: %v", err)
	}

	legacyArchive := filepath.Join(tmpDir, small.SmallDir, "archive", "run-1")
	legacyRuns := filepath.Join(tmpDir, small.SmallDir, "runs")
	if err := os.MkdirAll(legacyArchive, 0o755); err != nil {
		t.Fatalf("failed to create legacy archive: %v", err)
	}
	if err := os.MkdirAll(legacyRuns, 0o755); err != nil {
		t.Fatalf("failed to create legacy runs: %v", err)
	}
	if err := os.WriteFile(filepath.Join(legacyArchive, "archive.small.yml"), []byte("archive"), 0o644); err != nil {
		t.Fatalf("failed to seed archive: %v", err)
	}
	if err := os.WriteFile(filepath.Join(legacyRuns, small.RunIndexFileName), []byte("index"), 0o644); err != nil {
		t.Fatalf("failed to seed run index: %v", err)
	}

	cmd := startCmd()
	cmd.SetArgs([]string{"--dir", tmpDir, "--workspace", "any", "--fix"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("start --fix failed: %v", err)
	}

	if _, err := os.Stat(filepath.Join(tmpDir, small.ArchiveStoreDirName, "run-1", "archive.small.yml")); err != nil {
		t.Fatalf("expected migrated archive: %v", err)
	}
	if _, err := os.Stat(filepath.Join(tmpDir, small.RunStoreDirName, small.RunIndexFileName)); err != nil {
		t.Fatalf("expected migrated run index: %v", err)
	}
	progressData, err := os.ReadFile(filepath.Join(tmpDir, small.SmallDir, "progress.small.yml"))
	if err != nil {
		t.Fatalf("failed to read progress: %v", err)
	}
	if !strings.Contains(string(progressData), "meta/reconcile-runtime-layout") {
		t.Fatalf("expected runtime layout reconcile progress entry, got %q", string(progressData))
	}
}
