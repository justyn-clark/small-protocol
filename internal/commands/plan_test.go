package commands

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/justyn-clark/small-protocol/internal/small"
	"gopkg.in/yaml.v3"
)

func TestGenerateNextTaskID(t *testing.T) {
	tests := []struct {
		name     string
		tasks    []PlanTask
		expected string
	}{
		{
			name:     "empty tasks",
			tasks:    []PlanTask{},
			expected: "task-1",
		},
		{
			name: "single task",
			tasks: []PlanTask{
				{ID: "task-1", Title: "Test", Status: "pending"},
			},
			expected: "task-2",
		},
		{
			name: "multiple sequential tasks",
			tasks: []PlanTask{
				{ID: "task-1", Title: "Test 1", Status: "pending"},
				{ID: "task-2", Title: "Test 2", Status: "pending"},
				{ID: "task-3", Title: "Test 3", Status: "pending"},
			},
			expected: "task-4",
		},
		{
			name: "non-sequential task IDs",
			tasks: []PlanTask{
				{ID: "task-1", Title: "Test 1", Status: "pending"},
				{ID: "task-5", Title: "Test 5", Status: "pending"},
				{ID: "task-3", Title: "Test 3", Status: "pending"},
			},
			expected: "task-6",
		},
		{
			name: "tasks with non-numeric IDs mixed",
			tasks: []PlanTask{
				{ID: "task-1", Title: "Test 1", Status: "pending"},
				{ID: "custom-task", Title: "Custom", Status: "pending"},
				{ID: "task-10", Title: "Test 10", Status: "pending"},
			},
			expected: "task-11",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateNextTaskID(tt.tasks)
			if result != tt.expected {
				t.Errorf("generateNextTaskID() = %s, expected %s", result, tt.expected)
			}
		})
	}
}

func TestFindTask(t *testing.T) {
	plan := &PlanData{
		Tasks: []PlanTask{
			{ID: "task-1", Title: "Test 1", Status: "pending"},
			{ID: "task-2", Title: "Test 2", Status: "completed"},
			{ID: "task-3", Title: "Test 3", Status: "blocked"},
		},
	}

	tests := []struct {
		name      string
		taskID    string
		wantFound bool
		wantIndex int
	}{
		{"find first task", "task-1", true, 0},
		{"find middle task", "task-2", true, 1},
		{"find last task", "task-3", true, 2},
		{"task not found", "task-99", false, -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task, idx := findTask(plan, tt.taskID)
			if tt.wantFound && task == nil {
				t.Errorf("findTask(%s) should find task but didn't", tt.taskID)
			}
			if !tt.wantFound && task != nil {
				t.Errorf("findTask(%s) should not find task but did", tt.taskID)
			}
			if idx != tt.wantIndex {
				t.Errorf("findTask(%s) index = %d, want %d", tt.taskID, idx, tt.wantIndex)
			}
		})
	}
}

func TestSetTaskStatus(t *testing.T) {
	plan := &PlanData{
		Tasks: []PlanTask{
			{ID: "task-1", Title: "Test 1", Status: "pending"},
		},
	}

	// Test setting status
	err := setTaskStatus(plan, "task-1", "completed")
	if err != nil {
		t.Errorf("setTaskStatus() error = %v", err)
	}
	if plan.Tasks[0].Status != "completed" {
		t.Errorf("setTaskStatus() status = %s, want completed", plan.Tasks[0].Status)
	}

	// Test non-existent task
	err = setTaskStatus(plan, "task-99", "completed")
	if err == nil {
		t.Error("setTaskStatus() should error on non-existent task")
	}
}

