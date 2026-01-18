package commands

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/justyn-clark/small-protocol/internal/workspace"
)

func TestEmitIncludesProgressSummary(t *testing.T) {
	tmpDir := t.TempDir()
	artifacts := cloneArtifacts(defaultArtifacts())

	first := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	second := first.Add(2 * time.Second)
	artifacts["progress.small.yml"] = "small_version: \"1.0.0\"\nowner: \"agent\"\nentries:\n  - task_id: \"task-1\"\n    status: \"completed\"\n    timestamp: \"" + formatProgressTimestamp(first) + "\"\n    evidence: \"first\"\n  - task_id: \"task-2\"\n    status: \"completed\"\n    timestamp: \"" + formatProgressTimestamp(second) + "\"\n    evidence: \"second\"\n"

	writeArtifacts(t, tmpDir, artifacts)
	mustSaveWorkspace(t, tmpDir, workspace.KindRepoRoot)

	include, err := parseEmitInclude("")
	if err != nil {
		t.Fatalf("parseEmitInclude error: %v", err)
	}

	output, exitCode, err := buildEmitOutput(tmpDir, tmpDir, include, workspace.ScopeRoot, 1, 3, false)
	if err != nil {
		t.Fatalf("buildEmitOutput error: %v", err)
	}
	if exitCode != ExitValid {
		t.Fatalf("expected ExitValid, got %d", exitCode)
	}
	if output.Progress.LastTimestamp != formatProgressTimestamp(second) {
		t.Fatalf("expected last timestamp %q, got %q", formatProgressTimestamp(second), output.Progress.LastTimestamp)
	}
	if len(output.Progress.Recent) != 1 {
		t.Fatalf("expected 1 recent entry, got %d", len(output.Progress.Recent))
	}
	if output.Progress.Recent[0].Timestamp != formatProgressTimestamp(second) {
		t.Fatalf("expected recent timestamp %q, got %q", formatProgressTimestamp(second), output.Progress.Recent[0].Timestamp)
	}
}

func TestEmitCheckDoesNotMutateArtifacts(t *testing.T) {
	tmpDir := t.TempDir()
	artifacts := cloneArtifacts(defaultArtifacts())
	writeArtifacts(t, tmpDir, artifacts)
	mustSaveWorkspace(t, tmpDir, workspace.KindRepoRoot)

	progressPath := filepath.Join(tmpDir, ".small", "progress.small.yml")
	before, err := os.ReadFile(progressPath)
	if err != nil {
		t.Fatalf("failed to read progress before: %v", err)
	}

	include, err := parseEmitInclude("")
	if err != nil {
		t.Fatalf("parseEmitInclude error: %v", err)
	}

	output, exitCode, err := buildEmitOutput(tmpDir, tmpDir, include, workspace.ScopeRoot, 5, 3, true)
	if err != nil {
		t.Fatalf("buildEmitOutput error: %v", err)
	}
	if exitCode != ExitValid {
		t.Fatalf("expected ExitValid, got %d", exitCode)
	}
	if output.Enforcement == nil {
		t.Fatal("expected enforcement results")
	}

	after, err := os.ReadFile(progressPath)
	if err != nil {
		t.Fatalf("failed to read progress after: %v", err)
	}
	if string(before) != string(after) {
		t.Fatal("expected progress.small.yml unchanged after emit --check")
	}
}
