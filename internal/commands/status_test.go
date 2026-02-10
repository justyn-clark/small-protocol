package commands

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/justyn-clark/small-protocol/internal/workspace"
)

func TestDependencySatisfaction(t *testing.T) {
	tests := []struct {
		name           string
		tasks          []PlanTask
		maxActionable  int
		wantActionable []string
	}{
		{
			name: "no dependencies - all pending are actionable",
			tasks: []PlanTask{
				{ID: "task-1", Title: "Test 1", Status: "pending"},
				{ID: "task-2", Title: "Test 2", Status: "pending"},
				{ID: "task-3", Title: "Test 3", Status: "pending"},
			},
			maxActionable:  3,
			wantActionable: []string{"task-1", "task-2", "task-3"},
		},
		{
			name: "completed deps unlock pending",
			tasks: []PlanTask{
				{ID: "task-1", Title: "Test 1", Status: "completed"},
				{ID: "task-2", Title: "Test 2", Status: "pending", Dependencies: []string{"task-1"}},
			},
			maxActionable:  3,
			wantActionable: []string{"task-2"},
		},
		{
			name: "incomplete deps block pending",
			tasks: []PlanTask{
				{ID: "task-1", Title: "Test 1", Status: "pending"},
				{ID: "task-2", Title: "Test 2", Status: "pending", Dependencies: []string{"task-1"}},
			},
			maxActionable:  3,
			wantActionable: []string{"task-1"},
		},
		{
			name: "multiple deps all must be completed",
			tasks: []PlanTask{
				{ID: "task-1", Title: "Test 1", Status: "completed"},
				{ID: "task-2", Title: "Test 2", Status: "pending"},
				{ID: "task-3", Title: "Test 3", Status: "pending", Dependencies: []string{"task-1", "task-2"}},
			},
			maxActionable:  3,
			wantActionable: []string{"task-2"},
		},
		{
			name: "limit actionable count",
			tasks: []PlanTask{
				{ID: "task-1", Title: "Test 1", Status: "pending"},
				{ID: "task-2", Title: "Test 2", Status: "pending"},
				{ID: "task-3", Title: "Test 3", Status: "pending"},
				{ID: "task-4", Title: "Test 4", Status: "pending"},
			},
			maxActionable:  2,
			wantActionable: []string{"task-1", "task-2"},
		},
		{
			name: "only pending tasks are actionable",
			tasks: []PlanTask{
				{ID: "task-1", Title: "Test 1", Status: "completed"},
				{ID: "task-2", Title: "Test 2", Status: "in_progress"},
				{ID: "task-3", Title: "Test 3", Status: "blocked"},
				{ID: "task-4", Title: "Test 4", Status: "pending"},
			},
			maxActionable:  3,
			wantActionable: []string{"task-4"},
		},
		{
			name: "no pending tasks",
			tasks: []PlanTask{
				{ID: "task-1", Title: "Test 1", Status: "completed"},
				{ID: "task-2", Title: "Test 2", Status: "completed"},
			},
			maxActionable:  3,
			wantActionable: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Build a map of task statuses for dependency checking
			taskStatuses := make(map[string]string)
			tasksByStatus := make(map[string]int)
			for _, task := range tt.tasks {
				taskStatuses[task.ID] = task.Status
				tasksByStatus[task.Status]++
			}

			// Find actionable tasks (pending with all deps satisfied)
			var actionable []string
			for _, task := range tt.tasks {
				if task.Status != "pending" {
					continue
				}

				depsSatisfied := true
				for _, depID := range task.Dependencies {
					depStatus, exists := taskStatuses[depID]
					if !exists || depStatus != "completed" {
						depsSatisfied = false
						break
					}
				}

				if depsSatisfied && len(actionable) < tt.maxActionable {
					actionable = append(actionable, task.ID)
				}
			}

			// Compare results
			if len(actionable) != len(tt.wantActionable) {
				t.Errorf("got %d actionable tasks, want %d", len(actionable), len(tt.wantActionable))
				t.Errorf("got: %v, want: %v", actionable, tt.wantActionable)
				return
			}

			for i, id := range actionable {
				if id != tt.wantActionable[i] {
					t.Errorf("actionable[%d] = %s, want %s", i, id, tt.wantActionable[i])
				}
			}
		})
	}
}

func TestFormatTimestamp(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"2024-01-15T09:00:00Z", "2024-01-15 09:00:00"},
		{"2024-12-31T23:59:59Z", "2024-12-31 23:59:59"},
		{"invalid", "invalid"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := formatTimestamp(tt.input)
			if result != tt.expected {
				t.Errorf("formatTimestamp(%s) = %s, want %s", tt.input, result, tt.expected)
			}
		})
	}
}

func TestResolveNextTask(t *testing.T) {
	plan := &PlanStatus{
		TotalTasks:      2,
		TasksByStatus:   map[string]int{"completed": 1, "pending": 1},
		FirstIncomplete: "task-2",
	}
	if got := resolveNextTask(plan, "task-9"); got != "task-9" {
		t.Fatalf("resolveNextTask() with handoff = %q, want task-9", got)
	}
	if got := resolveNextTask(plan, ""); got != "task-2" {
		t.Fatalf("resolveNextTask() fallback = %q, want task-2", got)
	}

	complete := &PlanStatus{
		TotalTasks:      2,
		TasksByStatus:   map[string]int{"completed": 2},
		FirstIncomplete: "",
	}
	if got := resolveNextTask(complete, ""); got != "No active task (run complete)" {
		t.Fatalf("resolveNextTask() complete = %q, want run complete message", got)
	}
}