func TestAddDependency(t *testing.T) {
	tests := []struct {
		name    string
		plan    *PlanData
		taskID  string
		depID   string
		wantErr bool
	}{
		{
			name: "valid dependency",
			plan: &PlanData{
				Tasks: []PlanTask{
					{ID: "task-1", Title: "Test 1", Status: "pending"},
					{ID: "task-2", Title: "Test 2", Status: "pending"},
				},
			},
			taskID:  "task-2",
			depID:   "task-1",
			wantErr: false,
		},
		{
			name: "task not found",
			plan: &PlanData{
				Tasks: []PlanTask{
					{ID: "task-1", Title: "Test 1", Status: "pending"},
				},
			},
			taskID:  "task-99",
			depID:   "task-1",
			wantErr: true,
		},
		{
			name: "dependency not found",
			plan: &PlanData{
				Tasks: []PlanTask{
					{ID: "task-1", Title: "Test 1", Status: "pending"},
				},
			},
			taskID:  "task-1",
			depID:   "task-99",
			wantErr: true,
		},
		{
			name: "self dependency",
			plan: &PlanData{
				Tasks: []PlanTask{
					{ID: "task-1", Title: "Test 1", Status: "pending"},
				},
			},
			taskID:  "task-1",
			depID:   "task-1",
			wantErr: true,
		},
		{
			name: "duplicate dependency",
			plan: &PlanData{
				Tasks: []PlanTask{
					{ID: "task-1", Title: "Test 1", Status: "pending"},
					{ID: "task-2", Title: "Test 2", Status: "pending", Dependencies: []string{"task-1"}},
				},
			},
			taskID:  "task-2",
			depID:   "task-1",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := addDependency(tt.plan, tt.taskID, tt.depID)
			if (err != nil) != tt.wantErr {
				t.Errorf("addDependency() error = %v, wantErr = %v", err, tt.wantErr)
			}
		})
	}
}

func TestEnsureProgressEvidenceWritesValidEntry(t *testing.T) {
	artifactsDir := t.TempDir()
	smallDir := filepath.Join(artifactsDir, small.SmallDir)
	if err := os.MkdirAll(smallDir, 0o755); err != nil {
		t.Fatalf("failed to create small dir: %v", err)
	}

	taskID := "task-1"
	if err := ensureProgressEvidence(artifactsDir, taskID); err != nil {
		t.Fatalf("ensureProgressEvidence error: %v", err)
	}

	progressPath := filepath.Join(smallDir, "progress.small.yml")
	data, err := os.ReadFile(progressPath)
	if err != nil {
		t.Fatalf("failed to read progress file: %v", err)
	}

	var parsed map[string]any
	if err := yaml.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to parse progress YAML: %v", err)
	}

	entries, ok := parsed["entries"].([]any)
	if !ok || len(entries) != 1 {
		t.Fatalf("expected one progress entry, got %v", entries)
	}

	entry, ok := entries[0].(map[string]any)
	if !ok {
		t.Fatalf("progress entry is not an object: %T", entries[0])
	}

	if entry["task_id"] != taskID {
		t.Fatalf("task_id = %v, want %s", entry["task_id"], taskID)
	}

	note, _ := entry["notes"].(string)
	if note != planDoneProgressNote {
		t.Fatalf("notes = %q, want %q", note, planDoneProgressNote)
	}

	timestamp, ok := entry["timestamp"].(string)
	if !ok {
		t.Fatalf("timestamp missing or not a string: %v", entry["timestamp"])
	}
	if _, err := small.ParseProgressTimestamp(timestamp); err != nil {
		t.Fatalf("timestamp not RFC3339Nano with fractional seconds: %v", err)
	}

	if !small.ProgressEntryHasValidEvidence(entry) {
		t.Fatal("progress entry should satisfy evidence requirements")
	}

	if err := ensureProgressEvidence(artifactsDir, taskID); err != nil {
		t.Fatalf("ensureProgressEvidence error on second call: %v", err)
	}

	data, err = os.ReadFile(progressPath)
	if err != nil {
		t.Fatalf("failed to read progress file after second call: %v", err)
	}

	var parsedAfter map[string]any
	if err := yaml.Unmarshal(data, &parsedAfter); err != nil {
		t.Fatalf("failed to parse progress YAML after second call: %v", err)
	}

	entriesAfter, ok := parsedAfter["entries"].([]any)
	if !ok {
		t.Fatalf("progress entries missing after second call: %v", parsedAfter["entries"])
	}
	if len(entriesAfter) != 1 {
		t.Fatalf("expected one entry after second call, got %d", len(entriesAfter))
	}

	planArtifact := &small.Artifact{
		Path: filepath.Join(smallDir, "plan.small.yml"),
		Type: "plan",
		Data: map[string]any{
			"small_version": small.ProtocolVersion,
			"owner":         "agent",
			"tasks": []any{
				map[string]any{
					"id":     taskID,
					"title":  "Test CLI entry",
					"status": "completed",
				},
			},
		},
	}

	progressArtifact := &small.Artifact{
		Path: progressPath,
		Type: "progress",
		Data: parsedAfter,
	}

	violations := small.CheckInvariants(map[string]*small.Artifact{
		"plan":     planArtifact,
		"progress": progressArtifact,
	}, false)
	if len(violations) != 0 {
		t.Fatalf("expected no invariant violations, got %+v", violations)
	}
}
