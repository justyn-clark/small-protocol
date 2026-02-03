package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAgentsApplyCreatesWhenMissing(t *testing.T) {
	tmpDir := t.TempDir()
	oldBaseDir := baseDir
	baseDir = tmpDir
	defer func() {
		baseDir = oldBaseDir
	}()

	cmd := agentsApplyCmd()
	cmd.SetArgs([]string{})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("agents apply failed: %v", err)
	}

	agentsPath := filepath.Join(tmpDir, "AGENTS.md")
	data, err := os.ReadFile(agentsPath)
	if err != nil {
		t.Fatalf("failed to read AGENTS.md: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "<!-- BEGIN SMALL HARNESS") {
		t.Error("AGENTS.md missing BEGIN marker")
	}
	if !strings.Contains(content, "<!-- END SMALL HARNESS") {
		t.Error("AGENTS.md missing END marker")
	}
}

func TestAgentsApplyReplacesExistingBlock(t *testing.T) {
	tmpDir := t.TempDir()
	oldBaseDir := baseDir
	baseDir = tmpDir
	defer func() {
		baseDir = oldBaseDir
	}()

	// Create existing AGENTS.md with old block
	agentsPath := filepath.Join(tmpDir, "AGENTS.md")
	existingContent := `# Custom Header

<!-- BEGIN SMALL HARNESS v1.0.0 -->
# Old Content
This should be replaced
<!-- END SMALL HARNESS v1.0.0 -->

# Custom Footer
`
	if err := os.WriteFile(agentsPath, []byte(existingContent), 0644); err != nil {
		t.Fatalf("failed to write existing AGENTS.md: %v", err)
	}

	cmd := agentsApplyCmd()
	cmd.SetArgs([]string{})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("agents apply failed: %v", err)
	}

	data, err := os.ReadFile(agentsPath)
	if err != nil {
		t.Fatalf("failed to read AGENTS.md: %v", err)
	}

	content := string(data)

	// Should preserve surrounding content
	if !strings.Contains(content, "Custom Header") {
		t.Error("header should be preserved")
	}
	if !strings.Contains(content, "Custom Footer") {
		t.Error("footer should be preserved")
	}

	// Should not contain old block content
	if strings.Contains(content, "Old Content") {
		t.Error("old block content should be replaced")
	}
	if strings.Contains(content, "This should be replaced") {
		t.Error("old block content should be replaced")
	}

	// Should have exactly one block
	beginCount := strings.Count(content, "<!-- BEGIN SMALL HARNESS")
	endCount := strings.Count(content, "<!-- END SMALL HARNESS")
	if beginCount != 1 || endCount != 1 {
		t.Errorf("expected exactly 1 block, got %d BEGIN and %d END markers", beginCount, endCount)
	}
}

