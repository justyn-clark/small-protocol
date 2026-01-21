package commands

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/justyn-clark/small-protocol/internal/workspace"
)

func TestCheckExitCodes(t *testing.T) {
	t.Run("validate failure returns ExitInvalid", func(t *testing.T) {
		tmpDir := t.TempDir()
		artifacts := cloneArtifacts(defaultArtifacts())
		artifacts["intent.small.yml"] = `small_version: "1.0.0"
owner: "agent"
intent: "Test"
scope:
  include: []
  exclude: []
success_criteria: []
`
		writeArtifacts(t, tmpDir, artifacts)
		mustSaveWorkspace(t, tmpDir, workspace.KindRepoRoot)

		code, _, err := runCheck(tmpDir, false, true, false, workspace.ScopeRoot, false)
		if err != nil {
			t.Fatalf("runCheck error: %v", err)
		}
		if code != ExitInvalid {
			t.Fatalf("expected ExitInvalid, got %d", code)
		}
	})

	t.Run("lint failure returns ExitInvalid", func(t *testing.T) {
		tmpDir := t.TempDir()
		artifacts := cloneArtifacts(defaultArtifacts())
		artifacts["progress.small.yml"] = `small_version: "1.0.0"
owner: "agent"
entries:
  - task_id: "task-1"
    status: "completed"
    timestamp: "2025-01-01T00:00:00.000000000Z"
    evidence: "first entry"
  - task_id: "task-2"
    status: "completed"
    timestamp: "2025-01-01T00:00:00.000000000Z"
    evidence: "duplicate timestamp"
`
		writeArtifacts(t, tmpDir, artifacts)
		mustSaveWorkspace(t, tmpDir, workspace.KindRepoRoot)

		code, _, err := runCheck(tmpDir, false, true, false, workspace.ScopeRoot, false)
		if err != nil {
			t.Fatalf("runCheck error: %v", err)
		}
		if code != ExitInvalid {
			t.Fatalf("expected ExitInvalid, got %d", code)
		}
	})

	t.Run("verify failure returns ExitInvalid", func(t *testing.T) {
		tmpDir := t.TempDir()
		artifacts := cloneArtifacts(defaultArtifacts())
		artifacts["handoff.small.yml"] = `small_version: "1.0.0"
owner: "agent"
summary: "Test handoff"
resume:
  current_task_id: ""
  next_steps: []
links: []
`
		writeArtifacts(t, tmpDir, artifacts)
		mustSaveWorkspace(t, tmpDir, workspace.KindRepoRoot)

		code, _, err := runCheck(tmpDir, false, true, false, workspace.ScopeRoot, false)
		if err != nil {
			t.Fatalf("runCheck error: %v", err)
		}
		if code != ExitInvalid {
			t.Fatalf("expected ExitInvalid, got %d", code)
		}
	})
}

func TestCheckSuccess(t *testing.T) {
	tmpDir := t.TempDir()
	writeArtifacts(t, tmpDir, defaultArtifacts())
	mustSaveWorkspace(t, tmpDir, workspace.KindRepoRoot)

	code, _, err := runCheck(tmpDir, false, true, false, workspace.ScopeRoot, false)
	if err != nil {
		t.Fatalf("runCheck error: %v", err)
	}
	if code != ExitValid {
		t.Fatalf("expected ExitValid, got %d", code)
	}
}

func TestCheckMissingWorkspace(t *testing.T) {
	tmpDir := t.TempDir()
	writeArtifacts(t, tmpDir, defaultArtifacts())
	if err := os.Remove(filepath.Join(tmpDir, ".small", "workspace.small.yml")); err != nil && !os.IsNotExist(err) {
		t.Fatalf("failed to remove workspace metadata: %v", err)
	}
	_, _, err := runCheck(tmpDir, false, true, false, workspace.ScopeRoot, false)
	if err == nil {
		t.Fatal("expected runCheck to error on missing workspace")
	}
}
