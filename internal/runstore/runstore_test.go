package runstore

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/justyn-clark/small-protocol/internal/workspace"
)

func TestWriteSnapshotCreatesFiles(t *testing.T) {
	tmpDir := t.TempDir()
	writeTestWorkspace(t, tmpDir, "abc123", "Snapshot one", true)

	storeDir := filepath.Join(tmpDir, DefaultStoreDirName)
	snapshot, err := WriteSnapshot(tmpDir, storeDir, false)
	if err != nil {
		t.Fatalf("WriteSnapshot failed: %v", err)
	}

	if snapshot.ReplayID == "" {
		t.Fatalf("expected replayId to be set")
	}

	for _, filename := range RequiredArtifacts {
		path := filepath.Join(snapshot.Dir, filename)
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected %s to be copied: %v", filename, err)
		}
	}

	constraintsPath := filepath.Join(snapshot.Dir, "constraints.small.yml")
	if _, err := os.Stat(constraintsPath); err != nil {
		t.Fatalf("expected constraints.small.yml to be copied: %v", err)
	}

	meta, err := ReadMeta(snapshot.Dir)
	if err != nil {
		t.Fatalf("failed to read meta.json: %v", err)
	}
	if meta.ReplayID == "" || meta.CreatedAt == "" {
		t.Fatalf("expected meta.json to include replayId and created_at")
	}
	if meta.WorkspaceKind == "" {
		t.Fatalf("expected workspace_kind to be populated")
	}
}

func TestWriteSnapshotRefusesOverwrite(t *testing.T) {
	tmpDir := t.TempDir()
	writeTestWorkspace(t, tmpDir, "abc123", "Snapshot one", false)

	storeDir := filepath.Join(tmpDir, DefaultStoreDirName)
	if _, err := WriteSnapshot(tmpDir, storeDir, false); err != nil {
		t.Fatalf("initial snapshot failed: %v", err)
	}

	if _, err := WriteSnapshot(tmpDir, storeDir, false); err == nil {
		t.Fatalf("expected snapshot overwrite to fail")
	} else if !strings.Contains(err.Error(), "--force") {
		t.Fatalf("expected error to mention --force, got %v", err)
	}
}

func TestListSnapshotsSortsByCreatedAt(t *testing.T) {
	storeDir := filepath.Join(t.TempDir(), DefaultStoreDirName)
	if err := os.MkdirAll(storeDir, 0755); err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	olderDir := filepath.Join(storeDir, "older")
	newerDir := filepath.Join(storeDir, "newer")
	if err := os.MkdirAll(olderDir, 0755); err != nil {
		t.Fatalf("failed to create older dir: %v", err)
	}
	if err := os.MkdirAll(newerDir, 0755); err != nil {
		t.Fatalf("failed to create newer dir: %v", err)
	}

	olderMeta := Meta{ReplayID: "older", CreatedAt: time.Now().Add(-2 * time.Hour).UTC().Format(time.RFC3339Nano)}
	newerMeta := Meta{ReplayID: "newer", CreatedAt: time.Now().Add(-1 * time.Hour).UTC().Format(time.RFC3339Nano)}
	if err := WriteMeta(olderDir, olderMeta); err != nil {
		t.Fatalf("failed to write older meta: %v", err)
	}
	if err := WriteMeta(newerDir, newerMeta); err != nil {
		t.Fatalf("failed to write newer meta: %v", err)
	}

	writeTestHandoff(t, olderDir, "older", "Older summary")
	writeTestHandoff(t, newerDir, "newer", "Newer summary")

	snapshots, err := ListSnapshots(storeDir)
	if err != nil {
		t.Fatalf("ListSnapshots failed: %v", err)
	}
	if len(snapshots) != 2 {
		t.Fatalf("expected 2 snapshots, got %d", len(snapshots))
	}
	if snapshots[0].ReplayID != "newer" {
		t.Fatalf("expected newest snapshot first, got %s", snapshots[0].ReplayID)
	}
}

func TestLoadSnapshotReadsMetaAndHandoff(t *testing.T) {
	storeDir := filepath.Join(t.TempDir(), DefaultStoreDirName)
	if err := os.MkdirAll(storeDir, 0755); err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	snapshotDir := filepath.Join(storeDir, "snap")
	if err := os.MkdirAll(snapshotDir, 0755); err != nil {
		t.Fatalf("failed to create snapshot dir: %v", err)
	}

	meta := Meta{ReplayID: "snap", CreatedAt: time.Now().UTC().Format(time.RFC3339Nano)}
	if err := WriteMeta(snapshotDir, meta); err != nil {
		t.Fatalf("failed to write meta: %v", err)
	}
	writeTestHandoff(t, snapshotDir, "snap", "Snapshot summary")
	writeTestArtifacts(t, snapshotDir)

	loaded, err := LoadSnapshot(storeDir, "snap")
	if err != nil {
		t.Fatalf("LoadSnapshot failed: %v", err)
	}
	if loaded.Meta.ReplayID != "snap" {
		t.Fatalf("expected replayId snap, got %s", loaded.Meta.ReplayID)
	}
	if loaded.HandoffSummary != "Snapshot summary" {
		t.Fatalf("expected handoff summary, got %s", loaded.HandoffSummary)
	}
	if len(loaded.Artifacts) == 0 {
		t.Fatalf("expected artifacts to be listed")
	}
}

