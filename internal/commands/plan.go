package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/justyn-clark/small-protocol/internal/small"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// PlanData represents the plan.small.yml structure
type PlanData struct {
	SmallVersion string     `yaml:"small_version"`
	Owner        string     `yaml:"owner"`
	GeneratedAt  string     `yaml:"generated_at,omitempty"`
	Tasks        []PlanTask `yaml:"tasks"`
}

// PlanTask represents a task in the plan
type PlanTask struct {
	ID           string   `yaml:"id"`
	Description  string   `yaml:"description"`
	Status       string   `yaml:"status"`
	Dependencies []string `yaml:"dependencies,omitempty"`
}

func planCmd() *cobra.Command {
	var (
		reset      bool
		yes        bool
		addTask    string
		doneID     string
		pendingID  string
		blockedID  string
		dependsArg string
		dir        string
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
					ID:           newID,
					Description:  addTask,
					Status:       "pending",
					Dependencies: []string{},
				}
				plan.Tasks = append(plan.Tasks, newTask)
				modified = true
				fmt.Printf("Added task %s: %s\n", newID, addTask)
			}

			// Handle --done flag
			if doneID != "" {
				if err := setTaskStatus(&plan, doneID, "completed"); err != nil {
					return err
				}
				modified = true
				fmt.Printf("Marked task %s as completed\n", doneID)
			}

			// Handle --pending flag
			if pendingID != "" {
				if err := setTaskStatus(&plan, pendingID, "pending"); err != nil {
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
	cmd.Flags().StringVar(&addTask, "add", "", "Add a new task with the given description")
	cmd.Flags().StringVar(&doneID, "done", "", "Mark task as completed by ID")
	cmd.Flags().StringVar(&pendingID, "pending", "", "Mark task as pending by ID")
	cmd.Flags().StringVar(&blockedID, "blocked", "", "Mark task as blocked by ID")
	cmd.Flags().StringVar(&dependsArg, "depends", "", "Add dependency edge (format: <task-id>:<dep-id>)")
	cmd.Flags().StringVar(&dir, "dir", ".", "Directory containing .small/ artifacts")

	return cmd
}

func getDefaultPlan() PlanData {
	return PlanData{
		SmallVersion: "0.1",
		Owner:        "agent",
		Tasks: []PlanTask{
			{
				ID:           "task-1",
				Description:  "Define project intent and constraints",
				Status:       "pending",
				Dependencies: []string{},
			},
			{
				ID:           "task-2",
				Description:  "Validate SMALL artifacts against schemas",
				Status:       "pending",
				Dependencies: []string{"task-1"},
			},
			{
				ID:           "task-3",
				Description:  "Generate handoff for agent resume",
				Status:       "pending",
				Dependencies: []string{"task-2"},
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
	// Ensure empty dependencies are represented as empty arrays, not null
	for i := range plan.Tasks {
		if plan.Tasks[i].Dependencies == nil {
			plan.Tasks[i].Dependencies = []string{}
		}
	}

	data, err := yaml.Marshal(plan)
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
