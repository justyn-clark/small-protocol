package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/justyn-clark/small-protocol/internal/small"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

type handoffOut struct {
	SmallVersion   string         `yaml:"small_version"`
	GeneratedAt    string         `yaml:"generated_at"`
	CurrentPlan    currentPlanOut `yaml:"current_plan"`
	RecentProgress []progressOut  `yaml:"recent_progress,omitempty"`
	NextSteps      []string       `yaml:"next_steps,omitempty"`
}

type currentPlanOut struct {
	GeneratedAt string     `yaml:"generated_at,omitempty"`
	Tasks       []planTask `yaml:"tasks"`
}

type planTask struct {
	ID           string   `yaml:"id"`
	Description  string   `yaml:"description"`
	Status       string   `yaml:"status"`
	Dependencies []string `yaml:"dependencies,omitempty"`
}

type progressOut struct {
	Timestamp    string      `yaml:"timestamp"`
	TaskID       string      `yaml:"task_id"`
	Status       string      `yaml:"status"`
	Evidence     interface{} `yaml:"evidence,omitempty"`
	Verification interface{} `yaml:"verification,omitempty"`
	Command      string      `yaml:"command,omitempty"`
	Test         interface{} `yaml:"test,omitempty"`
	Link         string      `yaml:"link,omitempty"`
	Commit       string      `yaml:"commit,omitempty"`
	Notes        string      `yaml:"notes,omitempty"`
}

func handoffCmd() *cobra.Command {
	var recent int
	var dir string

	cmd := &cobra.Command{
		Use:   "handoff",
		Short: "Generate or update handoff.small.yml",
		Long:  "Generates handoff.small.yml from current plan and recent progress entries.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if dir == "" {
				dir = baseDir
			}

			artifactsDir := resolveArtifactsDir(dir)

			if recent <= 0 {
				recent = 10
			}

			planArtifact, err := small.LoadArtifact(artifactsDir, "plan.small.yml")
			if err != nil {
				return fmt.Errorf("failed to load plan.small.yml: %w", err)
			}

			progressArtifact, err := small.LoadArtifact(artifactsDir, "progress.small.yml")
			if err != nil {
				return fmt.Errorf("failed to load progress.small.yml: %w", err)
			}

			// ---- Build current_plan (deterministic order) ----
			var tasks []planTask
			if rawTasks, ok := planArtifact.Data["tasks"].([]interface{}); ok {
				for _, t := range rawTasks {
					m, ok := t.(map[string]interface{})
					if !ok {
						continue
					}

					task := planTask{
						ID:          stringVal(m["id"]),
						Description: stringVal(m["description"]),
						Status:      stringVal(m["status"]),
					}

					if deps, ok := m["dependencies"].([]interface{}); ok && len(deps) > 0 {
						task.Dependencies = make([]string, 0, len(deps))
						for _, d := range deps {
							if s, ok := d.(string); ok && s != "" {
								task.Dependencies = append(task.Dependencies, s)
							}
						}
						if len(task.Dependencies) == 0 {
							task.Dependencies = nil
						}
					}

					tasks = append(tasks, task)
				}
			}

			sort.Slice(tasks, func(i, j int) bool { return tasks[i].ID < tasks[j].ID })

			cp := currentPlanOut{Tasks: tasks}
			if genAt, ok := planArtifact.Data["generated_at"].(string); ok && genAt != "" {
				cp.GeneratedAt = genAt
			}

			// ---- Build recent_progress (deterministic by timestamp) ----
			var entries []progressOut
			if rawEntries, ok := progressArtifact.Data["entries"].([]interface{}); ok {
				for _, e := range rawEntries {
					m, ok := e.(map[string]interface{})
					if !ok {
						continue
					}
					entries = append(entries, progressOut{
						Timestamp:    stringVal(m["timestamp"]),
						TaskID:       stringVal(m["task_id"]),
						Status:       stringVal(m["status"]),
						Evidence:     m["evidence"],
						Verification: m["verification"],
						Command:      stringVal(m["command"]),
						Test:         m["test"],
						Link:         stringVal(m["link"]),
						Commit:       stringVal(m["commit"]),
						Notes:        stringVal(m["notes"]),
					})
				}
			}

			sort.Slice(entries, func(i, j int) bool { return entries[i].Timestamp < entries[j].Timestamp })

			if recent > 0 && len(entries) > recent {
				entries = entries[len(entries)-recent:]
			}

			h := handoffOut{
				SmallVersion: "0.1",
				GeneratedAt:  time.Now().UTC().Format(time.RFC3339),
				CurrentPlan:  cp,
			}
			if len(entries) > 0 {
				h.RecentProgress = entries
			}

			yml, err := yaml.Marshal(h)
			if err != nil {
				return fmt.Errorf("failed to marshal handoff: %w", err)
			}

			// Write to .small/handoff.small.yml
			smallDir := filepath.Join(artifactsDir, small.SmallDir)
			if err := os.MkdirAll(smallDir, 0755); err != nil {
				return fmt.Errorf("failed to create .small directory: %w", err)
			}
			outPath := filepath.Join(smallDir, "handoff.small.yml")
			if err := os.WriteFile(outPath, yml, 0644); err != nil {
				return fmt.Errorf("failed to write %s: %w", outPath, err)
			}

			fmt.Printf("Generated handoff.small.yml with %d recent progress entries\n", len(entries))
			fmt.Println(string(yml))
			return nil
		},
	}

	cmd.Flags().IntVar(&recent, "recent", 10, "Number of recent progress entries to include")
	cmd.Flags().StringVar(&dir, "dir", ".", "Directory containing .small/ artifacts")

	return cmd
}

func stringVal(v interface{}) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", v)
}
