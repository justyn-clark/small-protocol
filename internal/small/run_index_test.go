package small

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"gopkg.in/yaml.v3"
)

func TestAppendRunIndexEntryAppends(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmpDir, SmallDir), 0755); err != nil {
		t.Fatalf("failed to create .small: %v", err)
	}

	firstReplay := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	secondReplay := "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"

	entry1 := RunIndexEntry{
		ReplayID:  firstReplay,
		Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
		Reason:    "snapshot",
		Summary:   "first",
	}
	if err := AppendRunIndexEntry(tmpDir, entry1); err != nil {
		t.Fatalf("failed to append run index entry: %v", err)
	}

	index := readRunIndex(t, tmpDir)
	if len(index.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(index.Entries))
	}
	if index.Entries[0].ReplayID != firstReplay {
		t.Fatalf("expected first replayId %q, got %q", firstReplay, index.Entries[0].ReplayID)
	}

	entry2 := RunIndexEntry{
		ReplayID:  secondReplay,
		Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
		Reason:    "archive",
		Summary:   "second",
	}
	if err := AppendRunIndexEntry(tmpDir, entry2); err != nil {
		t.Fatalf("failed to append second run index entry: %v", err)
	}

	index = readRunIndex(t, tmpDir)
	if len(index.Entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(index.Entries))
	}
	if index.Entries[0].ReplayID != firstReplay {
		t.Fatalf("expected first entry to remain unchanged")
	}
	if index.Entries[1].ReplayID != secondReplay {
		t.Fatalf("expected second entry replayId %q, got %q", secondReplay, index.Entries[1].ReplayID)
	}
	if index.Entries[1].PreviousReplayID != firstReplay {
		t.Fatalf("expected previous_replay_id to be %q, got %q", firstReplay, index.Entries[1].PreviousReplayID)
	}
}

func readRunIndex(t *testing.T, baseDir string) RunIndex {
	t.Helper()
	path := filepath.Join(baseDir, SmallDir, "runs", "index.small.yml")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read run index: %v", err)
	}
	var index RunIndex
	if err := yaml.Unmarshal(data, &index); err != nil {
		t.Fatalf("failed to parse run index: %v", err)
	}
	return index
}
