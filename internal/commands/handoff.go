package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/justyn-clark/small-protocol/internal/small"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// handoffOut represents the v1.0.0 handoff structure
type handoffOut struct {
	SmallVersion string    `yaml:"small_version"`
	Owner        string    `yaml:"owner"`
	Summary      string    `yaml:"summary"`
	Resume       resumeOut `yaml:"resume"`
	Links        []linkOut `yaml:"links"`
}

type resumeOut struct {
	CurrentTaskID string   `yaml:"current_task_id"`
	NextSteps     []string `yaml:"next_steps"`
}

type linkOut struct {
	URL         string `yaml:"url,omitempty"`
	Description string `yaml:"description,omitempty"`
}

func handoffCmd() *cobra.Command {
	var (
		summary string
		dir     string
	)

	cmd := &cobra.Command{
		Use:   "handoff",
		Short: "Generate or update handoff.small.yml",
		Long:  "Generates handoff.small.yml from current plan with resume information.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if dir == "" {
				dir = baseDir
			}

			artifactsDir := resolveArtifactsDir(dir)

			planArtifact, err := small.LoadArtifact(artifactsDir, "plan.small.yml")
			if err != nil {
				return fmt.Errorf("failed to load plan.small.yml: %w", err)
			}

			// Build next_steps from pending tasks and find current task
			var nextSteps []string
			var currentTaskID string
			if rawTasks, ok := planArtifact.Data["tasks"].([]interface{}); ok {
				for _, t := range rawTasks {
					m, ok := t.(map[string]interface{})
					if !ok {
						continue
					}

					taskID := stringVal(m["id"])
					title := stringVal(m["title"])
					status := stringVal(m["status"])

					// Find the first in_progress task as current
					if status == "in_progress" && currentTaskID == "" {
						currentTaskID = taskID
					}

					// Add pending and in_progress tasks to next_steps
					if status == "pending" || status == "in_progress" || status == "" {
						step := title
						if step == "" {
							step = taskID
						}
						if step != "" {
							nextSteps = append(nextSteps, step)
						}
					}
				}
			}

			// Use provided summary or generate a default one
			handoffSummary := summary
			if handoffSummary == "" {
				handoffSummary = "Handoff generated from current plan state"
			}

			h := handoffOut{
				SmallVersion: small.ProtocolVersion,
				Owner:        "agent",
				Summary:      handoffSummary,
				Resume: resumeOut{
					CurrentTaskID: currentTaskID,
					NextSteps:     nextSteps,
				},
				Links: []linkOut{},
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

			fmt.Printf("Generated handoff.small.yml with %d next steps\n", len(nextSteps))
			fmt.Println(string(yml))
			return nil
		},
	}

	cmd.Flags().StringVar(&summary, "summary", "", "Summary description for the handoff")
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
