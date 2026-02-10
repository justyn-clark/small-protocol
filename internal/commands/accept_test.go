package commands

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/justyn-clark/small-protocol/internal/small"
	"github.com/justyn-clark/small-protocol/internal/workspace"
)

func TestAcceptIntentBootstrapDoesNotStampReplayID(t *testing.T) {
	tmpDir := t.TempDir()
	writeArtifacts(t, tmpDir, defaultArtifacts())
	mustSaveWorkspace(t, tmpDir, workspace.KindRepoRoot)

	draftDir := small.CacheDraftsDir(tmpDir)
	if err := os.MkdirAll(draftDir, 0o755); err != nil {
		t.Fatalf("failed to create draft dir: %v", err)
	}
	intent := "small_version: \"1.0.0\"\nowner: \"human\"\nintent: \"accepted intent\"\n"
	if err := os.WriteFile(filepath.Join(draftDir, "intent.small.yml"), []byte(intent), 0o644); err != nil {
		t.Fatalf("failed to write draft intent: %v", err)
	}

	cmd := acceptCmd()
	cmd.SetArgs([]string{"intent", "--dir", tmpDir})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("accept intent failed: %v", err)
	}

	progress, err := loadProgressData(filepath.Join(tmpDir, ".small", "progress.small.yml"))
	if err != nil {
		t.Fatalf("failed to load progress: %v", err)
	}
	if len(progress.Entries) == 0 {
		t.Fatal("expected accept to append a progress entry")
	}
	entry := progress.Entries[len(progress.Entries)-1]
	if entry["task_id"] != "meta/accept-intent" {
		t.Fatalf("task_id = %v, want meta/accept-intent", entry["task_id"])
	}
	if replayID, ok := entry["replayId"]; ok && replayID != nil && replayID != "" {
		t.Fatalf("expected replayId to be absent/empty for bootstrap accept, got %v", replayID)
	}
}
