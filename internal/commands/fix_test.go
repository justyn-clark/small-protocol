package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/justyn-clark/small-protocol/internal/small"
	"github.com/justyn-clark/small-protocol/internal/workspace"
)

func TestFixVersionsNormalizesSmallVersion(t *testing.T) {
	tmpDir := t.TempDir()
	smallDir := filepath.Join(tmpDir, small.SmallDir)
	if err := os.MkdirAll(smallDir, 0o755); err != nil {
		t.Fatalf("failed to create .small dir: %v", err)
	}

	intent := "small_version: 1.0.0\nowner: \"human\"\nintent: \"Test\"\nscope:\n  include: []\n  exclude: []\nsuccess_criteria: []\n"
	plan := "small_version: \"1.0.0\"\nowner: \"agent\"\ntasks:\n  - id: \"task-1\"\n    title: \"Test\"\n"

	if err := os.WriteFile(filepath.Join(smallDir, "intent.small.yml"), []byte(intent), 0o644); err != nil {
		t.Fatalf("failed to write intent: %v", err)
	}
	if err := os.WriteFile(filepath.Join(smallDir, "plan.small.yml"), []byte(plan), 0o644); err != nil {
		t.Fatalf("failed to write plan: %v", err)
	}
	if err := workspace.Save(tmpDir, workspace.KindRepoRoot); err != nil {
		t.Fatalf("failed to save workspace: %v", err)
	}

	cmd := fixCmd()
	cmd.SetArgs([]string{"--versions", "--dir", tmpDir, "--workspace", "any"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("fix failed: %v", err)
	}

	updatedIntent, err := os.ReadFile(filepath.Join(smallDir, "intent.small.yml"))
	if err != nil {
		t.Fatalf("failed to read intent: %v", err)
	}
	if !strings.Contains(string(updatedIntent), `small_version: "1.0.0"`) {
		t.Fatalf("expected intent small_version to be quoted after fix")
	}
	updatedPlan, err := os.ReadFile(filepath.Join(smallDir, "plan.small.yml"))
	if err != nil {
		t.Fatalf("failed to read plan: %v", err)
	}
	if !strings.Contains(string(updatedPlan), `small_version: "1.0.0"`) {
		t.Fatalf("expected plan to remain quoted")
	}
}

func TestFixWorkspaceCreatesMissingWorkspace(t *testing.T) {
	tmpDir := t.TempDir()
	smallDir := filepath.Join(tmpDir, small.SmallDir)
	if err := os.MkdirAll(smallDir, 0o755); err != nil {
		t.Fatalf("failed to create .small dir: %v", err)
	}

	// No workspace file exists
	wsPath := filepath.Join(smallDir, "workspace.small.yml")
	if _, err := os.Stat(wsPath); err == nil {
		t.Fatal("workspace file should not exist before test")
	}

	result, err := workspace.Fix(tmpDir, workspace.KindRepoRoot, false)
	if err != nil {
		t.Fatalf("fix workspace failed: %v", err)
	}

	if !result.Created {
		t.Error("expected workspace to be created")
	}
	if !result.AddedOwner {
		t.Error("expected owner to be added")
	}
	if !result.AddedCreatedAt {
		t.Error("expected created_at to be added")
	}
	if !result.AddedUpdatedAt {
		t.Error("expected updated_at to be added")
	}

	// Verify file was created
	data, err := os.ReadFile(wsPath)
	if err != nil {
		t.Fatalf("failed to read workspace: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, `small_version: "1.0.0"`) {
		t.Error("expected workspace to have small_version")
	}
	if !strings.Contains(content, `owner: agent`) {
		t.Error("expected workspace to have owner")
	}
	if !strings.Contains(content, "created_at:") {
		t.Error("expected workspace to have created_at")
	}
	if !strings.Contains(content, "updated_at:") {
		t.Error("expected workspace to have updated_at")
	}
}

func TestFixWorkspaceRepairsMissingTimestamps(t *testing.T) {
	tmpDir := t.TempDir()
	smallDir := filepath.Join(tmpDir, small.SmallDir)
	if err := os.MkdirAll(smallDir, 0o755); err != nil {
		t.Fatalf("failed to create .small dir: %v", err)
	}

	// Create workspace without timestamps
	wsPath := filepath.Join(smallDir, "workspace.small.yml")
	oldWs := `small_version: "1.0.0"
kind: repo-root
`
	if err := os.WriteFile(wsPath, []byte(oldWs), 0o644); err != nil {
		t.Fatalf("failed to write workspace: %v", err)
	}

	result, err := workspace.Fix(tmpDir, workspace.KindRepoRoot, false)
	if err != nil {
		t.Fatalf("fix workspace failed: %v", err)
	}

	if result.Created {
		t.Error("expected workspace to be repaired, not created")
	}
	if !result.AddedOwner {
		t.Error("expected owner to be added")
	}
	if !result.AddedCreatedAt {
		t.Error("expected created_at to be added")
	}
	if !result.AddedUpdatedAt {
		t.Error("expected updated_at to be added")
	}

	// Verify file was repaired
	data, err := os.ReadFile(wsPath)
	if err != nil {
		t.Fatalf("failed to read workspace: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "created_at:") {
		t.Error("expected workspace to have created_at after repair")
	}
	if !strings.Contains(content, "updated_at:") {
		t.Error("expected workspace to have updated_at after repair")
	}
	if !strings.Contains(content, `owner: agent`) {
		t.Error("expected workspace to have owner after repair")
	}
}

func TestFixWorkspaceValidWorkspaceNoOp(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a valid workspace using Save
	if err := workspace.Save(tmpDir, workspace.KindRepoRoot); err != nil {
		t.Fatalf("failed to save workspace: %v", err)
	}

	// Read original content
	wsPath := filepath.Join(tmpDir, small.SmallDir, "workspace.small.yml")
	originalData, err := os.ReadFile(wsPath)
	if err != nil {
		t.Fatalf("failed to read workspace: %v", err)
	}

	result, err := workspace.Fix(tmpDir, workspace.KindRepoRoot, false)
	if err != nil {
		t.Fatalf("fix workspace failed: %v", err)
	}

	if result.Created {
		t.Error("expected workspace not to be created")
	}

	// updated_at will always be updated, but other fields should not change
	if result.AddedOwner {
		t.Error("expected owner not to be added (already present)")
	}
	if result.AddedCreatedAt {
		t.Error("expected created_at not to be added (already present)")
	}

	// Verify file is still valid
	data, err := os.ReadFile(wsPath)
	if err != nil {
		t.Fatalf("failed to read workspace: %v", err)
	}

	// Should have preserved most of original content
	if !strings.Contains(string(data), `small_version: "1.0.0"`) {
		t.Error("expected workspace to still have small_version")
	}
	if !strings.Contains(string(originalData), `kind: repo-root`) {
		t.Error("expected original to have kind")
	}
}

func TestFixWorkspaceSubcommand(t *testing.T) {
	tmpDir := t.TempDir()
	smallDir := filepath.Join(tmpDir, small.SmallDir)
	if err := os.MkdirAll(smallDir, 0o755); err != nil {
		t.Fatalf("failed to create .small dir: %v", err)
	}

	cmd := fixCmd()
	cmd.SetArgs([]string{"workspace", "--dir", tmpDir})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("fix workspace subcommand failed: %v", err)
	}

	// Verify workspace was created
	wsPath := filepath.Join(smallDir, "workspace.small.yml")
	data, err := os.ReadFile(wsPath)
	if err != nil {
		t.Fatalf("failed to read workspace: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, `small_version: "1.0.0"`) {
		t.Error("expected workspace to have small_version")
	}
}

func TestFixRuntimeLayoutMigratesLegacyPaths(t *testing.T) {
	tmpDir := t.TempDir()
	smallDir := filepath.Join(tmpDir, small.SmallDir)
	if err := os.MkdirAll(filepath.Join(smallDir, "archive", "run-1"), 0o755); err != nil {
		t.Fatalf("failed to create legacy archive: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(smallDir, "runs"), 0o755); err != nil {
		t.Fatalf("failed to create legacy runs: %v", err)
	}
	if err := os.WriteFile(filepath.Join(smallDir, "archive", "run-1", "archive.small.yml"), []byte("archive"), 0o644); err != nil {
		t.Fatalf("failed to seed legacy archive: %v", err)
	}
	if err := os.WriteFile(filepath.Join(smallDir, "runs", small.RunIndexFileName), []byte("index"), 0o644); err != nil {
		t.Fatalf("failed to seed legacy run index: %v", err)
	}
	writeFixArtifact(t, tmpDir, "progress.small.yml", map[string]any{
		"small_version": small.ProtocolVersion,
		"owner":         "agent",
		"entries":       []any{},
	})
	if err := workspace.Save(tmpDir, workspace.KindRepoRoot); err != nil {
		t.Fatalf("failed to save workspace: %v", err)
	}

	cmd := fixCmd()
	cmd.SetArgs([]string{"--runtime-layout", "--dir", tmpDir, "--workspace", "any"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("fix runtime layout failed: %v", err)
	}

	if _, err := os.Stat(filepath.Join(tmpDir, small.ArchiveStoreDirName, "run-1", "archive.small.yml")); err != nil {
		t.Fatalf("expected migrated archive: %v", err)
	}
	if _, err := os.Stat(filepath.Join(tmpDir, small.RunStoreDirName, small.RunIndexFileName)); err != nil {
		t.Fatalf("expected migrated run index: %v", err)
	}
	progressData, err := os.ReadFile(filepath.Join(smallDir, "progress.small.yml"))
	if err != nil {
		t.Fatalf("failed to read progress: %v", err)
	}
	if !strings.Contains(string(progressData), "meta/reconcile-runtime-layout") {
		t.Fatalf("expected runtime layout reconcile progress entry, got %q", string(progressData))
	}
}

func writeFixArtifact(t *testing.T, baseDir, filename string, data map[string]any) {
	t.Helper()
	if err := small.SaveArtifact(baseDir, filename, data); err != nil {
		t.Fatalf("failed to write %s: %v", filename, err)
	}
}
