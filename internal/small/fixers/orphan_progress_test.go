package fixers

import (
	"testing"

	"github.com/justyn-clark/small-protocol/internal/small"
)

func TestFixOrphanProgress_RewritesInScope(t *testing.T) {
	baseDir := t.TempDir()
	currentReplayID := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	otherReplayID := "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"

	plan := map[string]any{
		"small_version": small.ProtocolVersion,
		"owner":         "agent",
		"tasks": []any{
			map[string]any{
				"id":    "task-1",
				"title": "Known task",
			},
		},
	}

	progress := map[string]any{
		"small_version": small.ProtocolVersion,
		"owner":         "agent",
		"entries": []any{
			map[string]any{
				"task_id":   "task-1",
				"status":    "completed",
				"timestamp": "2025-01-01T00:00:00.000000001Z",
				"evidence":  "ok",
				"replayId":  currentReplayID,
			},
			map[string]any{
				"task_id":   "reset",
				"status":    "completed",
				"timestamp": "2025-01-01T00:00:00.000000002Z",
				"evidence":  "reset",
				"replayId":  currentReplayID,
			},
			map[string]any{
				"task_id":   "task-99",
				"status":    "completed",
				"timestamp": "2025-01-01T00:00:00.000000003Z",
				"evidence":  "historical",
				"replayId":  currentReplayID,
			},
			map[string]any{
				"task_id":   "custom-task",
				"status":    "completed",
				"timestamp": "2025-01-01T00:00:00.000000004Z",
				"evidence":  "unknown",
				"replayId":  currentReplayID,
			},
			map[string]any{
				"task_id":   "task-legacy",
				"status":    "completed",
				"timestamp": "2025-01-01T00:00:00.000000005Z",
				"evidence":  "other run",
				"replayId":  otherReplayID,
			},
			map[string]any{
				"task_id":   "meta/keep",
				"status":    "completed",
				"timestamp": "2025-01-01T00:00:00.000000006Z",
				"evidence":  "meta",
				"replayId":  currentReplayID,
			},
		},
	}

	handoff := map[string]any{
		"small_version": small.ProtocolVersion,
		"owner":         "agent",
		"summary":       "Test handoff",
		"resume": map[string]any{
			"current_task_id": "",
			"next_steps":      []any{},
		},
		"links": []any{},
		"replayId": map[string]any{
			"value":  currentReplayID,
			"source": "auto",
		},
	}

	writeArtifact(t, baseDir, "plan.small.yml", plan)
	writeArtifact(t, baseDir, "progress.small.yml", progress)
	writeArtifact(t, baseDir, "handoff.small.yml", handoff)

	result, err := FixOrphanProgress(baseDir)
	if err != nil {
		t.Fatalf("FixOrphanProgress error: %v", err)
	}
	if len(result.Rewrites) != 3 {
		t.Fatalf("expected 3 rewrites, got %d", len(result.Rewrites))
	}
	if result.Counts.Operational != 1 || result.Counts.Historical != 1 || result.Counts.Unknown != 1 {
		t.Fatalf("unexpected counts: %+v", result.Counts)
	}

	progressAfter, err := small.LoadArtifact(baseDir, "progress.small.yml")
	if err != nil {
		t.Fatalf("failed to reload progress: %v", err)
	}
	entries, ok := progressAfter.Data["entries"].([]any)
	if !ok {
		t.Fatal("progress entries missing after rewrite")
	}

	got := make([]string, 0, len(entries))
	for _, entry := range entries {
		entryMap, ok := entry.(map[string]any)
		if !ok {
			continue
		}
		got = append(got, stringVal(entryMap["task_id"]))
	}

	expected := []string{
		"task-1",
		"meta/reset",
		"meta/historical/task-99",
		"meta/historical/unknown/custom-task",
		"task-legacy",
		"meta/keep",
	}

	if len(got) != len(expected) {
		t.Fatalf("expected %d entries, got %d", len(expected), len(got))
	}
	for i := range expected {
		if got[i] != expected[i] {
			t.Fatalf("entry %d expected %q, got %q", i, expected[i], got[i])
		}
	}

	result, err = FixOrphanProgress(baseDir)
	if err != nil {
		t.Fatalf("second FixOrphanProgress error: %v", err)
	}
	if len(result.Rewrites) != 0 {
		t.Fatalf("expected idempotent fix, got %d rewrites", len(result.Rewrites))
	}
}

func writeArtifact(t *testing.T, baseDir, filename string, data map[string]any) {
	t.Helper()
	if err := small.SaveArtifact(baseDir, filename, data); err != nil {
		t.Fatalf("failed to write %s: %v", filename, err)
	}
}
