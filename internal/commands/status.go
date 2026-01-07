package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/justyn-clark/small-protocol/internal/small"
	"github.com/justyn-clark/small-protocol/internal/version"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// StatusOutput represents the structured status output
type StatusOutput struct {
	Version        string           `json:"version"`
	SmallDirExists bool             `json:"small_dir_exists"`
	Artifacts      ArtifactPresence `json:"artifacts"`
	Plan           *PlanStatus      `json:"plan,omitempty"`
	RecentProgress []ProgressEntry  `json:"recent_progress,omitempty"`
	LastHandoff    string           `json:"last_handoff,omitempty"`
}

// ArtifactPresence shows which artifacts exist
type ArtifactPresence struct {
	Intent      bool `json:"intent"`
	Constraints bool `json:"constraints"`
	Plan        bool `json:"plan"`
	Progress    bool `json:"progress"`
	Handoff     bool `json:"handoff"`
}

// PlanStatus summarizes plan state
type PlanStatus struct {
	TotalTasks     int            `json:"total_tasks"`
	TasksByStatus  map[string]int `json:"tasks_by_status"`
	NextActionable []string       `json:"next_actionable"`
}

// ProgressEntry represents a progress entry for status output
type ProgressEntry struct {
	Timestamp string `json:"timestamp"`
	TaskID    string `json:"task_id"`
	Status    string `json:"status"`
	Notes     string `json:"notes,omitempty"`
}

func statusCmd() *cobra.Command {
	var (
		jsonOutput bool
		recent     int
		tasks      int
		dir        string
	)

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show SMALL project status summary",
		Long:  "Displays a compact summary of the current SMALL project state.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if dir == "" {
				dir = baseDir
			}
			artifactsDir := resolveArtifactsDir(dir)
			smallDir := filepath.Join(artifactsDir, small.SmallDir)

			status := StatusOutput{
				Version: version.GetVersion(),
			}

			// Check if .small directory exists
			if _, err := os.Stat(smallDir); os.IsNotExist(err) {
				status.SmallDirExists = false
				if jsonOutput {
					return outputJSON(status)
				}
				fmt.Printf("small %s\n", version.GetVersion())
				fmt.Println("\n.small/ directory does not exist")
				fmt.Println("Run 'small init' to create a SMALL project")
				return nil
			}
			status.SmallDirExists = true

			// Check artifact presence
			status.Artifacts = ArtifactPresence{
				Intent:      small.ArtifactExists(artifactsDir, "intent.small.yml"),
				Constraints: small.ArtifactExists(artifactsDir, "constraints.small.yml"),
				Plan:        small.ArtifactExists(artifactsDir, "plan.small.yml"),
				Progress:    small.ArtifactExists(artifactsDir, "progress.small.yml"),
				Handoff:     small.ArtifactExists(artifactsDir, "handoff.small.yml"),
			}

			// Load and analyze plan if it exists
			if status.Artifacts.Plan {
				planStatus, err := analyzePlan(artifactsDir, tasks)
				if err == nil {
					status.Plan = planStatus
				}
			}

			// Load recent progress entries
			if status.Artifacts.Progress {
				entries, err := getRecentProgress(artifactsDir, recent)
				if err == nil {
					status.RecentProgress = entries
				}
			}

			// Get last handoff timestamp
			if status.Artifacts.Handoff {
				timestamp, err := getHandoffTimestamp(artifactsDir)
				if err == nil && timestamp != "" {
					status.LastHandoff = timestamp
				}
			}

			if jsonOutput {
				return outputJSON(status)
			}

			return outputText(status)
		},
	}

	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output in JSON format")
	cmd.Flags().IntVar(&recent, "recent", 5, "Number of recent progress entries to show")
	cmd.Flags().IntVar(&tasks, "tasks", 3, "Number of next actionable tasks to show")
	cmd.Flags().StringVar(&dir, "dir", ".", "Directory containing .small/ artifacts")

	return cmd
}

