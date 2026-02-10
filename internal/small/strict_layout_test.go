package small

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStrictSmallLayoutViolations(t *testing.T) {
	baseDir := t.TempDir()
	smallDir := filepath.Join(baseDir, SmallDir)
	if err := os.MkdirAll(smallDir, 0o755); err != nil {
		t.Fatalf("failed to create .small: %v", err)
	}

	allowed := []string{
		"intent.small.yml",
		"constraints.small.yml",
		"plan.small.yml",
		"progress.small.yml",
		"handoff.small.yml",
		"workspace.small.yml",
	}
	for _, filename := range allowed {
		if err := os.WriteFile(filepath.Join(smallDir, filename), []byte("small_version: \"1.0.0\"\nowner: \"agent\"\n"), 0o644); err != nil {
			t.Fatalf("failed to write %s: %v", filename, err)
		}
	}

	violations, err := StrictSmallLayoutViolations(baseDir, "small check --strict")
	if err != nil {
		t.Fatalf("StrictSmallLayoutViolations error: %v", err)
	}
	if len(violations) != 0 {
		t.Fatalf("expected no violations, got %d", len(violations))
	}

	if err := os.WriteFile(filepath.Join(smallDir, "notes.txt"), []byte("x"), 0o644); err != nil {
		t.Fatalf("failed to write rogue file: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(smallDir, "logs"), 0o755); err != nil {
		t.Fatalf("failed to create logs dir: %v", err)
	}

	violations, err = StrictSmallLayoutViolations(baseDir, "small check --strict")
	if err != nil {
		t.Fatalf("StrictSmallLayoutViolations error: %v", err)
	}
	if len(violations) != 1 {
		t.Fatalf("expected 1 violation, got %d", len(violations))
	}

	message := violations[0].Message
	for _, expected := range []string{
		"strict invariant S4 failed",
		".small/logs/",
		".small/notes.txt",
		"Delete or move",
		"small check --strict",
	} {
		if !strings.Contains(message, expected) {
			t.Fatalf("expected message to contain %q, got %q", expected, message)
		}
	}
}