func TestGetRecentProgressFiltersTelemetryInSignalMode(t *testing.T) {
	tmpDir := t.TempDir()
	artifacts := cloneArtifacts(defaultArtifacts())
	artifacts["progress.small.yml"] = `small_version: "1.0.0"
owner: "agent"
entries:
  - task_id: "task-1"
    status: "in_progress"
    timestamp: "2026-01-01T00:00:00.000000000Z"
    evidence: "Apply started"
    notes: "apply: execution started"
  - task_id: "task-1"
    status: "completed"
    timestamp: "2026-01-01T00:00:01.000000000Z"
    evidence: "Command completed successfully"
    notes: "apply: exit code 0"
  - task_id: "meta/reconcile-plan"
    status: "completed"
    timestamp: "2026-01-01T00:00:02.000000000Z"
    evidence: "Reconciled"
`
	writeArtifacts(t, tmpDir, artifacts)
	mustSaveWorkspace(t, tmpDir, workspace.KindRepoRoot)
	if err := workspace.SetRunReplayID(tmpDir, strings.Repeat("b", 64)); err != nil {
		t.Fatalf("failed to set workspace replay id: %v", err)
	}

	entries, err := getRecentProgress(tmpDir, 5, true)
	if err != nil {
		t.Fatalf("getRecentProgress error: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 signal entries, got %d", len(entries))
	}
	if entries[0].TaskID != "meta/reconcile-plan" {
		t.Fatalf("entries[0].task_id = %q, want meta/reconcile-plan", entries[0].TaskID)
	}
	if entries[1].TaskID != "task-1" || entries[1].Status != "completed" {
		t.Fatalf("expected task-1 completed entry, got %+v", entries[1])
	}
}

func TestStatusJSONIncludesSignalSummary(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv(progressModeEnvVar, string(progressModeSignal))

	artifacts := cloneArtifacts(defaultArtifacts())
	artifacts["plan.small.yml"] = `small_version: "1.0.0"
owner: "agent"
tasks:
  - id: "task-1"
    title: "Done"
    status: "completed"
  - id: "task-2"
    title: "Next"
    status: "pending"
`
	artifacts["progress.small.yml"] = `small_version: "1.0.0"
owner: "agent"
entries:
  - task_id: "task-2"
    status: "in_progress"
    timestamp: "2026-01-01T00:00:00.000000000Z"
    evidence: "Apply started"
    notes: "apply: execution started"
  - task_id: "task-2"
    status: "completed"
    timestamp: "2026-01-01T00:00:01.000000000Z"
    evidence: "Command completed successfully"
    notes: "apply: exit code 0"
`
	artifacts["handoff.small.yml"] = `small_version: "1.0.0"
owner: "agent"
summary: "Run in progress. Next task: task-2."
generated_at: "2026-01-01T00:00:02.000000000Z"
resume:
  current_task_id: "task-2"
  next_steps: []
links: []
replayId:
  value: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
  source: "auto"
`
	writeArtifacts(t, tmpDir, artifacts)
	mustSaveWorkspace(t, tmpDir, workspace.KindRepoRoot)
	if err := workspace.SetRunReplayID(tmpDir, strings.Repeat("b", 64)); err != nil {
		t.Fatalf("failed to set workspace replay id: %v", err)
	}

	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe error: %v", err)
	}
	os.Stdout = w
	defer func() { os.Stdout = oldStdout }()

	var buf bytes.Buffer
	done := make(chan struct{})
	go func() {
		_, _ = io.Copy(&buf, r)
		_ = r.Close()
		close(done)
	}()

	cmd := statusCmd()
	cmd.SetArgs([]string{"--dir", tmpDir, "--json"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("status execute failed: %v", err)
	}
	_ = w.Close()
	<-done

	var out StatusOutput
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatalf("failed to parse status JSON: %v\noutput=%s", err, buf.String())
	}

	if out.ReplayID != strings.Repeat("b", 64) {
		t.Fatalf("replay_id = %q, want workspace replay id", out.ReplayID)
	}
	if out.NextTask != "task-2" {
		t.Fatalf("next_task = %q, want task-2", out.NextTask)
	}
	if out.Plan == nil {
		t.Fatal("expected plan summary")
	}
	if out.Plan.TasksByStatus["completed"] != 1 || out.Plan.TasksByStatus["pending"] != 1 {
		t.Fatalf("unexpected plan counts: %+v", out.Plan.TasksByStatus)
	}
	if len(out.RecentProgress) != 1 || out.RecentProgress[0].TaskID != "task-2" {
		t.Fatalf("unexpected recent signal progress: %+v", out.RecentProgress)
	}
}

func TestStatusJSONOmitsReplayIDWhenPlanMissing(t *testing.T) {
	tmpDir := t.TempDir()
	artifacts := cloneArtifacts(defaultArtifacts())
	delete(artifacts, "plan.small.yml")
	writeArtifacts(t, tmpDir, artifacts)
	mustSaveWorkspace(t, tmpDir, workspace.KindRepoRoot)

	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe error: %v", err)
	}
	os.Stdout = w
	defer func() { os.Stdout = oldStdout }()

	var buf bytes.Buffer
	done := make(chan struct{})
	go func() {
		_, _ = io.Copy(&buf, r)
		_ = r.Close()
		close(done)
	}()

	cmd := statusCmd()
	cmd.SetArgs([]string{"--dir", tmpDir, "--json"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("status execute failed: %v", err)
	}
	_ = w.Close()
	<-done

	var raw map[string]any
	if err := json.Unmarshal(buf.Bytes(), &raw); err != nil {
		t.Fatalf("failed to parse status JSON: %v\noutput=%s", err, buf.String())
	}
	if _, exists := raw["replay_id"]; exists {
		t.Fatalf("expected replay_id to be omitted when plan is missing, got %v", raw["replay_id"])
	}
}