func analyzePlan(baseDir string, maxActionable int) (*PlanStatus, error) {
	artifact, err := small.LoadArtifact(baseDir, "plan.small.yml")
	if err != nil {
		return nil, err
	}

	var plan PlanData
	yamlData, err := yaml.Marshal(artifact.Data)
	if err != nil {
		return nil, err
	}
	if err := yaml.Unmarshal(yamlData, &plan); err != nil {
		return nil, err
	}

	status := &PlanStatus{
		TotalTasks:     len(plan.Tasks),
		TasksByStatus:  make(map[string]int),
		NextActionable: []string{},
	}

	// Build a map of task statuses for dependency checking
	taskStatuses := make(map[string]string)
	for _, task := range plan.Tasks {
		taskStatuses[task.ID] = task.Status
		status.TasksByStatus[task.Status]++
	}

	// Find actionable tasks (pending with all deps satisfied)
	for _, task := range plan.Tasks {
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

		if depsSatisfied && len(status.NextActionable) < maxActionable {
			status.NextActionable = append(status.NextActionable, task.ID)
		}
	}

	return status, nil
}

func getRecentProgress(baseDir string, n int) ([]ProgressEntry, error) {
	artifact, err := small.LoadArtifact(baseDir, "progress.small.yml")
	if err != nil {
		return nil, err
	}

	entries, ok := artifact.Data["entries"].([]interface{})
	if !ok {
		return []ProgressEntry{}, nil
	}

	var progressEntries []ProgressEntry
	for _, e := range entries {
		m, ok := e.(map[string]interface{})
		if !ok {
			continue
		}

		entry := ProgressEntry{
			Timestamp: stringVal(m["timestamp"]),
			TaskID:    stringVal(m["task_id"]),
			Status:    stringVal(m["status"]),
		}
		if notes, ok := m["notes"].(string); ok {
			entry.Notes = notes
		}
		progressEntries = append(progressEntries, entry)
	}

	// Sort by timestamp descending (most recent first)
	sort.Slice(progressEntries, func(i, j int) bool {
		return progressEntries[i].Timestamp > progressEntries[j].Timestamp
	})

	// Return only the most recent N entries
	if n > 0 && len(progressEntries) > n {
		progressEntries = progressEntries[:n]
	}

	return progressEntries, nil
}

func getHandoffTimestamp(baseDir string) (string, error) {
	artifact, err := small.LoadArtifact(baseDir, "handoff.small.yml")
	if err != nil {
		return "", err
	}

	if timestamp, ok := artifact.Data["generated_at"].(string); ok {
		return timestamp, nil
	}

	return "", nil
}

func outputJSON(status StatusOutput) error {
	data, err := json.MarshalIndent(status, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}

func outputText(status StatusOutput) error {
	fmt.Printf("small v%s\n", status.Version)
	fmt.Println()

	// Artifact checklist
	fmt.Println("Artifacts:")
	printArtifactStatus("  intent.small.yml", status.Artifacts.Intent)
	printArtifactStatus("  constraints.small.yml", status.Artifacts.Constraints)
	printArtifactStatus("  plan.small.yml", status.Artifacts.Plan)
	printArtifactStatus("  progress.small.yml", status.Artifacts.Progress)
	printArtifactStatus("  handoff.small.yml", status.Artifacts.Handoff)
	fmt.Println()

	// Plan summary
	if status.Plan != nil {
		fmt.Printf("Plan: %d tasks\n", status.Plan.TotalTasks)
		for statusName, count := range status.Plan.TasksByStatus {
			fmt.Printf("  %s: %d\n", statusName, count)
		}
		if len(status.Plan.NextActionable) > 0 {
			fmt.Printf("Next actionable: %v\n", status.Plan.NextActionable)
		} else {
			fmt.Println("Next actionable: none")
		}
		fmt.Println()
	}

	// Recent progress
	if len(status.RecentProgress) > 0 {
		fmt.Printf("Recent progress (%d entries):\n", len(status.RecentProgress))
		for _, entry := range status.RecentProgress {
			ts := formatTimestamp(entry.Timestamp)
			fmt.Printf("  [%s] %s: %s\n", ts, entry.TaskID, entry.Status)
		}
		fmt.Println()
	}

	// Last handoff
	if status.LastHandoff != "" {
		ts := formatTimestamp(status.LastHandoff)
		fmt.Printf("Last handoff: %s\n", ts)
	}

	return nil
}

func printArtifactStatus(name string, exists bool) {
	if exists {
		fmt.Printf("%s [x]\n", name)
	} else {
		fmt.Printf("%s [ ]\n", name)
	}
}

func formatTimestamp(ts string) string {
	t, err := time.Parse(time.RFC3339, ts)
	if err != nil {
		return ts
	}
	return t.Format("2006-01-02 15:04:05")
}
