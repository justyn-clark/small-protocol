package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/justyn-clark/small-protocol/internal/small"
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

	var handoff map[string]interface{}
	if err := yaml.Unmarshal(handoffData, &handoff); err != nil {
		t.Fatalf("failed to parse handoff: %v", err)
	}

	replayId, ok := handoff["replayId"].(map[string]interface{})
	if !ok || stringVal(replayId["value"]) == "" || stringVal(replayId["source"]) == "" {
		t.Fatalf("expected replayId.value and replayId.source to be populated")
	}

	resume, ok := handoff["resume"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected resume object in handoff")
	}
	if _, ok := resume["current_task_id"].(string); !ok {
		t.Fatalf("expected resume.current_task_id to be present")
	}
	nextSteps, ok := resume["next_steps"].([]interface{})
	if !ok || len(nextSteps) == 0 {
		t.Fatalf("expected resume.next_steps to be populated")
	}

	runInfo, ok := handoff["run"].(map[string]interface{})
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
