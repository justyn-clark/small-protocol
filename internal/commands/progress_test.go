package commands

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/justyn-clark/small-protocol/internal/small"
	"github.com/justyn-clark/small-protocol/internal/workspace"
	"gopkg.in/yaml.v3"
)

func TestNormalizeProgressEntriesAddsFractional(t *testing.T) {
	entries := []map[string]any{
		{
			"task_id":   "task-1",
			"timestamp": "2025-01-01T00:00:00Z",
			"evidence":  "initial",
		},
	}

	changed, err := normalizeProgressEntries(entries)
	if err != nil {
		t.Fatalf("normalizeProgressEntries error: %v", err)
	}
	if changed != 1 {
		t.Fatalf("expected 1 change, got %d", changed)
	}

	timestamp := entries[0]["timestamp"].(string)
	if timestamp != "2025-01-01T00:00:00.000000000Z" {
		t.Fatalf("unexpected normalized timestamp %q", timestamp)
	}
}

func TestNormalizeProgressEntriesResolvesCollisions(t *testing.T) {
	entries := []map[string]any{
		{
			"task_id":   "task-1",
			"timestamp": "2025-01-01T00:00:00.000000000Z",
			"evidence":  "first",
		},
		{
			"task_id":   "task-2",
			"timestamp": "2025-01-01T00:00:00.000000000Z",
			"evidence":  "second",
		},
	}

	changed, err := normalizeProgressEntries(entries)
	if err != nil {
		t.Fatalf("normalizeProgressEntries error: %v", err)
	}
	if changed == 0 {
		t.Fatal("expected timestamp normalization changes")
	}

	firstTs := entries[0]["timestamp"].(string)
	secondTs := entries[1]["timestamp"].(string)
	firstParsed, _ := small.ParseProgressTimestamp(firstTs)
	secondParsed, _ := small.ParseProgressTimestamp(secondTs)
	if !secondParsed.After(firstParsed) {
		t.Fatalf("expected second timestamp after first: %s <= %s", secondTs, firstTs)
	}
}

func TestMigrateProgressFileRewritesTimestamps(t *testing.T) {
	oldNow := progressTimestampNow
	defer func() { progressTimestampNow = oldNow }()

	fixed := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	progressTimestampNow = func() time.Time { return fixed }
	tmpDir := t.TempDir()
	progressPath := filepath.Join(tmpDir, "progress.small.yml")

	progress := ProgressData{
		SmallVersion: small.ProtocolVersion,
		Owner:        "agent",
		Entries: []map[string]any{
			{
				"task_id":   "task-1",
				"timestamp": "2025-01-01T00:00:00Z",
				"evidence":  "first",
			},
			{
				"task_id":   "task-2",
				"timestamp": "2025-01-01T00:00:00Z",
				"evidence":  "second",
			},
		},
	}

	data, err := yaml.Marshal(&progress)
	if err != nil {
		t.Fatalf("failed to marshal progress: %v", err)
	}
	if err := os.WriteFile(progressPath, data, 0o644); err != nil {
		t.Fatalf("failed to write progress file: %v", err)
	}

	changed, err := migrateProgressFile(progressPath)
	if err != nil {
		t.Fatalf("migrateProgressFile error: %v", err)
	}
	if changed == 0 {
		t.Fatal("expected migrateProgressFile to rewrite timestamps")
	}

	updated, err := os.ReadFile(progressPath)
	if err != nil {
		t.Fatalf("failed to read updated progress file: %v", err)
	}

	var updatedProgress ProgressData
	if err := yaml.Unmarshal(updated, &updatedProgress); err != nil {
		t.Fatalf("failed to parse updated progress file: %v", err)
	}

	if len(updatedProgress.Entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(updatedProgress.Entries))
	}

	firstTs := updatedProgress.Entries[0]["timestamp"].(string)
	secondTs := updatedProgress.Entries[1]["timestamp"].(string)
	firstParsed, err := small.ParseProgressTimestamp(firstTs)
	if err != nil {
		t.Fatalf("invalid first timestamp %q: %v", firstTs, err)
	}
	secondParsed, err := small.ParseProgressTimestamp(secondTs)
	if err != nil {
		t.Fatalf("invalid second timestamp %q: %v", secondTs, err)
	}
	if !secondParsed.After(firstParsed) {
		t.Fatalf("expected timestamps to be strictly increasing: %s <= %s", secondTs, firstTs)
	}
}

