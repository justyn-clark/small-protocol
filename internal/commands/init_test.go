package commands

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/justyn-clark/small-protocol/internal/workspace"
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
