package commands

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/justyn-clark/small-protocol/internal/small"
	"github.com/justyn-clark/small-protocol/internal/workspace"
	"gopkg.in/yaml.v3"
)

func TestInitCommandWritesWorkspaceMetadata(t *testing.T) {
	tmpDir := t.TempDir()
	oldBaseDir := baseDir
	baseDir = tmpDir
	defer func() {
		baseDir = oldBaseDir
	}()

	cmd := initCmd()
	cmd.SetArgs([]string{})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("init command failed: %v", err)
	}

	info, err := workspace.Load(tmpDir)
	if err != nil {
		t.Fatalf("failed to load workspace metadata: %v", err)
	}

	if info.Kind != workspace.KindRepoRoot {
		t.Fatalf("expected workspace kind %q, got %q", workspace.KindRepoRoot, info.Kind)
	}

	progressPath := filepath.Join(tmpDir, ".small", "progress.small.yml")
	data, err := os.ReadFile(progressPath)
	if err != nil {
		t.Fatalf("failed to read progress file: %v", err)
	}

	var progress ProgressData
	if err := yaml.Unmarshal(data, &progress); err != nil {
		t.Fatalf("failed to parse progress file: %v", err)
	}
	if len(progress.Entries) == 0 {
		t.Fatal("expected init to append a progress entry")
	}

	entry := progress.Entries[len(progress.Entries)-1]
	if entry["task_id"] != "init" {
		t.Fatalf("expected init task_id, got %v", entry["task_id"])
	}
	timestamp, _ := entry["timestamp"].(string)
	if _, err := small.ParseProgressTimestamp(timestamp); err != nil {
		t.Fatalf("invalid init timestamp %q: %v", timestamp, err)
	}
}

func TestInitCommandWithDirFlag(t *testing.T) {
	tmpDir := t.TempDir()
	targetDir := filepath.Join(tmpDir, "project")
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		t.Fatalf("failed to create target directory: %v", err)
	}

	oldBaseDir := baseDir
	baseDir = tmpDir
	defer func() {
		baseDir = oldBaseDir
	}()

	cmd := initCmd()
	cmd.SetArgs([]string{"--dir", targetDir})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("init command failed: %v", err)
	}

	info, err := workspace.Load(targetDir)
	if err != nil {
		t.Fatalf("failed to load workspace metadata: %v", err)
	}

	if info.Kind != workspace.KindRepoRoot {
		t.Fatalf("expected workspace kind %q, got %q", workspace.KindRepoRoot, info.Kind)
	}
}
