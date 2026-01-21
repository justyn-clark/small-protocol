package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/justyn-clark/small-protocol/internal/small"
	"github.com/justyn-clark/small-protocol/internal/workspace"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// PlanData represents the plan.small.yml structure
type PlanData struct {
	SmallVersion string     `yaml:"small_version"`
	Owner        string     `yaml:"owner"`
	Tasks        []PlanTask `yaml:"tasks"`
}

// PlanTask represents a task in the plan
// id and title are required by v1.0.0; steps, acceptance, status, dependencies are optional CLI conveniences
type PlanTask struct {
	ID           string   `yaml:"id"`
	Title        string   `yaml:"title"`
	Steps        []string `yaml:"steps,omitempty"`
	Acceptance   []string `yaml:"acceptance,omitempty"`
	Status       string   `yaml:"status,omitempty"`
	Dependencies []string `yaml:"dependencies,omitempty"`
}

const planDoneProgressNote = "completed via CLI"

func planCmd() *cobra.Command {
	var (
		reset         bool
		yes           bool
		addTask       string
		doneID        string
		pendingID     string
		blockedID     string
		dependsArg    string
		dir           string
		workspaceFlag string
	)

	cmd := &cobra.Command{
		Use:   "plan",
		Short: "Create or update plan.small.yml",
		Long:  "Manages the plan artifact. Creates from template if missing, or modifies existing plan.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if dir == "" {
				dir = baseDir
			}
			artifactsDir := resolveArtifactsDir(dir)
			smallDir := filepath.Join(artifactsDir, small.SmallDir)

			// Check if .small directory exists
			if _, err := os.Stat(smallDir); os.IsNotExist(err) {
				return fmt.Errorf(".small/ directory does not exist. Run 'small init' first")
			}

			scope, err := workspace.ParseScope(workspaceFlag)
			if err != nil {
				return err
			}
			if scope == workspace.ScopeExamples {
				return fmt.Errorf("--workspace examples is not supported for plan (use --workspace any to bypass)")
			}
			if scope != workspace.ScopeAny {
				if err := enforceWorkspaceScope(artifactsDir, workspace.ScopeRoot); err != nil {
					return err
				}
			}

			planPath := filepath.Join(smallDir, "plan.small.yml")

			planExists := small.ArtifactExists(artifactsDir, "plan.small.yml")

			// Handle --reset flag
			if reset {
				if !yes {
					return fmt.Errorf("--reset requires --yes flag to confirm overwrite in non-interactive mode")
				}
				if err := writePlanTemplate(planPath); err != nil {
					return err
				}
				if err := appendPlanProgress(artifactsDir, "plan-reset", "completed", "Reset plan.small.yml to template", "small plan --reset"); err != nil {
					return err
				}
				fmt.Println("Plan reset to template")
				return nil
			}

			// Load or create plan
			var plan PlanData
			if planExists {
				loadedPlan, err := loadPlan(planPath)
				if err != nil {
					return fmt.Errorf("failed to load existing plan: %w", err)
				}
				plan = *loadedPlan
			} else {
				plan = getDefaultPlan()
				fmt.Println("Creating new plan from template")
			}

			modified := false

			// Handle --add flag
			if addTask != "" {
				newID := generateNextTaskID(plan.Tasks)
				newTask := PlanTask{
					ID:     newID,
					Title:  addTask,
					Status: "pending",
				}
				plan.Tasks = append(plan.Tasks, newTask)
				if err := appendPlanProgress(artifactsDir, newID, "pending", fmt.Sprintf("Added task %s via small plan --add", newID), addTask); err != nil {
					return err
				}
				modified = true
				fmt.Printf("Added task %s: %s\n", newID, addTask)
			}

			// Handle --done flag
			if doneID != "" {
				if err := setTaskStatus(&plan, doneID, "completed"); err != nil {
					return err
				}
				if err := ensureProgressEvidence(artifactsDir, doneID); err != nil {
					return fmt.Errorf("failed to record progress for task %s: %w", doneID, err)
				}
				modified = true
				fmt.Printf("Marked task %s as completed\n", doneID)
			}

			// Handle --pending flag
			if pendingID != "" {
				if err := setTaskStatus(&plan, pendingID, "pending"); err != nil {
					return err
				}
				if err := appendPlanProgress(artifactsDir, pendingID, "pending", fmt.Sprintf("Updated task %s to pending", pendingID), "small plan --pending"); err != nil {
					return err
				}
				modified = true
				fmt.Printf("Marked task %s as pending\n", pendingID)
			}

			// Handle --blocked flag
			if blockedID != "" {
				if err := setTaskStatus(&plan, blockedID, "blocked"); err != nil {
					return err
				}
				if err := appendPlanProgress(artifactsDir, blockedID, "blocked", fmt.Sprintf("Updated task %s to blocked", blockedID), "small plan --blocked"); err != nil {
					return err
				}
				modified = true
				fmt.Printf("Marked task %s as blocked\n", blockedID)
			}

			// Handle --depends flag
			if dependsArg != "" {
				parts := strings.SplitN(dependsArg, ":", 2)
				if len(parts) != 2 {
					return fmt.Errorf("--depends format must be <task-id>:<dep-id>")
				}
				taskID := strings.TrimSpace(parts[0])
				depID := strings.TrimSpace(parts[1])
				if err := addDependency(&plan, taskID, depID); err != nil {
					return err
				}
				status := ""
				if task, _ := findTask(&plan, taskID); task != nil {
					status = task.Status
				}
				if err := appendPlanProgress(artifactsDir, taskID, status, fmt.Sprintf("Added dependency %s to %s", depID, taskID), "small plan --depends"); err != nil {
					return err
				}
				modified = true
				fmt.Printf("Added dependency: %s depends on %s\n", taskID, depID)
			}

			// Save plan if modified or newly created
			if modified || !planExists {
				if err := savePlan(planPath, &plan); err != nil {
					return fmt.Errorf("failed to save plan: %w", err)
				}
				if !modified && !planExists {
					fmt.Printf("Created %s\n", planPath)
				}
			} else {
				fmt.Println("No changes made to plan")
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&reset, "reset", false, "Reset plan to template (requires --yes)")
	cmd.Flags().BoolVar(&yes, "yes", false, "Confirm destructive operations (required with --reset)")
	cmd.Flags().StringVar(&addTask, "add", "", "Add a new task with the given title")
	cmd.Flags().StringVar(&doneID, "done", "", "Mark task as completed by ID")
	cmd.Flags().StringVar(&pendingID, "pending", "", "Mark task as pending by ID")
	cmd.Flags().StringVar(&blockedID, "blocked", "", "Mark task as blocked by ID")
	cmd.Flags().StringVar(&dependsArg, "depends", "", "Add dependency edge (format: <task-id>:<dep-id>)")
	cmd.Flags().StringVar(&dir, "dir", ".", "Directory containing .small/ artifacts")
	cmd.Flags().StringVar(&workspaceFlag, "workspace", string(workspace.ScopeRoot), "Workspace scope (root or any)")

	return cmd
}

