package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/justyn-clark/small-protocol/internal/runstore"
	"github.com/justyn-clark/small-protocol/internal/workspace"
)

func TestRunSnapshotMissingReplayId(t *testing.T) {
	tmpDir := t.TempDir()
	if err := runSelftestInit(tmpDir); err != nil {
		t.Fatalf("failed to init workspace: %v", err)
	}
	handoffPath := filepath.Join(tmpDir, ".small", "handoff.small.yml")
	handoff := `small_version: "1.0.0"
owner: "agent"
summary: "Missing replayId"
resume:
  current_task_id: ""
  next_steps: []
links: []
`
	if err := os.WriteFile(handoffPath, []byte(handoff), 0644); err != nil {
		t.Fatalf("failed to overwrite handoff: %v", err)
	}

	artifactsDir, storeDir, err := resolveRunContext(tmpDir, "", string(workspace.ScopeRoot))
	if err != nil {
		t.Fatalf("failed to resolve run context: %v", err)
	}

	if _, err := runstore.WriteSnapshot(artifactsDir, storeDir, false); err == nil {
		t.Fatalf("expected snapshot to fail without replayId")
	} else if !strings.Contains(err.Error(), "small handoff") {
		t.Fatalf("expected error to mention small handoff, got %v", err)
	}
}

func TestRunSnapshotAfterHandoff(t *testing.T) {
	tmpDir := t.TempDir()
	if err := runSelftestInit(tmpDir); err != nil {
		t.Fatalf("failed to init workspace: %v", err)
	}
	if err := runSelftestHandoff(tmpDir); err != nil {
		t.Fatalf("failed to generate handoff: %v", err)
	}

	artifactsDir, storeDir, err := resolveRunContext(tmpDir, "", string(workspace.ScopeRoot))
	if err != nil {
		t.Fatalf("failed to resolve run context: %v", err)
	}

	snapshot, err := runstore.WriteSnapshot(artifactsDir, storeDir, false)
	if err != nil {
		t.Fatalf("snapshot failed: %v", err)
	}

	if _, err := os.Stat(snapshot.Dir); err != nil {
		t.Fatalf("expected snapshot dir to exist: %v", err)
	}
}

func TestRunListOutputBasic(t *testing.T) {
	tmpDir := t.TempDir()
	if err := runSelftestInit(tmpDir); err != nil {
		t.Fatalf("failed to init workspace: %v", err)
	}
	if err := runSelftestHandoff(tmpDir); err != nil {
		t.Fatalf("failed to generate handoff: %v", err)
	}

	artifactsDir, storeDir, err := resolveRunContext(tmpDir, "", string(workspace.ScopeRoot))
	if err != nil {
		t.Fatalf("failed to resolve run context: %v", err)
	}

	if _, err := runstore.WriteSnapshot(artifactsDir, storeDir, false); err != nil {
		t.Fatalf("snapshot failed: %v", err)
	}

	snapshots, err := runstore.ListSnapshots(storeDir)
	if err != nil {
		t.Fatalf("list snapshots failed: %v", err)
	}

	output, err := formatRunListOutput(snapshots, false)
	if err != nil {
		t.Fatalf("formatRunListOutput failed: %v", err)
	}
	if !strings.Contains(output, "created_at") {
		t.Fatalf("expected output to contain header, got: %s", output)
	}
	if !strings.Contains(output, "Selftest checkpoint") {
		t.Fatalf("expected output to include summary, got: %s", output)
	}
}

func TestRunCheckoutRespectsForce(t *testing.T) {
	sourceDir := t.TempDir()
	if err := runSelftestInit(sourceDir); err != nil {
		t.Fatalf("failed to init source: %v", err)
	}
	if err := runSelftestHandoff(sourceDir); err != nil {
		t.Fatalf("failed to handoff source: %v", err)
	}

	artifactsDir, storeDir, err := resolveRunContext(sourceDir, "", string(workspace.ScopeRoot))
	if err != nil {
		t.Fatalf("failed to resolve run context: %v", err)
	}

	snapshot, err := runstore.WriteSnapshot(artifactsDir, storeDir, false)
	if err != nil {
		t.Fatalf("snapshot failed: %v", err)
	}

	targetDir := t.TempDir()
	if err := runSelftestInit(targetDir); err != nil {
		t.Fatalf("failed to init target: %v", err)
	}
	planPath := filepath.Join(targetDir, ".small", "plan.small.yml")
	if err := os.WriteFile(planPath, []byte("small_version: \"1.0.0\"\nowner: \"agent\"\ntasks:\n  - id: \"task-x\"\n    title: \"Different\"\n"), 0644); err != nil {
		t.Fatalf("failed to mutate plan: %v", err)
	}

	if err := runstore.CheckoutSnapshot(targetDir, storeDir, snapshot.ReplayID, false); err == nil {
		t.Fatalf("expected checkout to fail without --force")
	}

	if err := runstore.CheckoutSnapshot(targetDir, storeDir, snapshot.ReplayID, true); err != nil {
		t.Fatalf("checkout with force failed: %v", err)
	}
}
