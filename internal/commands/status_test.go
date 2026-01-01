package commands

import (
	"testing"
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
				{ID: "task-1", Description: "Test 1", Status: "pending"},
				{ID: "task-2", Description: "Test 2", Status: "pending"},
				{ID: "task-3", Description: "Test 3", Status: "pending"},
			},
			maxActionable:  3,
			wantActionable: []string{"task-1", "task-2", "task-3"},
		},
		{
			name: "completed deps unlock pending",
			tasks: []PlanTask{
				{ID: "task-1", Description: "Test 1", Status: "completed"},
				{ID: "task-2", Description: "Test 2", Status: "pending", Dependencies: []string{"task-1"}},
			},
			maxActionable:  3,
			wantActionable: []string{"task-2"},
		},
		{
			name: "incomplete deps block pending",
			tasks: []PlanTask{
				{ID: "task-1", Description: "Test 1", Status: "pending"},
				{ID: "task-2", Description: "Test 2", Status: "pending", Dependencies: []string{"task-1"}},
			},
			maxActionable:  3,
			wantActionable: []string{"task-1"},
		},
		{
			name: "multiple deps all must be completed",
			tasks: []PlanTask{
				{ID: "task-1", Description: "Test 1", Status: "completed"},
				{ID: "task-2", Description: "Test 2", Status: "pending"},
				{ID: "task-3", Description: "Test 3", Status: "pending", Dependencies: []string{"task-1", "task-2"}},
			},
			maxActionable:  3,
			wantActionable: []string{"task-2"},
		},
		{
			name: "limit actionable count",
			tasks: []PlanTask{
				{ID: "task-1", Description: "Test 1", Status: "pending"},
				{ID: "task-2", Description: "Test 2", Status: "pending"},
				{ID: "task-3", Description: "Test 3", Status: "pending"},
				{ID: "task-4", Description: "Test 4", Status: "pending"},
			},
			maxActionable:  2,
			wantActionable: []string{"task-1", "task-2"},
		},
		{
			name: "only pending tasks are actionable",
			tasks: []PlanTask{
				{ID: "task-1", Description: "Test 1", Status: "completed"},
				{ID: "task-2", Description: "Test 2", Status: "in_progress"},
				{ID: "task-3", Description: "Test 3", Status: "blocked"},
				{ID: "task-4", Description: "Test 4", Status: "pending"},
			},
			maxActionable:  3,
			wantActionable: []string{"task-4"},
		},
		{
			name: "no pending tasks",
			tasks: []PlanTask{
				{ID: "task-1", Description: "Test 1", Status: "completed"},
				{ID: "task-2", Description: "Test 2", Status: "completed"},
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