func TestProgressAddMonotonicTimestamp(t *testing.T) {
	oldNow := progressTimestampNow
	defer func() { progressTimestampNow = oldNow }()

	fixed := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	progressTimestampNow = func() time.Time { return fixed }

	tmpDir := t.TempDir()
	smallDir := filepath.Join(tmpDir, ".small")
	if err := os.MkdirAll(smallDir, 0o755); err != nil {
		t.Fatalf("failed to create .small dir: %v", err)
	}
	mustSaveWorkspace(t, tmpDir, workspace.KindRepoRoot)

	progress := ProgressData{
		SmallVersion: small.ProtocolVersion,
		Owner:        "agent",
		Entries:      []map[string]any{},
	}
	data, err := yaml.Marshal(&progress)
	if err != nil {
		t.Fatalf("failed to marshal progress: %v", err)
	}
	progressPath := filepath.Join(smallDir, "progress.small.yml")
	if err := os.WriteFile(progressPath, data, 0o644); err != nil {
		t.Fatalf("failed to write progress file: %v", err)
	}

	entry := map[string]any{
		"task_id":   "task-1",
		"status":    "in_progress",
		"timestamp": formatProgressTimestamp(fixed),
	}
	if err := appendProgressEntryWithData(tmpDir, entry, progress); err != nil {
		t.Fatalf("appendProgressEntryWithData error: %v", err)
	}
	updatedProgress, err := loadProgressData(progressPath)
	if err != nil {
		t.Fatalf("loadProgressData error: %v", err)
	}
	second := map[string]any{
		"task_id":   "task-2",
		"status":    "in_progress",
		"timestamp": formatProgressTimestamp(fixed),
	}
	if err := appendProgressEntryWithData(tmpDir, second, updatedProgress); err != nil {
		t.Fatalf("appendProgressEntryWithData second error: %v", err)
	}

	updated, err := loadProgressData(progressPath)
	if err != nil {
		t.Fatalf("loadProgressData error: %v", err)
	}
	if len(updated.Entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(updated.Entries))
	}
	firstTs := updated.Entries[0]["timestamp"].(string)
	secondTs := updated.Entries[1]["timestamp"].(string)
	firstParsed, _ := small.ParseProgressTimestamp(firstTs)
	secondParsed, _ := small.ParseProgressTimestamp(secondTs)
	if !secondParsed.After(firstParsed) {
		t.Fatalf("expected monotonic timestamps: %s <= %s", secondTs, firstTs)
	}
}

func TestProgressAddCreatesFileIfMissing(t *testing.T) {
	oldNow := progressTimestampNow
	defer func() { progressTimestampNow = oldNow }()

	fixed := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	progressTimestampNow = func() time.Time { return fixed }
	tmpDir := t.TempDir()
	smallDir := filepath.Join(tmpDir, ".small")
	if err := os.MkdirAll(smallDir, 0o755); err != nil {
		t.Fatalf("failed to create .small dir: %v", err)
	}
	mustSaveWorkspace(t, tmpDir, workspace.KindRepoRoot)

	progressPath := filepath.Join(smallDir, "progress.small.yml")
	progress := ProgressData{SmallVersion: small.ProtocolVersion, Owner: "agent", Entries: []map[string]any{}}
	entry := map[string]any{
		"task_id":   "task-1",
		"status":    "pending",
		"notes":     "created via test",
		"timestamp": formatProgressTimestamp(fixed),
	}
	if err := appendProgressEntryWithData(tmpDir, entry, progress); err != nil {
		t.Fatalf("appendProgressEntryWithData error: %v", err)
	}
	if _, err := os.Stat(progressPath); err != nil {
		t.Fatalf("expected progress.small.yml to exist: %v", err)
	}
}

func TestMigrateProgressFileRejectsUnparseable(t *testing.T) {
	oldNow := progressTimestampNow
	defer func() { progressTimestampNow = oldNow }()

	fixed := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	progressTimestampNow = func() time.Time { return fixed }
	tmpDir := t.TempDir()
	progressPath := filepath.Join(tmpDir, "progress.small.yml")

	progress := ProgressData{
		SmallVersion: small.ProtocolVersion,
		Owner:        "agent",
		Entries: []map[string]any{
			{
				"task_id":   "task-1",
				"timestamp": "not-a-time",
				"evidence":  "bad",
			},
		},
	}

	data, err := yaml.Marshal(&progress)
	if err != nil {
		t.Fatalf("failed to marshal progress: %v", err)
	}
	if err := os.WriteFile(progressPath, data, 0o644); err != nil {
		t.Fatalf("failed to write progress file: %v", err)
	}

	if _, err := migrateProgressFile(progressPath); err == nil {
		t.Fatal("expected error for unparseable timestamp")
	}
}

