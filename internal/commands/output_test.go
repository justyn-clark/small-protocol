package commands

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/justyn-clark/small-protocol/internal/workspace"
)

func TestFormatErrorBlockAddsANSI(t *testing.T) {
	p := NewPrinter(io.Discard, io.Discard, true, false)
	block := p.FormatErrorBlock("Error", []string{"Line one"})
	if !strings.Contains(block, string(ansiBold)) {
		t.Fatalf("expected ANSI bold sequence in block, got %q", block)
	}
	if !strings.Contains(block, "  Line one") {
		t.Fatalf("expected indented line in block, got %q", block)
	}
}

func TestRootUnknownFlagUsesErrorBlock(t *testing.T) {
	oldArgs := os.Args
	oldPrinter := globalPrinter
	oldQuiet := outputQuiet
	oldNoColor := outputNoColor
	defer func() {
		os.Args = oldArgs
		globalPrinter = oldPrinter
		outputQuiet = oldQuiet
		outputNoColor = oldNoColor
	}()

	var errBuf bytes.Buffer
	globalPrinter = NewPrinter(io.Discard, &errBuf, false, false)
	outputQuiet = false
	outputNoColor = true
	os.Args = []string{"small", "--bogus-flag"}

	if err := Execute(); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	output := errBuf.String()
	if !strings.Contains(output, "Error") {
		t.Fatalf("expected error block title, got %q", output)
	}
	if !strings.Contains(output, "Usage:") {
		t.Fatalf("expected usage in error output, got %q", output)
	}
	if !strings.Contains(output, "unknown flag") {
		t.Fatalf("expected unknown flag error, got %q", output)
	}
}

func TestStrictS2FailureUsesFormattedBlock(t *testing.T) {
	oldPrinter := globalPrinter
	oldQuiet := outputQuiet
	oldNoColor := outputNoColor
	defer func() {
		globalPrinter = oldPrinter
		outputQuiet = oldQuiet
		outputNoColor = oldNoColor
	}()

	artifacts := cloneArtifacts(defaultArtifacts())
	artifacts["progress.small.yml"] = `small_version: "1.0.0"
owner: "agent"
entries:
  - task_id: "task-999"
    status: "completed"
    timestamp: "2025-01-01T00:00:00.000000000Z"
    evidence: "unknown task"
    replayId: "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2"
`

	tmpDir := t.TempDir()
	writeArtifacts(t, tmpDir, artifacts)
	mustSaveWorkspace(t, tmpDir, workspace.KindRepoRoot)

	var errBuf bytes.Buffer
	globalPrinter = NewPrinter(io.Discard, &errBuf, true, false)
	outputQuiet = false
	outputNoColor = false

	code, _, err := runCheck(tmpDir, true, false, false, workspace.ScopeRoot, false)
	if err != nil {
		t.Fatalf("runCheck error: %v", err)
	}
	if code != ExitInvalid {
		t.Fatalf("expected ExitInvalid, got %d", code)
	}

	output := errBuf.String()
	if !strings.Contains(output, "Strict S2 failed (current run only)") {
		t.Fatalf("expected strict S2 block, got %q", output)
	}
	if !strings.Contains(output, string(ansiBold)) {
		t.Fatalf("expected bold ANSI sequence in output, got %q", output)
	}
}