func getDefaultPlan() PlanData {
	return PlanData{
		SmallVersion: small.ProtocolVersion,
		Owner:        "agent",
		Tasks: []PlanTask{
			{
				ID:    "task-1",
				Title: "Initial task",
			},
		},
	}
}

func writePlanTemplate(path string) error {
	plan := getDefaultPlan()
	return savePlan(path, &plan)
}

func loadPlan(path string) (*PlanData, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var plan PlanData
	if err := yaml.Unmarshal(data, &plan); err != nil {
		return nil, err
	}

	return &plan, nil
}

func savePlan(path string, plan *PlanData) error {
	data, err := small.MarshalYAMLWithQuotedVersion(plan)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

func generateNextTaskID(tasks []PlanTask) string {
	maxNum := 0
	taskIDPattern := regexp.MustCompile(`^task-(\d+)$`)

	for _, task := range tasks {
		matches := taskIDPattern.FindStringSubmatch(task.ID)
		if len(matches) == 2 {
			if num, err := strconv.Atoi(matches[1]); err == nil && num > maxNum {
				maxNum = num
			}
		}
	}

	return fmt.Sprintf("task-%d", maxNum+1)
}

func findTask(plan *PlanData, taskID string) (*PlanTask, int) {
	for i := range plan.Tasks {
		if plan.Tasks[i].ID == taskID {
			return &plan.Tasks[i], i
		}
	}
	return nil, -1
}

func setTaskStatus(plan *PlanData, taskID, status string) error {
	task, _ := findTask(plan, taskID)
	if task == nil {
		return fmt.Errorf("task %s not found", taskID)
	}
	task.Status = status
	return nil
}

func addDependency(plan *PlanData, taskID, depID string) error {
	task, _ := findTask(plan, taskID)
	if task == nil {
		return fmt.Errorf("task %s not found", taskID)
	}

	// Verify dependency task exists
	depTask, _ := findTask(plan, depID)
	if depTask == nil {
		return fmt.Errorf("dependency task %s not found", depID)
	}

	// Check for self-dependency
	if taskID == depID {
		return fmt.Errorf("task cannot depend on itself")
	}

	// Check if dependency already exists
	for _, existingDep := range task.Dependencies {
		if existingDep == depID {
			return fmt.Errorf("dependency %s already exists for task %s", depID, taskID)
		}
	}

	task.Dependencies = append(task.Dependencies, depID)
	return nil
}

func ensureProgressEvidence(artifactsDir, taskID string) error {
	progressPath := filepath.Join(artifactsDir, small.SmallDir, "progress.small.yml")
	var data map[string]interface{}

	if !small.ArtifactExists(artifactsDir, "progress.small.yml") {
		progress := ProgressData{
			SmallVersion: small.ProtocolVersion,
			Owner:        "agent",
			Entries:      []map[string]interface{}{},
		}
		yamlData, err := small.MarshalYAMLWithQuotedVersion(&progress)
		if err != nil {
			return err
		}
		if err := os.WriteFile(progressPath, yamlData, 0o644); err != nil {
			return err
		}
	}

	raw, err := os.ReadFile(progressPath)
	if err != nil {
		return err
	}
	if err := yaml.Unmarshal(raw, &data); err != nil {
		return err
	}

	entries, _ := data["entries"].([]interface{})
	if entries == nil {
		entries = []interface{}{}
	}
	for _, entry := range entries {
		if entryMap, ok := entry.(map[string]interface{}); ok {
			if id, _ := entryMap["task_id"].(string); id == taskID && small.ProgressEntryHasValidEvidence(entryMap) {
				return nil
			}
		}
	}

	entry := map[string]interface{}{
		"task_id":   taskID,
		"status":    "completed",
		"timestamp": formatProgressTimestamp(time.Now().UTC()),
		"evidence":  fmt.Sprintf("Recorded completion via small plan --done %s", taskID),
		"notes":     planDoneProgressNote,
	}

	return appendProgressEntry(artifactsDir, entry)
}

func appendPlanProgress(artifactsDir, taskID, status, evidence, notes string) error {
	entry := map[string]interface{}{
		"task_id":   taskID,
		"timestamp": formatProgressTimestamp(time.Now().UTC()),
		"evidence":  evidence,
	}
	if status != "" {
		entry["status"] = status
	}
	if strings.TrimSpace(notes) != "" {
		entry["notes"] = notes
	}

	return appendProgressEntry(artifactsDir, entry)
}
