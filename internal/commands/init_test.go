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

func TestInitCommandCreatesAgentsMd(t *testing.T) {
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

	agentsPath := filepath.Join(tmpDir, "AGENTS.md")
	data, err := os.ReadFile(agentsPath)
	if err != nil {
		t.Fatalf("failed to read AGENTS.md: %v", err)
	}

	content := string(data)

	// Check for BEGIN/END markers
	if !contains(content, "<!-- BEGIN SMALL HARNESS") {
		t.Error("AGENTS.md missing BEGIN marker")
	}
	if !contains(content, "<!-- END SMALL HARNESS") {
		t.Error("AGENTS.md missing END marker")
	}

	// Check for required content
	requiredContent := []string{
		"Ownership Rules",
		"human",
		"agent",
		"Strict Mode Rules",
		"localhost",
		"Ralph Loop",
	}

	for _, required := range requiredContent {
		if !contains(content, required) {
			t.Errorf("AGENTS.md missing required content: %q", required)
		}
	}
}

func TestInitCommandNoAgentsFlag(t *testing.T) {
	tmpDir := t.TempDir()
	oldBaseDir := baseDir
	baseDir = tmpDir
	defer func() {
		baseDir = oldBaseDir
	}()

	cmd := initCmd()
	cmd.SetArgs([]string{"--no-agents"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("init command failed: %v", err)
	}

	agentsPath := filepath.Join(tmpDir, "AGENTS.md")
	if _, err := os.Stat(agentsPath); err == nil {
		t.Error("AGENTS.md should not be created when --no-agents is set")
	}
}

func TestInitCommandExistingAgentsMdNoFlags(t *testing.T) {
	tmpDir := t.TempDir()
	oldBaseDir := baseDir
	baseDir = tmpDir
	defer func() {
		baseDir = oldBaseDir
	}()

	// Create existing AGENTS.md
	agentsPath := filepath.Join(tmpDir, "AGENTS.md")
	existingContent := "# Existing AGENTS.md\nCustom content"
	if err := os.WriteFile(agentsPath, []byte(existingContent), 0644); err != nil {
		t.Fatalf("failed to write existing AGENTS.md: %v", err)
	}

	cmd := initCmd()
	cmd.SetArgs([]string{})
	err := cmd.Execute()

	// Should fail with guidance message
	if err == nil {
		t.Error("init should fail when AGENTS.md exists without flags")
	}

	// File should be unchanged
	data, readErr := os.ReadFile(agentsPath)
	if readErr != nil {
		t.Fatalf("failed to read AGENTS.md: %v", readErr)
	}
	if string(data) != existingContent {
		t.Error("AGENTS.md should not be modified")
	}
}

func TestInitCommandOverwriteAgentsFlag(t *testing.T) {
	tmpDir := t.TempDir()
	oldBaseDir := baseDir
	baseDir = tmpDir
	defer func() {
		baseDir = oldBaseDir
	}()

	// Create existing AGENTS.md
	agentsPath := filepath.Join(tmpDir, "AGENTS.md")
	existingContent := "# Existing AGENTS.md\nCustom content"
	if err := os.WriteFile(agentsPath, []byte(existingContent), 0644); err != nil {
		t.Fatalf("failed to write existing AGENTS.md: %v", err)
	}

	cmd := initCmd()
	cmd.SetArgs([]string{"--overwrite-agents"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("init command failed: %v", err)
	}

	data, err := os.ReadFile(agentsPath)
	if err != nil {
		t.Fatalf("failed to read AGENTS.md: %v", err)
	}

	if string(data) == existingContent {
		t.Error("AGENTS.md should be overwritten when --overwrite-agents flag is set")
	}

	// Should only contain SMALL block (no existing content)
	if contains(string(data), "Existing AGENTS.md") {
		t.Error("existing content should be replaced entirely")
	}

	if !contains(string(data), "<!-- BEGIN SMALL HARNESS") {
		t.Error("AGENTS.md should contain SMALL harness block after overwrite")
	}
}

func TestInitCommandAgentsModeAppend(t *testing.T) {
	tmpDir := t.TempDir()
	oldBaseDir := baseDir
	baseDir = tmpDir
	defer func() {
		baseDir = oldBaseDir
	}()

	// Create existing AGENTS.md
	agentsPath := filepath.Join(tmpDir, "AGENTS.md")
	existingContent := "# My Custom Agent Instructions\n\nThese are my rules."
	if err := os.WriteFile(agentsPath, []byte(existingContent), 0644); err != nil {
		t.Fatalf("failed to write existing AGENTS.md: %v", err)
	}

	cmd := initCmd()
	cmd.SetArgs([]string{"--agents-mode=append"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("init command failed: %v", err)
	}

	data, err := os.ReadFile(agentsPath)
	if err != nil {
		t.Fatalf("failed to read AGENTS.md: %v", err)
	}

	content := string(data)

	// Should preserve existing content
	if !contains(content, "My Custom Agent Instructions") {
		t.Error("existing content should be preserved")
	}

	// Should have SMALL block
	if !contains(content, "<!-- BEGIN SMALL HARNESS") {
		t.Error("SMALL harness block should be added")
	}

	// Existing content should come BEFORE the block (append mode)
	customIdx := indexOf(content, "My Custom Agent Instructions")
	blockIdx := indexOf(content, "<!-- BEGIN SMALL HARNESS")
	if customIdx >= blockIdx {
		t.Error("existing content should come before SMALL block in append mode")
	}
}

func TestInitCommandAgentsModePrepend(t *testing.T) {
	tmpDir := t.TempDir()
	oldBaseDir := baseDir
	baseDir = tmpDir
	defer func() {
		baseDir = oldBaseDir
	}()

	// Create existing AGENTS.md
	agentsPath := filepath.Join(tmpDir, "AGENTS.md")
	existingContent := "# My Custom Agent Instructions\n\nThese are my rules."
	if err := os.WriteFile(agentsPath, []byte(existingContent), 0644); err != nil {
		t.Fatalf("failed to write existing AGENTS.md: %v", err)
	}

	cmd := initCmd()
	cmd.SetArgs([]string{"--agents-mode=prepend"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("init command failed: %v", err)
	}

	data, err := os.ReadFile(agentsPath)
	if err != nil {
		t.Fatalf("failed to read AGENTS.md: %v", err)
	}

	content := string(data)

	// Should preserve existing content
	if !contains(content, "My Custom Agent Instructions") {
		t.Error("existing content should be preserved")
	}

	// Should have SMALL block
	if !contains(content, "<!-- BEGIN SMALL HARNESS") {
		t.Error("SMALL harness block should be added")
	}

	// Block should come BEFORE existing content (prepend mode)
	customIdx := indexOf(content, "My Custom Agent Instructions")
	blockIdx := indexOf(content, "<!-- BEGIN SMALL HARNESS")
	if blockIdx >= customIdx {
		t.Error("SMALL block should come before existing content in prepend mode")
	}
}

func TestInitCommandAgentsModeReplacesExistingBlock(t *testing.T) {
	tmpDir := t.TempDir()
	oldBaseDir := baseDir
	baseDir = tmpDir
	defer func() {
		baseDir = oldBaseDir
	}()

	// Create existing AGENTS.md with old SMALL block
	agentsPath := filepath.Join(tmpDir, "AGENTS.md")
	existingContent := `# My Custom Agent Instructions

These are my rules.

<!-- BEGIN SMALL HARNESS v1.0.0 -->
# Old SMALL Content
This should be replaced.
<!-- END SMALL HARNESS v1.0.0 -->

# Footer Section
More custom content.
`
	if err := os.WriteFile(agentsPath, []byte(existingContent), 0644); err != nil {
		t.Fatalf("failed to write existing AGENTS.md: %v", err)
	}

	cmd := initCmd()
	cmd.SetArgs([]string{"--agents-mode=append"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("init command failed: %v", err)
	}

	data, err := os.ReadFile(agentsPath)
	if err != nil {
		t.Fatalf("failed to read AGENTS.md: %v", err)
	}

	content := string(data)

	// Should preserve surrounding content
	if !contains(content, "My Custom Agent Instructions") {
		t.Error("header content should be preserved")
	}
	if !contains(content, "Footer Section") {
		t.Error("footer content should be preserved")
	}

	// Should NOT contain old block content
	if contains(content, "Old SMALL Content") {
		t.Error("old block content should be replaced")
	}
	if contains(content, "This should be replaced") {
		t.Error("old block content should be replaced")
	}

	// Should have exactly one block
	beginCount := countOccurrences(content, "<!-- BEGIN SMALL HARNESS")
	endCount := countOccurrences(content, "<!-- END SMALL HARNESS")
	if beginCount != 1 || endCount != 1 {
		t.Errorf("expected exactly 1 block, got %d BEGIN and %d END markers", beginCount, endCount)
	}
}

func TestInitCommandMutuallyExclusiveFlags(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{"agents-mode and no-agents", []string{"--agents-mode=append", "--no-agents"}},
		{"agents-mode and overwrite-agents", []string{"--agents-mode=append", "--overwrite-agents"}},
		{"no-agents and overwrite-agents", []string{"--no-agents", "--overwrite-agents"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			oldBaseDir := baseDir
			baseDir = tmpDir
			defer func() {
				baseDir = oldBaseDir
			}()

			cmd := initCmd()
			cmd.SetArgs(tt.args)
			err := cmd.Execute()
			if err == nil {
				t.Error("expected error for mutually exclusive flags")
			}
		})
	}
}

// Helper functions

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

func countOccurrences(s, substr string) int {
	count := 0
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			count++
		}
	}
	return count
}