func TestResolveProgressTimestampAtValidation(t *testing.T) {
	last := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	_, err := resolveProgressTimestamp(last, "2025-01-01T00:00:00Z", "")
	if err == nil {
		t.Fatal("expected error for non-RFC3339Nano timestamp")
	}

	_, err = resolveProgressTimestamp(last, formatProgressTimestamp(last), "")
	if err == nil {
		t.Fatal("expected error for non-monotonic --at timestamp")
	}

	at := formatProgressTimestamp(last.Add(time.Second))
	resolved, err := resolveProgressTimestamp(last, at, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resolved != at {
		t.Fatalf("expected resolved timestamp %q, got %q", at, resolved)
	}
}

func TestResolveProgressTimestampAfterGeneratesMonotonic(t *testing.T) {
	last := time.Date(2026, 1, 1, 0, 0, 0, 5, time.UTC)
	after := "2026-01-01T00:00:00.000000001Z"
	resolved, err := resolveProgressTimestamp(last, "", after)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	resolvedTime, err := small.ParseProgressTimestamp(resolved)
	if err != nil {
		t.Fatalf("resolved timestamp invalid: %v", err)
	}
	afterTime, err := small.ParseProgressTimestamp(after)
	if err != nil {
		t.Fatalf("after timestamp invalid: %v", err)
	}
	if !resolvedTime.After(afterTime) {
		t.Fatalf("expected resolved timestamp after --after value")
	}
	if !resolvedTime.After(last) {
		t.Fatalf("expected resolved timestamp after last entry")
	}
}

func TestAppendProgressUpdatesWorkspaceTimestamp(t *testing.T) {
	tmpDir := t.TempDir()
	smallDir := filepath.Join(tmpDir, ".small")
	if err := os.MkdirAll(smallDir, 0o755); err != nil {
		t.Fatalf("failed to create .small dir: %v", err)
	}
	mustSaveWorkspace(t, tmpDir, workspace.KindRepoRoot)

	progress := ProgressData{SmallVersion: small.ProtocolVersion, Owner: "agent", Entries: []map[string]any{}}
	data, err := yaml.Marshal(&progress)
	if err != nil {
		t.Fatalf("failed to marshal progress: %v", err)
	}
	progressPath := filepath.Join(smallDir, "progress.small.yml")
	if err := os.WriteFile(progressPath, data, 0o644); err != nil {
		t.Fatalf("failed to write progress: %v", err)
	}

	before, err := workspace.Load(tmpDir)
	if err != nil {
		t.Fatalf("failed to load workspace before: %v", err)
	}
	time.Sleep(time.Nanosecond)

	entry := map[string]any{
		"task_id":   "task-1",
		"status":    "in_progress",
		"timestamp": "2026-01-01T00:00:00.000000000Z",
		"evidence":  "started",
	}
	if err := appendProgressEntry(tmpDir, entry); err != nil {
		t.Fatalf("appendProgressEntry failed: %v", err)
	}

	after, err := workspace.Load(tmpDir)
	if err != nil {
		t.Fatalf("failed to load workspace after: %v", err)
	}
	if after.UpdatedAt == before.UpdatedAt {
		t.Fatalf("expected workspace updated_at to change after progress write")
	}
}

func TestAppendProgressReplayIDUsesWorkspaceAuthority(t *testing.T) {
	tmpDir := t.TempDir()
	smallDir := filepath.Join(tmpDir, ".small")
	if err := os.MkdirAll(smallDir, 0o755); err != nil {
		t.Fatalf("failed to create .small dir: %v", err)
	}
	mustSaveWorkspace(t, tmpDir, workspace.KindRepoRoot)

	progress := ProgressData{SmallVersion: small.ProtocolVersion, Owner: "agent", Entries: []map[string]any{}}
	data, err := yaml.Marshal(&progress)
	if err != nil {
		t.Fatalf("failed to marshal progress: %v", err)
	}
	progressPath := filepath.Join(smallDir, "progress.small.yml")
	if err := os.WriteFile(progressPath, data, 0o644); err != nil {
		t.Fatalf("failed to write progress: %v", err)
	}

	bootstrap := map[string]any{
		"task_id":   "meta/init",
		"status":    "completed",
		"timestamp": "2026-01-01T00:00:00.000000000Z",
		"evidence":  "bootstrap",
	}
	if err := appendProgressEntry(tmpDir, bootstrap); err != nil {
		t.Fatalf("append bootstrap entry failed: %v", err)
	}

	const replayID = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	if err := workspace.SetRunReplayID(tmpDir, replayID); err != nil {
		t.Fatalf("failed to set workspace replay id: %v", err)
	}

	taskEntry := map[string]any{
		"task_id":   "task-1",
		"status":    "completed",
		"timestamp": "2026-01-01T00:00:01.000000000Z",
		"evidence":  "run work",
	}
	if err := appendProgressEntry(tmpDir, taskEntry); err != nil {
		t.Fatalf("append task entry failed: %v", err)
	}

	updated, err := loadProgressData(progressPath)
	if err != nil {
		t.Fatalf("failed to load updated progress: %v", err)
	}
	if len(updated.Entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(updated.Entries))
	}
	if _, ok := updated.Entries[0]["replayId"]; ok {
		t.Fatalf("bootstrap entry should not include replayId: %+v", updated.Entries[0])
	}
	if got := stringVal(updated.Entries[1]["replayId"]); got != replayID {
		t.Fatalf("task entry replayId = %q, want %q", got, replayID)
	}
}
