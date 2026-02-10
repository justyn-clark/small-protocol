package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/justyn-clark/small-protocol/internal/small"
	"github.com/justyn-clark/small-protocol/internal/workspace"
	"gopkg.in/yaml.v3"
)

func TestNormalizeTaskID(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"", "apply"},
		{"task-1", "task-1"},
		{"custom-id", "custom-id"},
		{"task-99", "task-99"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := normalizeTaskID(tt.input)
			if result != tt.expected {
				t.Errorf("normalizeTaskID(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestAppendProgressEntry(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "small-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	smallDir := filepath.Join(tmpDir, ".small")
	if err := os.MkdirAll(smallDir, 0755); err != nil {
		t.Fatalf("failed to create .small dir: %v", err)
	}

	initialProgress := ProgressData{
		SmallVersion: small.ProtocolVersion,
		Owner:        "agent",
		Entries:      []map[string]any{},
	}

	data, err := yaml.Marshal(&initialProgress)
	if err != nil {
		t.Fatalf("failed to marshal initial progress: %v", err)
	}

	progressPath := filepath.Join(smallDir, "progress.small.yml")
	if err := os.WriteFile(progressPath, data, 0644); err != nil {
		t.Fatalf("failed to write initial progress: %v", err)
	}

	entry := map[string]any{
		"timestamp": "2024-01-15T09:00:00.000000000Z",
		"task_id":   "task-1",
		"status":    "completed",
		"evidence":  "Test evidence",
		"notes":     "Test notes",
	}

	if err := appendProgressEntry(tmpDir, entry); err != nil {
		t.Fatalf("appendProgressEntry() error = %v", err)
	}

	updatedData, err := os.ReadFile(progressPath)
	if err != nil {
		t.Fatalf("failed to read updated progress: %v", err)
	}

	var updatedProgress ProgressData
	if err := yaml.Unmarshal(updatedData, &updatedProgress); err != nil {
		t.Fatalf("failed to unmarshal updated progress: %v", err)
	}

	if len(updatedProgress.Entries) != 1 {
		t.Errorf("expected 1 entry, got %d", len(updatedProgress.Entries))
	}

	if updatedProgress.Entries[0]["task_id"] != "task-1" {
		t.Errorf("expected task_id 'task-1', got %v", updatedProgress.Entries[0]["task_id"])
	}

	firstTimestamp, _ := updatedProgress.Entries[0]["timestamp"].(string)
	if _, err := small.ParseProgressTimestamp(firstTimestamp); err != nil {
		t.Fatalf("expected valid RFC3339Nano timestamp, got %q (%v)", firstTimestamp, err)
	}

	entry2 := map[string]any{
		"timestamp": "2024-01-15T10:00:00.000000000Z",
		"task_id":   "task-2",
		"status":    "in_progress",
		"evidence":  "Second entry",
	}

	if err := appendProgressEntry(tmpDir, entry2); err != nil {
		t.Fatalf("appendProgressEntry() second entry error = %v", err)
	}

	updatedData, err = os.ReadFile(progressPath)
	if err != nil {
		t.Fatalf("failed to read progress after second append: %v", err)
	}

	if err := yaml.Unmarshal(updatedData, &updatedProgress); err != nil {
		t.Fatalf("failed to unmarshal progress after second append: %v", err)
	}

	if len(updatedProgress.Entries) != 2 {
		t.Errorf("expected 2 entries, got %d", len(updatedProgress.Entries))
	}

	secondTimestamp, _ := updatedProgress.Entries[1]["timestamp"].(string)
	secondParsed, err := small.ParseProgressTimestamp(secondTimestamp)
	if err != nil {
		t.Fatalf("expected valid timestamp for second entry, got %q (%v)", secondTimestamp, err)
	}

	firstParsed, _ := small.ParseProgressTimestamp(firstTimestamp)
	if !secondParsed.After(firstParsed) {
		t.Fatalf("expected second timestamp after first: %s <= %s", secondTimestamp, firstTimestamp)
	}
}

func TestAppendProgressEntryMissingFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "small-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	smallDir := filepath.Join(tmpDir, ".small")
	if err := os.MkdirAll(smallDir, 0755); err != nil {
		t.Fatalf("failed to create .small dir: %v", err)
	}

	entry := map[string]any{
		"timestamp": "2024-01-15T09:00:00.000000000Z",
		"task_id":   "task-1",
		"status":    "completed",
	}

	err = appendProgressEntry(tmpDir, entry)
	if err == nil {
		t.Error("appendProgressEntry() should error when progress file is missing")
	}
}

func TestProgressEntryStatusValues(t *testing.T) {
	validStatuses := []string{"pending", "in_progress", "completed", "blocked"}
	applyStatuses := []string{"pending", "in_progress", "completed", "blocked"}

	for _, status := range applyStatuses {
		found := false
		for _, valid := range validStatuses {
			if status == valid {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("apply uses invalid status %q", status)
		}
	}
}

func TestApplyCommandMetadataLogging(t *testing.T) {
	tmpDir := t.TempDir()
	smallDir := filepath.Join(tmpDir, ".small")
	if err := os.MkdirAll(smallDir, 0o755); err != nil {
		t.Fatalf("failed to create .small dir: %v", err)
	}

	replayId := strings.Repeat("a", 64)
	handoff := fmt.Sprintf(`small_version: %q
owner: %q
summary: %q
resume:
  current_task_id: %q
  next_steps: []
links: []
replayId:
  value: %q
  source: %q
`,
		small.ProtocolVersion,
		"agent",
		"test",
		"",
		replayId,
		"test",
	)

	if err := os.WriteFile(filepath.Join(smallDir, "handoff.small.yml"), []byte(handoff), 0o644); err != nil {
		t.Fatalf("failed to write handoff: %v", err)
	}

	progress := ProgressData{
		SmallVersion: small.ProtocolVersion,
		Owner:        "agent",
		Entries:      []map[string]any{},
	}
	progressBytes, err := yaml.Marshal(&progress)
	if err != nil {
		t.Fatalf("failed to marshal progress: %v", err)
	}
	if err := os.WriteFile(filepath.Join(smallDir, "progress.small.yml"), progressBytes, 0o644); err != nil {
		t.Fatalf("failed to write progress: %v", err)
	}

	command := "echo " + strings.Repeat("x", 300)
	timestamp := formatProgressTimestamp(time.Date(2026, 1, 22, 10, 40, 42, 0, time.UTC))
	summary, ref, sha, err := applyCommandMetadata(tmpDir, timestamp, command)
	if err != nil {
		t.Fatalf("applyCommandMetadata error: %v", err)
	}
	entry := map[string]any{
		"timestamp":       timestamp,
		"task_id":         "task-1",
		"status":          "in_progress",
		"evidence":        "Apply started",
		"command":         summary,
		"command_summary": summary,
		"command_ref":     ref,
		"command_sha256":  sha,
	}
	if err := appendProgressEntry(tmpDir, entry); err != nil {
		t.Fatalf("appendProgressEntry error: %v", err)
	}

	created, err := loadProgressData(filepath.Join(smallDir, "progress.small.yml"))
	if err != nil {
		t.Fatalf("failed to load progress: %v", err)
	}
	if len(created.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(created.Entries))
	}
	stored := created.Entries[0]
	if stored["command_summary"] == nil || stored["command_ref"] == nil || stored["command_sha256"] == nil {
		t.Fatalf("expected command fields to be set")
	}

	logPath := filepath.Join(tmpDir, filepath.FromSlash(ref))
	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("expected command log to exist: %v", err)
	}
	if string(content) != command {
		t.Fatalf("expected command log content to match")
	}

	if len(summary) > small.DefaultCommandSummaryCap {
		t.Fatalf("expected command summary to be truncated")
	}
}

