package commands

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAgentsCheckMissingFile(t *testing.T) {
	tmpDir := t.TempDir()
	oldBaseDir := baseDir
	baseDir = tmpDir
	defer func() {
		baseDir = oldBaseDir
	}()

	cmd := agentsCheckCmd()
	cmd.SetArgs([]string{})
	err := cmd.Execute()

	if err == nil {
		t.Error("agents check should fail when file is missing")
	}
}

func TestAgentsCheckMissingFileAllowMissing(t *testing.T) {
	tmpDir := t.TempDir()
	oldBaseDir := baseDir
	baseDir = tmpDir
	defer func() {
		baseDir = oldBaseDir
	}()

	cmd := agentsCheckCmd()
	cmd.SetArgs([]string{"--allow-missing"})
	err := cmd.Execute()

	if err != nil {
		t.Errorf("agents check --allow-missing should not fail: %v", err)
	}
}

func TestAgentsCheckValidBlock(t *testing.T) {
	tmpDir := t.TempDir()
	oldBaseDir := baseDir
	baseDir = tmpDir
	defer func() {
		baseDir = oldBaseDir
	}()

	// Create valid AGENTS.md
	agentsPath := filepath.Join(tmpDir, "AGENTS.md")
	if err := os.WriteFile(agentsPath, []byte(GenerateAgentsBlock()), 0644); err != nil {
		t.Fatalf("failed to write AGENTS.md: %v", err)
	}

	cmd := agentsCheckCmd()
	cmd.SetArgs([]string{})
	err := cmd.Execute()

	if err != nil {
		t.Errorf("agents check should pass for valid block: %v", err)
	}
}

func TestAgentsCheckNoBlockNotStrict(t *testing.T) {
	tmpDir := t.TempDir()
	oldBaseDir := baseDir
	baseDir = tmpDir
	defer func() {
		baseDir = oldBaseDir
	}()

	// Create AGENTS.md without block
	agentsPath := filepath.Join(tmpDir, "AGENTS.md")
	if err := os.WriteFile(agentsPath, []byte("# My AGENTS.md\nNo block here."), 0644); err != nil {
		t.Fatalf("failed to write AGENTS.md: %v", err)
	}

	cmd := agentsCheckCmd()
	cmd.SetArgs([]string{})
	err := cmd.Execute()

	// Should pass without --strict
	if err != nil {
		t.Errorf("agents check should pass without --strict when no block: %v", err)
	}
}

func TestAgentsCheckNoBlockStrict(t *testing.T) {
	tmpDir := t.TempDir()
	oldBaseDir := baseDir
	baseDir = tmpDir
	defer func() {
		baseDir = oldBaseDir
	}()

	// Create AGENTS.md without block
	agentsPath := filepath.Join(tmpDir, "AGENTS.md")
	if err := os.WriteFile(agentsPath, []byte("# My AGENTS.md\nNo block here."), 0644); err != nil {
		t.Fatalf("failed to write AGENTS.md: %v", err)
	}

	cmd := agentsCheckCmd()
	cmd.SetArgs([]string{"--strict"})
	err := cmd.Execute()

	// Should fail with --strict
	if err == nil {
		t.Error("agents check --strict should fail when no block exists")
	}
}

func TestAgentsCheckMalformedMarkers(t *testing.T) {
	tmpDir := t.TempDir()
	oldBaseDir := baseDir
	baseDir = tmpDir
	defer func() {
		baseDir = oldBaseDir
	}()

	// Create malformed AGENTS.md
	agentsPath := filepath.Join(tmpDir, "AGENTS.md")
	malformedContent := `<!-- BEGIN SMALL HARNESS v1.0.0 -->
Content without end marker
`
	if err := os.WriteFile(agentsPath, []byte(malformedContent), 0644); err != nil {
		t.Fatalf("failed to write AGENTS.md: %v", err)
	}

	cmd := agentsCheckCmd()
	cmd.SetArgs([]string{})
	err := cmd.Execute()

	if err == nil {
		t.Error("agents check should fail on malformed markers")
	}
}

