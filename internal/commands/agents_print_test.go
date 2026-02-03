package commands

import (
	"os"
	"strings"
	"testing"
)

func TestAgentsPrintWithMarkers(t *testing.T) {
	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	cmd := agentsPrintCmd()
	cmd.SetArgs([]string{})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("agents print failed: %v", err)
	}

	w.Close()
	os.Stdout = oldStdout

	// Read captured output
	buf := make([]byte, 8192)
	n, _ := r.Read(buf)
	output := string(buf[:n])

	// Should contain BEGIN marker
	if !strings.Contains(output, "<!-- BEGIN SMALL HARNESS") {
		t.Error("output should contain BEGIN marker")
	}

	// Should contain END marker
	if !strings.Contains(output, "<!-- END SMALL HARNESS") {
		t.Error("output should contain END marker")
	}

	// Should contain harness content
	if !strings.Contains(output, "SMALL Execution Harness") {
		t.Error("output should contain harness content")
	}
}

func TestAgentsPrintWithoutMarkers(t *testing.T) {
	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	cmd := agentsPrintCmd()
	cmd.SetArgs([]string{"--with-markers=false"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("agents print failed: %v", err)
	}

	w.Close()
	os.Stdout = oldStdout

	// Read captured output
	buf := make([]byte, 8192)
	n, _ := r.Read(buf)
	output := string(buf[:n])

	// Should NOT contain BEGIN marker
	if strings.Contains(output, "<!-- BEGIN SMALL HARNESS") {
		t.Error("output should not contain BEGIN marker")
	}

	// Should NOT contain END marker
	if strings.Contains(output, "<!-- END SMALL HARNESS") {
		t.Error("output should not contain END marker")
	}

	// Should still contain harness content
	if !strings.Contains(output, "SMALL Execution Harness") {
		t.Error("output should contain harness content")
	}
}

func TestAgentsPrintRequiredSections(t *testing.T) {
	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	cmd := agentsPrintCmd()
	cmd.SetArgs([]string{})
	_ = cmd.Execute()

	w.Close()
	os.Stdout = oldStdout

	// Read captured output
	buf := make([]byte, 8192)
	n, _ := r.Read(buf)
	output := string(buf[:n])

	requiredSections := []string{
		"Ownership Rules",
		"Artifact Rules",
		"Strict Mode Rules",
		"Localhost HTTP Allowlist",
		"Ralph Loop",
	}

	for _, section := range requiredSections {
		if !strings.Contains(output, section) {
			t.Errorf("output should contain section: %s", section)
		}
	}
}

func TestAgentsPrintPlainFormat(t *testing.T) {
	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	cmd := agentsPrintCmd()
	cmd.SetArgs([]string{"--with-markers=false", "--format=plain"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("agents print failed: %v", err)
	}

	w.Close()
	os.Stdout = oldStdout

	// Read captured output
	buf := make([]byte, 8192)
	n, _ := r.Read(buf)
	output := string(buf[:n])

	// Plain format should strip markdown formatting
	// Check that headings don't have ## prefix
	if strings.Contains(output, "## Ownership") {
		t.Error("plain format should strip heading markers")
	}

	// Should still contain content
	if !strings.Contains(output, "Ownership Rules") {
		t.Error("output should contain content even in plain format")
	}
}

func TestAgentsPrintDoesNotCreateFiles(t *testing.T) {
	tmpDir := t.TempDir()
	oldBaseDir := baseDir
	baseDir = tmpDir
	defer func() {
		baseDir = oldBaseDir
	}()

	// Capture stdout to prevent test output pollution
	oldStdout := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w

	cmd := agentsPrintCmd()
	cmd.SetArgs([]string{})
	_ = cmd.Execute()

	w.Close()
	os.Stdout = oldStdout

	// No files should be created
	entries, err := os.ReadDir(tmpDir)
	if err != nil {
		t.Fatalf("failed to read temp dir: %v", err)
	}

	if len(entries) > 0 {
		t.Errorf("agents print should not create any files, found: %v", entries)
	}
}

func TestAgentsPrintMatchesGenerateAgentsBlock(t *testing.T) {
	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	cmd := agentsPrintCmd()
	cmd.SetArgs([]string{"--with-markers=true"})
	_ = cmd.Execute()

	w.Close()
	os.Stdout = oldStdout

	// Read captured output
	buf := make([]byte, 8192)
	n, _ := r.Read(buf)
	output := string(buf[:n])

	// Should match GenerateAgentsBlock()
	expected := GenerateAgentsBlock()
	if output != expected {
		t.Errorf("print output should match GenerateAgentsBlock()\nGot: %s\nExpected: %s", output, expected)
	}
}