func TestCheckoutSnapshotRestoresArtifacts(t *testing.T) {
	sourceDir := t.TempDir()
	writeTestWorkspace(t, sourceDir, "source", "Source summary", true)

	storeDir := filepath.Join(sourceDir, DefaultStoreDirName)
	snapshot, err := WriteSnapshot(sourceDir, storeDir, false)
	if err != nil {
		t.Fatalf("failed to create snapshot: %v", err)
	}

	targetDir := t.TempDir()
	writeTestWorkspace(t, targetDir, "target", "Target summary", true)
	workspacePath := filepath.Join(targetDir, ".small", "workspace.small.yml")
	workspaceData, err := os.ReadFile(workspacePath)
	if err != nil {
		t.Fatalf("failed to read workspace metadata: %v", err)
	}

	if err := CheckoutSnapshot(targetDir, storeDir, snapshot.ReplayID, true); err != nil {
		t.Fatalf("CheckoutSnapshot failed: %v", err)
	}

	handoffPath := filepath.Join(targetDir, ".small", "handoff.small.yml")
	handoffData, err := os.ReadFile(handoffPath)
	if err != nil {
		t.Fatalf("failed to read restored handoff: %v", err)
	}
	if !strings.Contains(string(handoffData), "Source summary") {
		t.Fatalf("expected restored handoff from snapshot")
	}

	updatedWorkspace, err := os.ReadFile(workspacePath)
	if err != nil {
		t.Fatalf("failed to read workspace metadata after checkout: %v", err)
	}
	if string(updatedWorkspace) != string(workspaceData) {
		t.Fatalf("expected workspace.small.yml to be preserved")
	}
}

func writeTestWorkspace(t *testing.T, dir, replayID, summary string, includeConstraints bool) {
	t.Helper()
	smallDir := filepath.Join(dir, ".small")
	if err := os.MkdirAll(smallDir, 0755); err != nil {
		t.Fatalf("failed to create .small dir: %v", err)
	}

	intent := `small_version: "1.0.0"
owner: "human"
intent: "Test intent"
scope:
  include: []
  exclude: []
success_criteria: []
`
	if err := os.WriteFile(filepath.Join(smallDir, "intent.small.yml"), []byte(intent), 0644); err != nil {
		t.Fatalf("failed to write intent: %v", err)
	}

	plan := `small_version: "1.0.0"
owner: "agent"
tasks: []
`
	if err := os.WriteFile(filepath.Join(smallDir, "plan.small.yml"), []byte(plan), 0644); err != nil {
		t.Fatalf("failed to write plan: %v", err)
	}

	progress := `small_version: "1.0.0"
owner: "agent"
entries: []
`
	if err := os.WriteFile(filepath.Join(smallDir, "progress.small.yml"), []byte(progress), 0644); err != nil {
		t.Fatalf("failed to write progress: %v", err)
	}

	handoff := fmt.Sprintf(`small_version: "1.0.0"
owner: "agent"
summary: %q
resume:
  current_task_id: ""
  next_steps: []
links: []
replayId:
  value: %q
  source: "auto"
`, summary, replayID)
	if err := os.WriteFile(filepath.Join(smallDir, "handoff.small.yml"), []byte(handoff), 0644); err != nil {
		t.Fatalf("failed to write handoff: %v", err)
	}

	if includeConstraints {
		constraints := `small_version: "1.0.0"
owner: "human"
constraints: []
`
		if err := os.WriteFile(filepath.Join(smallDir, "constraints.small.yml"), []byte(constraints), 0644); err != nil {
			t.Fatalf("failed to write constraints: %v", err)
		}
	}

	if err := workspace.Save(dir, workspace.KindRepoRoot); err != nil {
		t.Fatalf("failed to write workspace metadata: %v", err)
	}
}

func writeTestHandoff(t *testing.T, dir, replayID, summary string) {
	t.Helper()
	handoff := fmt.Sprintf(`small_version: "1.0.0"
owner: "agent"
summary: %q
resume:
  current_task_id: ""
  next_steps: ["step-1"]
links: []
replayId:
  value: %q
  source: "auto"
`, summary, replayID)
	if err := os.WriteFile(filepath.Join(dir, "handoff.small.yml"), []byte(handoff), 0644); err != nil {
		t.Fatalf("failed to write handoff: %v", err)
	}
}

func writeTestArtifacts(t *testing.T, dir string) {
	t.Helper()
	for _, filename := range RequiredArtifacts {
		if filename == "handoff.small.yml" {
			continue
		}
		path := filepath.Join(dir, filename)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			if err := os.WriteFile(path, []byte("test"), 0644); err != nil {
				t.Fatalf("failed to write %s: %v", filename, err)
			}
		}
	}
	for _, filename := range OptionalArtifacts {
		path := filepath.Join(dir, filename)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			if err := os.WriteFile(path, []byte("optional"), 0644); err != nil {
				t.Fatalf("failed to write %s: %v", filename, err)
			}
		}
	}
}