func TestAgentsCheckJSONFormat(t *testing.T) {
	tmpDir := t.TempDir()
	oldBaseDir := baseDir
	baseDir = tmpDir
	defer func() {
		baseDir = oldBaseDir
	}()

	// Create valid AGENTS.md
	agentsPath := filepath.Join(tmpDir, "AGENTS.md")
	if err := os.WriteFile(agentsPath, []byte(GenerateAgentsBlock()), 0644); err != nil {
		t.Fatalf("failed to write AGENTS.md: %v", err)
	}

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	cmd := agentsCheckCmd()
	cmd.SetArgs([]string{"--format", "json"})
	err := cmd.Execute()

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Errorf("agents check should pass: %v", err)
	}

	// Read captured output
	buf := make([]byte, 4096)
	n, _ := r.Read(buf)
	output := string(buf[:n])

	// Parse JSON
	var result AgentsCheckResult
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("failed to parse JSON output: %v\nOutput: %s", err, output)
	}

	if !result.HasFile {
		t.Error("JSON result should have hasFile=true")
	}
	if !result.HasBlock {
		t.Error("JSON result should have hasBlock=true")
	}
	if !result.BlockValid {
		t.Error("JSON result should have blockValid=true")
	}
}

func TestAgentsCheckJSONFormatMissing(t *testing.T) {
	tmpDir := t.TempDir()
	oldBaseDir := baseDir
	baseDir = tmpDir
	defer func() {
		baseDir = oldBaseDir
	}()

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	cmd := agentsCheckCmd()
	cmd.SetArgs([]string{"--format", "json", "--allow-missing"})
	_ = cmd.Execute()

	w.Close()
	os.Stdout = oldStdout

	// Read captured output
	buf := make([]byte, 4096)
	n, _ := r.Read(buf)
	output := string(buf[:n])

	// Parse JSON
	var result AgentsCheckResult
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("failed to parse JSON output: %v\nOutput: %s", err, output)
	}

	if result.HasFile {
		t.Error("JSON result should have hasFile=false")
	}
}

func TestAgentsCheckDriftDetection(t *testing.T) {
	tmpDir := t.TempDir()
	oldBaseDir := baseDir
	baseDir = tmpDir
	defer func() {
		baseDir = oldBaseDir
	}()

	// Create AGENTS.md with modified block content
	agentsPath := filepath.Join(tmpDir, "AGENTS.md")
	modifiedContent := `<!-- BEGIN SMALL HARNESS v1.0.0 -->
# SMALL Execution Harness

Modified content that differs from canonical template.

## Ownership Rules
Modified ownership rules.

## Artifact Rules
Modified artifact rules.

## Strict Mode Rules
Modified strict rules.
<!-- END SMALL HARNESS v1.0.0 -->
`
	if err := os.WriteFile(agentsPath, []byte(modifiedContent), 0644); err != nil {
		t.Fatalf("failed to write AGENTS.md: %v", err)
	}

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	cmd := agentsCheckCmd()
	cmd.SetArgs([]string{"--format", "json"})
	_ = cmd.Execute()

	w.Close()
	os.Stdout = oldStdout

	// Read captured output
	buf := make([]byte, 4096)
	n, _ := r.Read(buf)
	output := string(buf[:n])

	// Parse JSON
	var result AgentsCheckResult
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("failed to parse JSON output: %v\nOutput: %s", err, output)
	}

	if !result.HasDrift {
		t.Error("JSON result should detect drift")
	}
}

