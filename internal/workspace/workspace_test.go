package workspace

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/justyn-clark/small-protocol/internal/small"
)

func TestLoadInvalidKindIncludesValidKinds(t *testing.T) {
	tmpDir := t.TempDir()
	smallDir := filepath.Join(tmpDir, small.SmallDir)
	if err := os.MkdirAll(smallDir, 0755); err != nil {
		t.Fatalf("failed to create %s: %v", smallDir, err)
	}

	content := fmt.Sprintf("small_version: %q\nkind: invalid-kind\n", small.ProtocolVersion)
	if err := os.WriteFile(filepath.Join(smallDir, "workspace.small.yml"), []byte(content), 0644); err != nil {
		t.Fatalf("failed to write workspace metadata: %v", err)
	}

	_, err := Load(tmpDir)
	if err == nil {
		t.Fatalf("expected error loading workspace metadata with invalid kind")
	}

	errMsg := err.Error()
	if !strings.Contains(errMsg, "valid kinds") {
		t.Fatalf("error should mention valid kinds, got %q", errMsg)
	}

	if !strings.Contains(errMsg, string(KindRepoRoot)) {
		t.Fatalf("error should include %q, got %q", KindRepoRoot, errMsg)
	}

	if !strings.Contains(errMsg, string(KindExamples)) {
		t.Fatalf("error should include %q, got %q", KindExamples, errMsg)
	}
}

func TestTouchUpdatedAt(t *testing.T) {
	tmpDir := t.TempDir()
	if err := Save(tmpDir, KindRepoRoot); err != nil {
		t.Fatalf("failed to save workspace: %v", err)
	}

	before, err := Load(tmpDir)
	if err != nil {
		t.Fatalf("failed to load workspace before: %v", err)
	}
	if before.UpdatedAt == "" {
		t.Fatalf("expected updated_at to be set")
	}

	if _, err := TouchUpdatedAt(tmpDir); err != nil {
		t.Fatalf("TouchUpdatedAt failed: %v", err)
	}
	after, err := Load(tmpDir)
	if err != nil {
		t.Fatalf("failed to load workspace after: %v", err)
	}
	if after.UpdatedAt == "" {
		t.Fatalf("expected updated_at after touch")
	}
	if after.UpdatedAt < before.UpdatedAt {
		t.Fatalf("expected updated_at to advance")
	}
}

func TestSetAndGetRunReplayID(t *testing.T) {
	tmpDir := t.TempDir()
	if err := Save(tmpDir, KindRepoRoot); err != nil {
		t.Fatalf("failed to save workspace: %v", err)
	}

	const replayID = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	if err := SetRunReplayID(tmpDir, replayID); err != nil {
		t.Fatalf("SetRunReplayID failed: %v", err)
	}

	value, err := RunReplayID(tmpDir)
	if err != nil {
		t.Fatalf("RunReplayID failed: %v", err)
	}
	if value != replayID {
		t.Fatalf("RunReplayID = %q, want %q", value, replayID)
	}

	info, err := Load(tmpDir)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if info.Run == nil || info.Run.ReplayID != replayID {
		t.Fatalf("workspace run replay_id not persisted: %+v", info.Run)
	}
}
