package commands

import (
	"testing"
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
				{ID: "task-1", Description: "Test", Status: "pending"},
			},
			expected: "task-2",
		},
		{
			name: "multiple sequential tasks",
			tasks: []PlanTask{
				{ID: "task-1", Description: "Test 1", Status: "pending"},
				{ID: "task-2", Description: "Test 2", Status: "pending"},
				{ID: "task-3", Description: "Test 3", Status: "pending"},
			},
			expected: "task-4",
		},
		{
			name: "non-sequential task IDs",
			tasks: []PlanTask{
				{ID: "task-1", Description: "Test 1", Status: "pending"},
				{ID: "task-5", Description: "Test 5", Status: "pending"},
				{ID: "task-3", Description: "Test 3", Status: "pending"},
			},
			expected: "task-6",
		},
		{
			name: "tasks with non-numeric IDs mixed",
			tasks: []PlanTask{
				{ID: "task-1", Description: "Test 1", Status: "pending"},
				{ID: "custom-task", Description: "Custom", Status: "pending"},
				{ID: "task-10", Description: "Test 10", Status: "pending"},
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
			{ID: "task-1", Description: "Test 1", Status: "pending"},
			{ID: "task-2", Description: "Test 2", Status: "completed"},
			{ID: "task-3", Description: "Test 3", Status: "blocked"},
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
			{ID: "task-1", Description: "Test 1", Status: "pending"},
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
					{ID: "task-1", Description: "Test 1", Status: "pending"},
					{ID: "task-2", Description: "Test 2", Status: "pending"},
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
					{ID: "task-1", Description: "Test 1", Status: "pending"},
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
					{ID: "task-1", Description: "Test 1", Status: "pending"},
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
					{ID: "task-1", Description: "Test 1", Status: "pending"},
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
					{ID: "task-1", Description: "Test 1", Status: "pending"},
					{ID: "task-2", Description: "Test 2", Status: "pending", Dependencies: []string{"task-1"}},
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