func TestApplyProgressSignalModeDefault(t *testing.T) {
	t.Setenv(progressModeEnvVar, "")

	tmpDir := t.TempDir()
	writeArtifacts(t, tmpDir, defaultArtifacts())
	mustSaveWorkspace(t, tmpDir, workspace.KindRepoRoot)

	cmd := applyCmd()
	cmd.SetArgs([]string{"--dir", tmpDir, "--workspace", "any", "--task", "task-1", "--cmd", "true"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("apply execute failed: %v", err)
	}

	progress, err := loadProgressData(filepath.Join(tmpDir, ".small", "progress.small.yml"))
	if err != nil {
		t.Fatalf("failed to load progress: %v", err)
	}
	if len(progress.Entries) != 1 {
		t.Fatalf("expected 1 signal progress entry, got %d", len(progress.Entries))
	}
	entry := progress.Entries[0]
	if stringVal(entry["status"]) != "completed" {
		t.Fatalf("status = %q, want completed", stringVal(entry["status"]))
	}
	if stringVal(entry["notes"]) == "apply: execution started" {
		t.Fatal("did not expect start telemetry in signal mode")
	}
}

func TestApplyProgressAuditModeEmitsVerboseEntries(t *testing.T) {
	t.Setenv(progressModeEnvVar, string(progressModeAudit))

	tmpDir := t.TempDir()
	writeArtifacts(t, tmpDir, defaultArtifacts())
	mustSaveWorkspace(t, tmpDir, workspace.KindRepoRoot)

	cmd := applyCmd()
	cmd.SetArgs([]string{"--dir", tmpDir, "--workspace", "any", "--task", "task-1", "--cmd", "true"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("apply execute failed: %v", err)
	}

	progress, err := loadProgressData(filepath.Join(tmpDir, ".small", "progress.small.yml"))
	if err != nil {
		t.Fatalf("failed to load progress: %v", err)
	}
	if len(progress.Entries) != 2 {
		t.Fatalf("expected 2 audit progress entries, got %d", len(progress.Entries))
	}
	if stringVal(progress.Entries[0]["status"]) != "in_progress" {
		t.Fatalf("first status = %q, want in_progress", stringVal(progress.Entries[0]["status"]))
	}
	if stringVal(progress.Entries[1]["status"]) != "completed" {
		t.Fatalf("second status = %q, want completed", stringVal(progress.Entries[1]["status"]))
	}
}

func TestApplyProgressSignalModeSkipsNonTaskEntries(t *testing.T) {
	t.Setenv(progressModeEnvVar, string(progressModeSignal))

	tmpDir := t.TempDir()
	writeArtifacts(t, tmpDir, defaultArtifacts())
	mustSaveWorkspace(t, tmpDir, workspace.KindRepoRoot)

	cmd := applyCmd()
	cmd.SetArgs([]string{"--dir", tmpDir, "--workspace", "any", "--cmd", "true"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("apply execute failed: %v", err)
	}

	progress, err := loadProgressData(filepath.Join(tmpDir, ".small", "progress.small.yml"))
	if err != nil {
		t.Fatalf("failed to load progress: %v", err)
	}
	if len(progress.Entries) != 0 {
		t.Fatalf("expected no progress entries for non-task apply in signal mode, got %d", len(progress.Entries))
	}
}

func TestApplyProgressSignalModeDeterministicShape(t *testing.T) {
	t.Setenv(progressModeEnvVar, string(progressModeSignal))

	tmpDir := t.TempDir()
	writeArtifacts(t, tmpDir, defaultArtifacts())
	mustSaveWorkspace(t, tmpDir, workspace.KindRepoRoot)

	for i := 0; i < 2; i++ {
		cmd := applyCmd()
		cmd.SetArgs([]string{"--dir", tmpDir, "--workspace", "any", "--task", "task-1", "--cmd", "true"})
		if err := cmd.Execute(); err != nil {
			t.Fatalf("apply execute failed on run %d: %v", i+1, err)
		}
	}

	progress, err := loadProgressData(filepath.Join(tmpDir, ".small", "progress.small.yml"))
	if err != nil {
		t.Fatalf("failed to load progress: %v", err)
	}
	if len(progress.Entries) != 2 {
		t.Fatalf("expected 2 completion entries, got %d", len(progress.Entries))
	}
	for i, entry := range progress.Entries {
		if stringVal(entry["status"]) != "completed" {
			t.Fatalf("entry %d status = %q, want completed", i, stringVal(entry["status"]))
		}
		if stringVal(entry["notes"]) == "apply: execution started" {
			t.Fatalf("entry %d unexpectedly includes start telemetry", i)
		}
	}
}