func TestAgentsCheckDriftStrictMode(t *testing.T) {
	tmpDir := t.TempDir()
	oldBaseDir := baseDir
	baseDir = tmpDir
	defer func() {
		baseDir = oldBaseDir
	}()

	// Create AGENTS.md with modified block content
	agentsPath := filepath.Join(tmpDir, "AGENTS.md")
	modifiedContent := `<!-- BEGIN SMALL HARNESS v1.0.0 -->
# SMALL Execution Harness

Modified content that differs from canonical template.

## Ownership Rules
Modified ownership rules.

## Artifact Rules
Modified artifact rules.

## Strict Mode Rules
Modified strict rules.
<!-- END SMALL HARNESS v1.0.0 -->
`
	if err := os.WriteFile(agentsPath, []byte(modifiedContent), 0644); err != nil {
		t.Fatalf("failed to write AGENTS.md: %v", err)
	}

	cmd := agentsCheckCmd()
	cmd.SetArgs([]string{"--strict"})
	err := cmd.Execute()

	// Should fail in strict mode due to drift
	if err == nil {
		t.Error("agents check --strict should fail when drift is detected")
	}
}

func TestAgentsCheckCustomFile(t *testing.T) {
	tmpDir := t.TempDir()
	oldBaseDir := baseDir
	baseDir = tmpDir
	defer func() {
		baseDir = oldBaseDir
	}()

	// Create GOVERNANCE.md with valid block
	govPath := filepath.Join(tmpDir, "GOVERNANCE.md")
	if err := os.WriteFile(govPath, []byte(GenerateAgentsBlock()), 0644); err != nil {
		t.Fatalf("failed to write GOVERNANCE.md: %v", err)
	}

	cmd := agentsCheckCmd()
	cmd.SetArgs([]string{"--file", "GOVERNANCE.md"})
	err := cmd.Execute()

	if err != nil {
		t.Errorf("agents check should pass for valid GOVERNANCE.md: %v", err)
	}
}

func TestAgentsCheckDoesNotModifyFiles(t *testing.T) {
	tmpDir := t.TempDir()
	oldBaseDir := baseDir
	baseDir = tmpDir
	defer func() {
		baseDir = oldBaseDir
	}()

	// Create AGENTS.md
	agentsPath := filepath.Join(tmpDir, "AGENTS.md")
	originalContent := GenerateAgentsBlock()
	if err := os.WriteFile(agentsPath, []byte(originalContent), 0644); err != nil {
		t.Fatalf("failed to write AGENTS.md: %v", err)
	}

	// Get original modification time
	origInfo, _ := os.Stat(agentsPath)
	origModTime := origInfo.ModTime()

	cmd := agentsCheckCmd()
	cmd.SetArgs([]string{})
	_ = cmd.Execute()

	// Check file wasn't modified
	newInfo, _ := os.Stat(agentsPath)
	if !newInfo.ModTime().Equal(origModTime) {
		t.Error("agents check should not modify the file")
	}

	// Check content unchanged
	data, _ := os.ReadFile(agentsPath)
	if string(data) != originalContent {
		t.Error("agents check should not change file contents")
	}
}

func TestAgentsCheckBlockSpanInJSON(t *testing.T) {
	tmpDir := t.TempDir()
	oldBaseDir := baseDir
	baseDir = tmpDir
	defer func() {
		baseDir = oldBaseDir
	}()

	// Create AGENTS.md with content before and after block
	agentsPath := filepath.Join(tmpDir, "AGENTS.md")
	content := "# Header\n\n" + GenerateAgentsBlock() + "\n# Footer\n"
	if err := os.WriteFile(agentsPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write AGENTS.md: %v", err)
	}

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	cmd := agentsCheckCmd()
	cmd.SetArgs([]string{"--format", "json"})
	_ = cmd.Execute()

	w.Close()
	os.Stdout = oldStdout

	// Read captured output
	buf := make([]byte, 8192)
	n, _ := r.Read(buf)
	output := string(buf[:n])

	// Parse JSON
	var result AgentsCheckResult
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("failed to parse JSON output: %v\nOutput: %s", err, output)
	}

	// BlockSpan should be set
	if result.BlockSpan[0] == 0 && result.BlockSpan[1] == 0 {
		t.Error("BlockSpan should be set for found block")
	}

	// Start should be after "# Header\n\n"
	expectedStart := strings.Index(content, "<!-- BEGIN SMALL HARNESS")
	if result.BlockSpan[0] != expectedStart {
		t.Errorf("BlockSpan start should be %d, got %d", expectedStart, result.BlockSpan[0])
	}
}
