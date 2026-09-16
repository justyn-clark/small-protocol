package commands

import (
	"strings"
	"testing"
	"time"
)

func TestBuildReconstructOutputFiltersByTask(t *testing.T) {
	tmpDir := t.TempDir()
	if err := runSelftestInit(tmpDir); err != nil {
		t.Fatalf("runSelftestInit failed: %v", err)
	}

	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	if err := appendProgressEntry(tmpDir, map[string]any{
		"task_id":         "task-1",
		"status":          "in_progress",
		"timestamp":       formatProgressTimestamp(base),
		"evidence":        "started task one",
		"command_summary": "go test ./...",
		"command_ref":     ".small-cache/logs/example/commands/one.txt",
	}); err != nil {
		t.Fatalf("append task-1 progress failed: %v", err)
	}
	if err := appendProgressEntry(tmpDir, map[string]any{
		"task_id":   "meta/other",
		"status":    "completed",
		"timestamp": formatProgressTimestamp(base.Add(time.Second)),
		"evidence":  "other work",
	}); err != nil {
		t.Fatalf("append other progress failed: %v", err)
	}

	output, err := buildReconstructOutput(tmpDir, "task-1", "", 50)
	if err != nil {
		t.Fatalf("buildReconstructOutput failed: %v", err)
	}
	if output.Task == nil || output.Task.ID != "task-1" {
		t.Fatalf("expected task metadata for task-1")
	}
	if len(output.Entries) != 1 {
		t.Fatalf("expected one task-1 entry, got %d", len(output.Entries))
	}
	entry := output.Entries[0]
	if entry.Evidence != "started task one" || entry.CommandSummary != "go test ./..." || entry.CommandRef == "" {
		t.Fatalf("unexpected reconstructed entry: %+v", entry)
	}
}

func TestBuildReconstructOutputReportsTruncation(t *testing.T) {
	tmpDir := t.TempDir()
	if err := runSelftestInit(tmpDir); err != nil {
		t.Fatalf("runSelftestInit failed: %v", err)
	}

	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	for i := range 3 {
		if err := appendProgressEntry(tmpDir, map[string]any{
			"task_id":   "task-1",
			"status":    "in_progress",
			"timestamp": formatProgressTimestamp(base.Add(time.Duration(i) * time.Second)),
			"evidence":  "step",
		}); err != nil {
			t.Fatalf("append progress %d failed: %v", i, err)
		}
	}

	output, err := buildReconstructOutput(tmpDir, "task-1", "", 2)
	if err != nil {
		t.Fatalf("buildReconstructOutput failed: %v", err)
	}
	if output.MatchedEntries != 3 {
		t.Fatalf("expected 3 matched entries, got %d", output.MatchedEntries)
	}
	if output.ReturnedEntries != 2 || len(output.Entries) != 2 {
		t.Fatalf("expected 2 returned entries, got %d", output.ReturnedEntries)
	}
	if !output.Truncated {
		t.Fatalf("expected truncated to be true")
	}

	text := formatReconstructText(output)
	if !strings.Contains(text, "Progress entries: 2 of 3") {
		t.Fatalf("expected text to report matched-vs-returned counts, got: %s", text)
	}
}

func TestFormatReconstructTextIncludesEvidence(t *testing.T) {
	text := formatReconstructText(reconstructOutput{
		Workspace: "/tmp/example",
		Task:      &reconstructTask{ID: "task-1", Title: "Example", Status: "completed"},
		Handoff:   reconstructHandoff{Summary: "Done", ReplayID: "abc123", NextSteps: []string{"Archive"}},
		Entries: []reconstructEntry{{
			Index:     0,
			Timestamp: "2026-01-01T00:00:00.000000000Z",
			TaskID:    "task-1",
			Status:    "completed",
			Evidence:  "tests passed",
		}},
	})

	for _, expected := range []string{"Workspace: /tmp/example", "Task: task-1 - Example", "tests passed"} {
		if !strings.Contains(text, expected) {
			t.Fatalf("expected reconstruct text to contain %q, got %s", expected, text)
		}
	}
}
