package commands

import (
	"fmt"
	"sort"
	"time"

	"github.com/agentlegible/small-cli/internal/small"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

func handoffCmd() *cobra.Command {
	var fromProgress int

	cmd := &cobra.Command{
		Use:   "handoff",
		Short: "Generate or update handoff.small.yml",
		Long:  "Generates handoff.small.yml from current plan and recent progress entries.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if fromProgress <= 0 {
				fromProgress = 10
			}

			planArtifact, err := small.LoadArtifact(baseDir, "plan.small.yml")
			if err != nil {
				return fmt.Errorf("failed to load plan.small.yml: %w", err)
			}

			progressArtifact, err := small.LoadArtifact(baseDir, "progress.small.yml")
			if err != nil {
				return fmt.Errorf("failed to load progress.small.yml: %w", err)
			}

			handoff := make(map[string]interface{})
			handoff["small_version"] = "0.1"
			handoff["generated_at"] = time.Now().UTC().Format(time.RFC3339)

			currentPlan := make(map[string]interface{})
			if genAt, ok := planArtifact.Data["generated_at"].(string); ok {
				currentPlan["generated_at"] = genAt
			}
			if tasks, ok := planArtifact.Data["tasks"].([]interface{}); ok {
				currentPlan["tasks"] = tasks
			}
			handoff["current_plan"] = currentPlan

			var recentProgress []interface{}
			if entries, ok := progressArtifact.Data["entries"].([]interface{}); ok {
				start := len(entries) - fromProgress
				if start < 0 {
					start = 0
				}
				recentProgress = entries[start:]
			}
			handoff["recent_progress"] = recentProgress

			handoffData, err := marshalDeterministic(handoff)
			if err != nil {
				return fmt.Errorf("failed to marshal handoff: %w", err)
			}

			if err := small.SaveArtifact(baseDir, "handoff.small.yml", handoff); err != nil {
				return fmt.Errorf("failed to save handoff.small.yml: %w", err)
			}

			fmt.Printf("Generated handoff.small.yml with %d recent progress entries\n", len(recentProgress))
			fmt.Println(string(handoffData))
			return nil
		},
	}

	cmd.Flags().IntVar(&fromProgress, "from-progress", 10, "Number of recent progress entries to include")

	return cmd
}

func marshalDeterministic(data map[string]interface{}) ([]byte, error) {
	keys := make([]string, 0, len(data))
	for k := range data {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	ordered := make([]interface{}, 0, len(keys))
	for _, k := range keys {
		ordered = append(ordered, map[string]interface{}{k: data[k]})
	}

	return yaml.Marshal(data)
}
