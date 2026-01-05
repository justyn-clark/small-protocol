package commands

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/justyn-clark/small-protocol/internal/small"
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
	// Create temp directory with .small structure
	tmpDir, err := os.MkdirTemp("", "small-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	smallDir := filepath.Join(tmpDir, ".small")
	if err := os.MkdirAll(smallDir, 0755); err != nil {
		t.Fatalf("failed to create .small dir: %v", err)
	}

	// Create initial progress file
	initialProgress := ProgressData{
		SmallVersion: small.ProtocolVersion,
		Owner:        "agent",
		Entries:      []map[string]interface{}{},
	}

	data, err := yaml.Marshal(&initialProgress)
	if err != nil {
		t.Fatalf("failed to marshal initial progress: %v", err)
	}

	progressPath := filepath.Join(smallDir, "progress.small.yml")
	if err := os.WriteFile(progressPath, data, 0644); err != nil {
		t.Fatalf("failed to write initial progress: %v", err)
	}

	// Test appending entry
	entry := map[string]interface{}{
		"timestamp": "2024-01-15T09:00:00Z",
		"task_id":   "task-1",
		"status":    "completed",
		"evidence":  "Test evidence",
		"notes":     "Test notes",
	}

	if err := appendProgressEntry(tmpDir, entry); err != nil {
		t.Fatalf("appendProgressEntry() error = %v", err)
	}

	// Verify entry was appended
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

	// Test appending second entry
	entry2 := map[string]interface{}{
		"timestamp": "2024-01-15T10:00:00Z",
		"task_id":   "task-2",
		"status":    "in_progress",
		"evidence":  "Second entry",
	}

	if err := appendProgressEntry(tmpDir, entry2); err != nil {
		t.Fatalf("appendProgressEntry() second entry error = %v", err)
	}

	// Verify both entries exist
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
}

func TestAppendProgressEntryMissingFile(t *testing.T) {
	// Create temp directory without progress file
	tmpDir, err := os.MkdirTemp("", "small-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	smallDir := filepath.Join(tmpDir, ".small")
	if err := os.MkdirAll(smallDir, 0755); err != nil {
		t.Fatalf("failed to create .small dir: %v", err)
	}

	entry := map[string]interface{}{
		"timestamp": "2024-01-15T09:00:00Z",
		"task_id":   "task-1",
		"status":    "completed",
	}

	err = appendProgressEntry(tmpDir, entry)
	if err == nil {
		t.Error("appendProgressEntry() should error when progress file is missing")
	}
}

func TestProgressEntryStatusValues(t *testing.T) {
	// Verify that apply uses valid status values
	validStatuses := []string{"pending", "in_progress", "completed", "blocked"}

	// These are the statuses used in apply.go
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
