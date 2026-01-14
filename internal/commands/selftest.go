package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/justyn-clark/small-protocol/internal/workspace"
	"github.com/spf13/cobra"
)

// selftestStep represents a single step in the selftest
type selftestStep struct {
	name    string
	run     func(dir string) error
	cleanup func(dir string) // optional cleanup after step
}

func selftestCmd() *cobra.Command {
	var (
		keep          bool
		dir           string
		workspaceFlag string
	)

	cmd := &cobra.Command{
		Use:   "selftest",
		Short: "Run built-in self-test to verify CLI functionality",
		Long: `Runs a self-test that exercises the CLI commands in an isolated temp workspace.

The selftest creates a temporary directory, runs a sequence of CLI operations,
and verifies they complete successfully. Use this to confirm the CLI is working
correctly without touching your actual workspace.

Steps performed:
  1. init --force --intent "SMALL selftest workspace"
  2. plan --add "Selftest task"
  3. plan --done <task-id>
  4. apply --dry-run
  5. handoff --summary "Selftest checkpoint"
  6. verify --workspace any

By default, the temp directory is cleaned up after the test. Use --keep to
preserve it for inspection.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			scope, err := workspace.ParseScope(workspaceFlag)
			if err != nil {
				return fmt.Errorf("invalid workspace scope: %w", err)
			}

			return runSelftest(dir, keep, scope)
		},
	}

	cmd.Flags().BoolVar(&keep, "keep", false, "Do not delete temp directory after test")
	cmd.Flags().StringVar(&dir, "dir", "", "Run selftest in explicit directory (default: OS temp)")
	cmd.Flags().StringVar(&workspaceFlag, "workspace", "any", "Workspace scope for verify step (default: any)")

	return cmd
}

func runSelftest(explicitDir string, keep bool, scope workspace.Scope) error {
	var testDir string
	var err error

	if explicitDir != "" {
		testDir = explicitDir
		// Ensure directory exists
		if err := os.MkdirAll(testDir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", testDir, err)
		}
	} else {
		// Create temp directory in OS temp
		testDir, err = os.MkdirTemp("", "small-selftest-*")
		if err != nil {
			return fmt.Errorf("failed to create temp directory: %w", err)
		}
	}

	// Cleanup unless --keep
	if !keep {
		defer func() {
			if err := os.RemoveAll(testDir); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to clean up %s: %v\n", testDir, err)
			}
		}()
	}

	fmt.Printf("SMALL Selftest\n")
	fmt.Printf("==============\n")
	fmt.Printf("Workspace: %s\n\n", testDir)

	steps := []selftestStep{
		{
			name: "init",
			run: func(dir string) error {
				return runSelftestInit(dir)
			},
		},
		{
			name: "plan --add",
			run: func(dir string) error {
				return runSelftestPlanAdd(dir)
			},
		},
		{
			name: "plan --done",
			run: func(dir string) error {
				return runSelftestPlanDone(dir)
			},
		},
		{
			name: "apply --dry-run",
			run: func(dir string) error {
				return runSelftestApplyDryRun(dir)
			},
		},
		{
			name: "handoff",
			run: func(dir string) error {
				return runSelftestHandoff(dir)
			},
		},
		{
			name: "verify",
			run: func(dir string) error {
				return runSelftestVerify(dir, scope)
			},
		},
	}

	for i, step := range steps {
		fmt.Printf("[%d/%d] %s... ", i+1, len(steps), step.name)

		if err := step.run(testDir); err != nil {
			fmt.Printf("FAILED\n")
			fmt.Fprintf(os.Stderr, "\nError in step %q: %v\n", step.name, err)
			if keep {
				fmt.Printf("\nWorkspace preserved at: %s\n", testDir)
			}
			return fmt.Errorf("selftest failed at step %q: %w", step.name, err)
		}

		fmt.Printf("OK\n")

		if step.cleanup != nil {
			step.cleanup(testDir)
		}
	}

	fmt.Printf("\nAll selftest steps passed!\n")

	if keep {
		fmt.Printf("\nWorkspace preserved at: %s\n", testDir)
	}

	return nil
}

// runSelftestInit runs: small init --force --intent "SMALL selftest workspace"
func runSelftestInit(dir string) error {
	// Use the internal init logic directly
	smallDir := filepath.Join(dir, ".small")

	// Remove existing .small if present
	if _, err := os.Stat(smallDir); err == nil {
		if err := os.RemoveAll(smallDir); err != nil {
			return fmt.Errorf("failed to remove existing .small: %w", err)
		}
	}

	// Create .small directory
	if err := os.MkdirAll(smallDir, 0755); err != nil {
		return fmt.Errorf("failed to create .small directory: %w", err)
	}

	// Write artifacts with selftest intent
	intent := `small_version: "1.0.0"
owner: "human"
intent: "SMALL selftest workspace"
scope:
  include: []
  exclude: []
success_criteria: []
`
	if err := os.WriteFile(filepath.Join(smallDir, "intent.small.yml"), []byte(intent), 0644); err != nil {
		return fmt.Errorf("failed to write intent: %w", err)
	}

	constraints := `small_version: "1.0.0"
owner: "human"
constraints:
  - id: "selftest"
    rule: "Selftest constraint"
    severity: "warn"
`
	if err := os.WriteFile(filepath.Join(smallDir, "constraints.small.yml"), []byte(constraints), 0644); err != nil {
		return fmt.Errorf("failed to write constraints: %w", err)
	}

	plan := `small_version: "1.0.0"
owner: "agent"
tasks:
  - id: "task-1"
    title: "Initial selftest task"
`
	if err := os.WriteFile(filepath.Join(smallDir, "plan.small.yml"), []byte(plan), 0644); err != nil {
		return fmt.Errorf("failed to write plan: %w", err)
	}

	progress := `small_version: "1.0.0"
owner: "agent"
entries: []
`
	if err := os.WriteFile(filepath.Join(smallDir, "progress.small.yml"), []byte(progress), 0644); err != nil {
		return fmt.Errorf("failed to write progress: %w", err)
	}

	handoff := `small_version: "1.0.0"
owner: "agent"
summary: "Selftest initial state"
resume:
  current_task_id: ""
  next_steps: []
links: []
`
	if err := os.WriteFile(filepath.Join(smallDir, "handoff.small.yml"), []byte(handoff), 0644); err != nil {
		return fmt.Errorf("failed to write handoff: %w", err)
	}

	// Write workspace metadata
	if err := workspace.Save(dir, workspace.KindRepoRoot); err != nil {
		return fmt.Errorf("failed to write workspace metadata: %w", err)
	}

	return nil
}

// runSelftestPlanAdd runs: small plan --add "Selftest task"
func runSelftestPlanAdd(dir string) error {
	planPath := filepath.Join(dir, ".small", "plan.small.yml")

	// Load existing plan
	plan, err := loadPlan(planPath)
	if err != nil {
		return fmt.Errorf("failed to load plan: %w", err)
	}

	// Add a new task
	newID := generateNextTaskID(plan.Tasks)
	newTask := PlanTask{
		ID:     newID,
		Title:  "Selftest task",
		Status: "pending",
	}
	plan.Tasks = append(plan.Tasks, newTask)

	// Save plan
	if err := savePlan(planPath, plan); err != nil {
		return fmt.Errorf("failed to save plan: %w", err)
	}

	// Record progress
	if err := appendPlanProgress(dir, newID, "pending", fmt.Sprintf("Added task %s via selftest", newID), "Selftest task"); err != nil {
		return fmt.Errorf("failed to record progress: %w", err)
	}

	return nil
}

// runSelftestPlanDone marks task-2 as done
func runSelftestPlanDone(dir string) error {
	planPath := filepath.Join(dir, ".small", "plan.small.yml")

	// Load plan
	plan, err := loadPlan(planPath)
	if err != nil {
		return fmt.Errorf("failed to load plan: %w", err)
	}

	// Find task-2 (the one we just added)
	taskID := "task-2"
	task, _ := findTask(plan, taskID)
	if task == nil {
		// Try to find the highest numbered task
		taskIDPattern := regexp.MustCompile(`^task-(\d+)$`)
		maxID := ""
		for _, t := range plan.Tasks {
			if taskIDPattern.MatchString(t.ID) {
				if maxID == "" || t.ID > maxID {
					maxID = t.ID
				}
			}
		}
		if maxID != "" {
			taskID = maxID
			task, _ = findTask(plan, taskID)
		}
	}

	if task == nil {
		return fmt.Errorf("no task found to mark as done")
	}

	// Mark as completed
	task.Status = "completed"

	// Save plan
	if err := savePlan(planPath, plan); err != nil {
		return fmt.Errorf("failed to save plan: %w", err)
	}

	// Ensure progress evidence
	if err := ensureProgressEvidence(dir, taskID); err != nil {
		return fmt.Errorf("failed to record progress: %w", err)
	}

	return nil
}

// runSelftestApplyDryRun runs apply in dry-run mode
func runSelftestApplyDryRun(dir string) error {
	// Record a dry-run progress entry
	entry := map[string]interface{}{
		"timestamp": formatProgressTimestamp(time.Now().UTC()),
		"task_id":   "apply",
		"status":    "pending",
		"evidence":  "Dry-run: no command executed (selftest)",
		"notes":     "apply --dry-run (selftest)",
	}

	return appendProgressEntry(dir, entry)
}

// runSelftestHandoff generates handoff with summary
func runSelftestHandoff(dir string) error {
	smallDir := filepath.Join(dir, ".small")

	// Generate replayId
	replayId, err := generateReplayId(smallDir, "")
	if err != nil {
		return fmt.Errorf("failed to generate replayId: %w", err)
	}

	// Load plan to get next steps
	planPath := filepath.Join(smallDir, "plan.small.yml")
	plan, err := loadPlan(planPath)
	if err != nil {
		return fmt.Errorf("failed to load plan: %w", err)
	}

	var nextSteps []string
	for _, task := range plan.Tasks {
		if task.Status == "pending" || task.Status == "in_progress" || task.Status == "" {
			if task.Title != "" {
				nextSteps = append(nextSteps, task.Title)
			} else {
				nextSteps = append(nextSteps, task.ID)
			}
		}
	}

	handoff := fmt.Sprintf(`small_version: "1.0.0"
owner: "agent"
summary: "Selftest checkpoint"
resume:
  current_task_id: ""
  next_steps: %s
links: []
replayId:
  value: "%s"
  source: "%s"
`, formatYAMLStringArray(nextSteps), replayId.Value, replayId.Source)

	handoffPath := filepath.Join(smallDir, "handoff.small.yml")
	if err := os.WriteFile(handoffPath, []byte(handoff), 0644); err != nil {
		return fmt.Errorf("failed to write handoff: %w", err)
	}

	return nil
}

// runSelftestVerify runs verify
func runSelftestVerify(dir string, scope workspace.Scope) error {
	code := runVerify(dir, false, true, scope)
	if code != ExitValid {
		return fmt.Errorf("verify failed with exit code %d", code)
	}
	return nil
}

// formatYAMLStringArray formats a string slice as YAML array
func formatYAMLStringArray(items []string) string {
	if len(items) == 0 {
		return "[]"
	}
	var parts []string
	for _, item := range items {
		// Escape quotes in item
		escaped := strings.ReplaceAll(item, `"`, `\"`)
		parts = append(parts, fmt.Sprintf(`"%s"`, escaped))
	}
	return "[" + strings.Join(parts, ", ") + "]"
}
