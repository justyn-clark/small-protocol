package commands

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDoctorCommand(t *testing.T) {
	// Create temp directory for tests
	tmpDir, err := os.MkdirTemp("", "small-doctor-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	t.Run("reports missing .small/ directory", func(t *testing.T) {
		results := runDoctor(tmpDir)

		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}

		if results[0].Status != "error" {
			t.Errorf("expected error status, got %s", results[0].Status)
		}

		if results[0].Suggestion != "Run: small init" {
			t.Errorf("unexpected suggestion: %s", results[0].Suggestion)
		}
	})

	// Create .small/ directory
	smallDir := filepath.Join(tmpDir, ".small")
	if err := os.MkdirAll(smallDir, 0755); err != nil {
		t.Fatalf("failed to create .small dir: %v", err)
	}

	t.Run("reports missing required files", func(t *testing.T) {
		results := runDoctor(tmpDir)

		hasFilesError := false
		for _, r := range results {
			if r.Category == "Files" && r.Status == "error" {
				hasFilesError = true
				break
			}
		}

		if !hasFilesError {
			t.Error("expected Files error for missing files")
		}
	})

	// Create valid artifacts
	validArtifacts := map[string]string{
		"intent.small.yml": `small_version: "1.0.0"
owner: "human"
intent: "Test intent"
scope:
  include: []
  exclude: []
success_criteria: []
`,
		"constraints.small.yml": `small_version: "1.0.0"
owner: "human"
constraints:
  - id: "test"
    rule: "Test rule"
    severity: "error"
`,
		"plan.small.yml": `small_version: "1.0.0"
owner: "agent"
tasks:
  - id: "task-1"
    title: "Test task"
    status: "pending"
`,
		"progress.small.yml": `small_version: "1.0.0"
owner: "agent"
entries: []
`,
		"handoff.small.yml": `small_version: "1.0.0"
owner: "agent"
summary: "Test handoff"
resume:
  current_task_id: null
  next_steps: []
links: []
replayId:
  value: "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2"
  source: "auto"
`,
	}

	for filename, content := range validArtifacts {
		path := filepath.Join(smallDir, filename)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("failed to write %s: %v", filename, err)
		}
	}

	t.Run("reports all OK for valid workspace", func(t *testing.T) {
		results := runDoctor(tmpDir)

		hasError := false
		for _, r := range results {
			if r.Status == "error" {
				hasError = true
				t.Errorf("unexpected error: [%s] %s", r.Category, r.Message)
			}
		}

		if hasError {
			t.Error("expected no errors for valid workspace")
		}
	})

	t.Run("detects schema violations", func(t *testing.T) {
		// Create invalid intent (wrong owner)
		invalidIntent := `small_version: "1.0.0"
owner: "agent"
intent: "Test"
scope:
  include: []
  exclude: []
success_criteria: []
`
		path := filepath.Join(smallDir, "intent.small.yml")
		if err := os.WriteFile(path, []byte(invalidIntent), 0644); err != nil {
			t.Fatalf("failed to write intent: %v", err)
		}

		results := runDoctor(tmpDir)

		hasSchemaError := false
		for _, r := range results {
			if r.Category == "Schema" && r.Status == "error" {
				hasSchemaError = true
				break
			}
		}

		if !hasSchemaError {
			t.Error("expected Schema error for invalid owner")
		}

		// Restore valid intent
		if err := os.WriteFile(path, []byte(validArtifacts["intent.small.yml"]), 0644); err != nil {
			t.Fatalf("failed to restore intent: %v", err)
		}
	})

	t.Run("analyzes run state", func(t *testing.T) {
		results := runDoctor(tmpDir)

		hasRunState := false
		for _, r := range results {
			if r.Category == "Run State" {
				hasRunState = true
				break
			}
		}

		if !hasRunState {
			t.Error("expected Run State analysis in results")
		}
	})
}

func TestDoctorNeverMutates(t *testing.T) {
	// Create temp directory for tests
	tmpDir, err := os.MkdirTemp("", "small-doctor-mutate-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	smallDir := filepath.Join(tmpDir, ".small")
	if err := os.MkdirAll(smallDir, 0755); err != nil {
		t.Fatalf("failed to create .small dir: %v", err)
	}

	// Create test files
	testContent := "test: content\n"
	files := []string{"intent.small.yml", "plan.small.yml", "progress.small.yml"}
	for _, f := range files {
		path := filepath.Join(smallDir, f)
		if err := os.WriteFile(path, []byte(testContent), 0644); err != nil {
			t.Fatalf("failed to write %s: %v", f, err)
		}
	}

	// Record modification times before
	modTimes := make(map[string]int64)
	for _, f := range files {
		path := filepath.Join(smallDir, f)
		info, _ := os.Stat(path)
		modTimes[f] = info.ModTime().UnixNano()
	}

	// Run doctor (should never mutate)
	_ = runDoctor(tmpDir)

	// Verify no files were modified
	for _, f := range files {
		path := filepath.Join(smallDir, f)
		info, _ := os.Stat(path)
		if info.ModTime().UnixNano() != modTimes[f] {
			t.Errorf("doctor mutated file: %s", f)
		}
	}
}
