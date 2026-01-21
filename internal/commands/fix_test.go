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
