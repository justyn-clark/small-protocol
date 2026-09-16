package commands

import (
	"strings"
	"testing"

	"github.com/justyn-clark/small-protocol/internal/runstore"
	"github.com/justyn-clark/small-protocol/internal/workspace"
)

func TestInspectWorkspaceHealthReportsDigestAndSnapshots(t *testing.T) {
	tmpDir := t.TempDir()
	if err := runSelftestInit(tmpDir); err != nil {
		t.Fatalf("runSelftestInit failed: %v", err)
	}
	if err := runSelftestHandoff(tmpDir); err != nil {
		t.Fatalf("runSelftestHandoff failed: %v", err)
	}

	storeDir := runstore.ResolveStoreDir(tmpDir, "")
	if _, err := runstore.WriteSnapshot(tmpDir, storeDir, false); err != nil {
		t.Fatalf("WriteSnapshot failed: %v", err)
	}

	health := inspectWorkspaceHealth(tmpDir, "", workspace.ScopeAny)
	if health.StrictStatus != "ok" {
		t.Fatalf("expected strict status ok, got %s: %v", health.StrictStatus, health.StrictErrors)
	}
	if health.ArtifactDigest == "" {
		t.Fatalf("expected current artifact digest")
	}
	if health.SnapshotCount != 1 {
		t.Fatalf("expected one snapshot, got %d", health.SnapshotCount)
	}
	if health.LatestSnapshotArtifactDigest == "" {
		t.Fatalf("expected latest snapshot artifact digest")
	}
	if health.LatestSnapshotStale {
		t.Fatalf("snapshot should not be stale immediately after snapshot")
	}
}

func TestFormatHealthTextIncludesCoreColumns(t *testing.T) {
	output := formatHealthText(healthOutput{Workspaces: []healthWorkspace{{
		Path:           "/tmp/example",
		StrictStatus:   "ok",
		ReplayID:       "1234567890abcdef",
		ArtifactDigest: "abcdef1234567890",
		SnapshotCount:  2,
	}}})

	for _, expected := range []string{"strict", "snapshots", "artifact_digest", "/tmp/example"} {
		if !strings.Contains(output, expected) {
			t.Fatalf("expected health output to contain %q, got %s", expected, output)
		}
	}
}