func TestAgentsApplyErrorsWhenExistsNoFlagsNoBlock(t *testing.T) {
	tmpDir := t.TempDir()
	oldBaseDir := baseDir
	baseDir = tmpDir
	defer func() {
		baseDir = oldBaseDir
	}()

	// Create existing AGENTS.md without SMALL block
	agentsPath := filepath.Join(tmpDir, "AGENTS.md")
	existingContent := "# My Custom AGENTS.md\n\nThis has no SMALL block."
	if err := os.WriteFile(agentsPath, []byte(existingContent), 0644); err != nil {
		t.Fatalf("failed to write existing AGENTS.md: %v", err)
	}

	cmd := agentsApplyCmd()
	cmd.SetArgs([]string{})
	err := cmd.Execute()

	if err == nil {
		t.Error("agents apply should fail when file exists without block and no flags")
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

func TestAgentsApplyAppendMode(t *testing.T) {
	tmpDir := t.TempDir()
	oldBaseDir := baseDir
	baseDir = tmpDir
	defer func() {
		baseDir = oldBaseDir
	}()

	// Create existing AGENTS.md without SMALL block
	agentsPath := filepath.Join(tmpDir, "AGENTS.md")
	existingContent := "# My Custom AGENTS.md\n\nThese are my rules."
	if err := os.WriteFile(agentsPath, []byte(existingContent), 0644); err != nil {
		t.Fatalf("failed to write existing AGENTS.md: %v", err)
	}

	cmd := agentsApplyCmd()
	cmd.SetArgs([]string{"--agents-mode=append"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("agents apply failed: %v", err)
	}

	data, err := os.ReadFile(agentsPath)
	if err != nil {
		t.Fatalf("failed to read AGENTS.md: %v", err)
	}

	content := string(data)

	// Should preserve existing content
	if !strings.Contains(content, "My Custom AGENTS.md") {
		t.Error("existing content should be preserved")
	}

	// Should have SMALL block
	if !strings.Contains(content, "<!-- BEGIN SMALL HARNESS") {
		t.Error("SMALL harness block should be added")
	}

	// Existing content should come BEFORE the block (append mode)
	customIdx := strings.Index(content, "My Custom AGENTS.md")
	blockIdx := strings.Index(content, "<!-- BEGIN SMALL HARNESS")
	if customIdx >= blockIdx {
		t.Error("existing content should come before SMALL block in append mode")
	}
}

func TestAgentsApplyPrependMode(t *testing.T) {
	tmpDir := t.TempDir()
	oldBaseDir := baseDir
	baseDir = tmpDir
	defer func() {
		baseDir = oldBaseDir
	}()

	// Create existing AGENTS.md without SMALL block
	agentsPath := filepath.Join(tmpDir, "AGENTS.md")
	existingContent := "# My Custom AGENTS.md\n\nThese are my rules."
	if err := os.WriteFile(agentsPath, []byte(existingContent), 0644); err != nil {
		t.Fatalf("failed to write existing AGENTS.md: %v", err)
	}

	cmd := agentsApplyCmd()
	cmd.SetArgs([]string{"--agents-mode=prepend"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("agents apply failed: %v", err)
	}

	data, err := os.ReadFile(agentsPath)
	if err != nil {
		t.Fatalf("failed to read AGENTS.md: %v", err)
	}

	content := string(data)

	// Should preserve existing content
	if !strings.Contains(content, "My Custom AGENTS.md") {
		t.Error("existing content should be preserved")
	}

	// Should have SMALL block
	if !strings.Contains(content, "<!-- BEGIN SMALL HARNESS") {
		t.Error("SMALL harness block should be added")
	}

	// Block should come BEFORE existing content (prepend mode)
	customIdx := strings.Index(content, "My Custom AGENTS.md")
	blockIdx := strings.Index(content, "<!-- BEGIN SMALL HARNESS")
	if blockIdx >= customIdx {
		t.Error("SMALL block should come before existing content in prepend mode")
	}
}

func TestAgentsApplyOverwrite(t *testing.T) {
	tmpDir := t.TempDir()
	oldBaseDir := baseDir
	baseDir = tmpDir
	defer func() {
		baseDir = oldBaseDir
	}()

	// Create existing AGENTS.md
	agentsPath := filepath.Join(tmpDir, "AGENTS.md")
	existingContent := "# My Custom AGENTS.md\n\nThese are my rules."
	if err := os.WriteFile(agentsPath, []byte(existingContent), 0644); err != nil {
		t.Fatalf("failed to write existing AGENTS.md: %v", err)
	}

	cmd := agentsApplyCmd()
	cmd.SetArgs([]string{"--overwrite-agents"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("agents apply failed: %v", err)
	}

	data, err := os.ReadFile(agentsPath)
	if err != nil {
		t.Fatalf("failed to read AGENTS.md: %v", err)
	}

	content := string(data)

	// Should NOT contain existing content
	if strings.Contains(content, "My Custom AGENTS.md") {
		t.Error("existing content should be overwritten")
	}

	// Should have SMALL block
	if !strings.Contains(content, "<!-- BEGIN SMALL HARNESS") {
		t.Error("SMALL harness block should exist")
	}
}

func TestAgentsApplyDryRunDoesNotWrite(t *testing.T) {
	tmpDir := t.TempDir()
	oldBaseDir := baseDir
	baseDir = tmpDir
	defer func() {
		baseDir = oldBaseDir
	}()

	agentsPath := filepath.Join(tmpDir, "AGENTS.md")

	cmd := agentsApplyCmd()
	cmd.SetArgs([]string{"--dry-run"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("agents apply --dry-run failed: %v", err)
	}

	// File should not exist
	if _, err := os.Stat(agentsPath); err == nil {
		t.Error("AGENTS.md should not be created in dry-run mode")
	}
}

func TestAgentsApplyDryRunOnExisting(t *testing.T) {
	tmpDir := t.TempDir()
	oldBaseDir := baseDir
	baseDir = tmpDir
	defer func() {
		baseDir = oldBaseDir
	}()

	// Create existing AGENTS.md
	agentsPath := filepath.Join(tmpDir, "AGENTS.md")
	existingContent := "# My Custom AGENTS.md\n\nThese are my rules."
	if err := os.WriteFile(agentsPath, []byte(existingContent), 0644); err != nil {
		t.Fatalf("failed to write existing AGENTS.md: %v", err)
	}

	cmd := agentsApplyCmd()
	cmd.SetArgs([]string{"--dry-run", "--agents-mode=append"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("agents apply --dry-run failed: %v", err)
	}

	// File should be unchanged
	data, err := os.ReadFile(agentsPath)
	if err != nil {
		t.Fatalf("failed to read AGENTS.md: %v", err)
	}
	if string(data) != existingContent {
		t.Error("AGENTS.md should not be modified in dry-run mode")
	}
}

func TestAgentsApplyMalformedMarkers(t *testing.T) {
	tmpDir := t.TempDir()
	oldBaseDir := baseDir
	baseDir = tmpDir
	defer func() {
		baseDir = oldBaseDir
	}()

	// Create malformed AGENTS.md (BEGIN without END)
	agentsPath := filepath.Join(tmpDir, "AGENTS.md")
	malformedContent := `# My AGENTS.md

<!-- BEGIN SMALL HARNESS v1.0.0 -->
Content without end marker
`
	if err := os.WriteFile(agentsPath, []byte(malformedContent), 0644); err != nil {
		t.Fatalf("failed to write malformed AGENTS.md: %v", err)
	}

	cmd := agentsApplyCmd()
	cmd.SetArgs([]string{})
	err := cmd.Execute()

	if err == nil {
		t.Error("agents apply should fail on malformed markers")
	}

	if !strings.Contains(err.Error(), "malformed") {
		t.Errorf("error should mention malformed: %v", err)
	}
}

func TestAgentsApplyCustomFile(t *testing.T) {
	tmpDir := t.TempDir()
	oldBaseDir := baseDir
	baseDir = tmpDir
	defer func() {
		baseDir = oldBaseDir
	}()

	cmd := agentsApplyCmd()
	cmd.SetArgs([]string{"--file", "GOVERNANCE.md"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("agents apply failed: %v", err)
	}

	// GOVERNANCE.md should exist
	govPath := filepath.Join(tmpDir, "GOVERNANCE.md")
	data, err := os.ReadFile(govPath)
	if err != nil {
		t.Fatalf("failed to read GOVERNANCE.md: %v", err)
	}

	if !strings.Contains(string(data), "<!-- BEGIN SMALL HARNESS") {
		t.Error("GOVERNANCE.md should contain SMALL harness block")
	}

	// AGENTS.md should not exist
	agentsPath := filepath.Join(tmpDir, "AGENTS.md")
	if _, err := os.Stat(agentsPath); err == nil {
		t.Error("AGENTS.md should not be created when --file is specified")
	}
}

func TestAgentsApplyForceAliasOverwrite(t *testing.T) {
	tmpDir := t.TempDir()
	oldBaseDir := baseDir
	baseDir = tmpDir
	defer func() {
		baseDir = oldBaseDir
	}()

	// Create existing AGENTS.md without SMALL block
	agentsPath := filepath.Join(tmpDir, "AGENTS.md")
	existingContent := "# My Custom AGENTS.md\n\nThis has no SMALL block."
	if err := os.WriteFile(agentsPath, []byte(existingContent), 0644); err != nil {
		t.Fatalf("failed to write existing AGENTS.md: %v", err)
	}

	cmd := agentsApplyCmd()
	cmd.SetArgs([]string{"--force"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("agents apply --force failed: %v", err)
	}

	data, err := os.ReadFile(agentsPath)
	if err != nil {
		t.Fatalf("failed to read AGENTS.md: %v", err)
	}

	content := string(data)

	// Should NOT contain existing content (file was overwritten)
	if strings.Contains(content, "My Custom AGENTS.md") {
		t.Error("existing content should be overwritten with --force")
	}

	// Should have SMALL block
	if !strings.Contains(content, "<!-- BEGIN SMALL HARNESS") {
		t.Error("SMALL harness block should exist")
	}
}

func TestAgentsApplyForceShortFlag(t *testing.T) {
	tmpDir := t.TempDir()
	oldBaseDir := baseDir
	baseDir = tmpDir
	defer func() {
		baseDir = oldBaseDir
	}()

	// Create existing AGENTS.md without SMALL block
	agentsPath := filepath.Join(tmpDir, "AGENTS.md")
	existingContent := "# My Custom AGENTS.md\n\nThis has no SMALL block."
	if err := os.WriteFile(agentsPath, []byte(existingContent), 0644); err != nil {
		t.Fatalf("failed to write existing AGENTS.md: %v", err)
	}

	cmd := agentsApplyCmd()
	cmd.SetArgs([]string{"-f"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("agents apply -f failed: %v", err)
	}

	data, err := os.ReadFile(agentsPath)
	if err != nil {
		t.Fatalf("failed to read AGENTS.md: %v", err)
	}

	content := string(data)

	// Should NOT contain existing content (file was overwritten)
	if strings.Contains(content, "My Custom AGENTS.md") {
		t.Error("existing content should be overwritten with -f")
	}

	// Should have SMALL block
	if !strings.Contains(content, "<!-- BEGIN SMALL HARNESS") {
		t.Error("SMALL harness block should exist")
	}
}

func TestAgentsApplyForceIdenticalToOverwrite(t *testing.T) {
	// Test that --force and --overwrite-agents produce identical results
	tmpDir1 := t.TempDir()
	tmpDir2 := t.TempDir()

	existingContent := "# My Custom AGENTS.md\n\nThis has no SMALL block."
	agentsPath1 := filepath.Join(tmpDir1, "AGENTS.md")
	agentsPath2 := filepath.Join(tmpDir2, "AGENTS.md")

	if err := os.WriteFile(agentsPath1, []byte(existingContent), 0644); err != nil {
		t.Fatalf("failed to write AGENTS.md: %v", err)
	}
	if err := os.WriteFile(agentsPath2, []byte(existingContent), 0644); err != nil {
		t.Fatalf("failed to write AGENTS.md: %v", err)
	}

	// Apply with --force
	oldBaseDir := baseDir
	baseDir = tmpDir1
	cmd1 := agentsApplyCmd()
	cmd1.SetArgs([]string{"--force"})
	if err := cmd1.Execute(); err != nil {
		t.Fatalf("agents apply --force failed: %v", err)
	}

	// Apply with --overwrite-agents
	baseDir = tmpDir2
	cmd2 := agentsApplyCmd()
	cmd2.SetArgs([]string{"--overwrite-agents"})
	if err := cmd2.Execute(); err != nil {
		t.Fatalf("agents apply --overwrite-agents failed: %v", err)
	}
	baseDir = oldBaseDir

	// Results should be identical
	data1, _ := os.ReadFile(agentsPath1)
	data2, _ := os.ReadFile(agentsPath2)

	if string(data1) != string(data2) {
		t.Error("--force and --overwrite-agents should produce identical results")
	}
}

func TestAgentsApplyForceAndAgentsModeMutuallyExclusive(t *testing.T) {
	tmpDir := t.TempDir()
	oldBaseDir := baseDir
	baseDir = tmpDir
	defer func() {
		baseDir = oldBaseDir
	}()

	cmd := agentsApplyCmd()
	cmd.SetArgs([]string{"--force", "--agents-mode=append"})
	err := cmd.Execute()

	if err == nil {
		t.Error("--force and --agents-mode should be mutually exclusive")
	}
}

func TestAgentsApplyMutuallyExclusiveFlags(t *testing.T) {
	tmpDir := t.TempDir()
	oldBaseDir := baseDir
	baseDir = tmpDir
	defer func() {
		baseDir = oldBaseDir
	}()

	cmd := agentsApplyCmd()
	cmd.SetArgs([]string{"--agents-mode=append", "--overwrite-agents"})
	err := cmd.Execute()

	if err == nil {
		t.Error("--agents-mode and --overwrite-agents should be mutually exclusive")
	}
}

func TestAgentsApplyDoesNotTouchSmallDir(t *testing.T) {
	tmpDir := t.TempDir()
	oldBaseDir := baseDir
	baseDir = tmpDir
	defer func() {
		baseDir = oldBaseDir
	}()

	// Create existing .small/ directory
	smallDir := filepath.Join(tmpDir, ".small")
	if err := os.MkdirAll(smallDir, 0755); err != nil {
		t.Fatalf("failed to create .small directory: %v", err)
	}
	markerFile := filepath.Join(smallDir, "test-marker.txt")
	if err := os.WriteFile(markerFile, []byte("marker"), 0644); err != nil {
		t.Fatalf("failed to write marker file: %v", err)
	}

	cmd := agentsApplyCmd()
	cmd.SetArgs([]string{})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("agents apply failed: %v", err)
	}

	// .small/ should be unchanged
	if _, err := os.Stat(markerFile); err != nil {
		t.Error(".small/ directory should not be modified")
	}
}
