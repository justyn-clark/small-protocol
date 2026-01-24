package small

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestWriteCommandLog(t *testing.T) {
	baseDir := t.TempDir()
	replayId := strings.Repeat("a", 64)
	timestamp := time.Date(2026, 1, 22, 10, 40, 42, 806397000, time.UTC).Format("2006-01-02T15:04:05.000000000Z")
	command := "echo hello\necho world"

	ref, sha, err := WriteCommandLog(baseDir, replayId, timestamp, command)
	if err != nil {
		t.Fatalf("WriteCommandLog error: %v", err)
	}

	sanitized, err := SanitizeTimestampForFilename(timestamp)
	if err != nil {
		t.Fatalf("SanitizeTimestampForFilename error: %v", err)
	}

	expectedRel := filepath.ToSlash(filepath.Join(SmallDir, "logs", replayId, "commands", sanitized+".txt"))
	if ref != expectedRel {
		t.Fatalf("expected ref %q, got %q", expectedRel, ref)
	}

	expectedPath := filepath.Join(baseDir, SmallDir, "logs", replayId, "commands", sanitized+".txt")
	content, err := os.ReadFile(expectedPath)
	if err != nil {
		t.Fatalf("expected log file read error: %v", err)
	}
	if string(content) != command {
		t.Fatalf("expected command log content to match")
	}

	hash := sha256.Sum256([]byte(command))
	expectedSha := hex.EncodeToString(hash[:])
	if sha != expectedSha {
		t.Fatalf("expected sha %q, got %q", expectedSha, sha)
	}
}

func TestSummarizeCommandCap(t *testing.T) {
	long := strings.Repeat("a", DefaultCommandSummaryCap+25)
	summary := SummarizeCommand(long, DefaultCommandSummaryCap)
	if len(summary) != DefaultCommandSummaryCap {
		t.Fatalf("expected summary length %d, got %d", DefaultCommandSummaryCap, len(summary))
	}
	if !strings.HasSuffix(summary, "...") {
		t.Fatalf("expected summary to end with ...")
	}

	multi := "echo one\n\t echo two  \n  echo three"
	normalized := SummarizeCommand(multi, DefaultCommandSummaryCap)
	if strings.Contains(normalized, "\n") || strings.Contains(normalized, "\t") {
		t.Fatalf("expected summary to normalize whitespace")
	}
}
