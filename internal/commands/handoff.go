package commands

import (
	"fmt"
	"sort"
	"time"

	"github.com/justyn-clark/agent-legible-cms-spec/internal/small"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

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

			handoff := make(map[string]interface{})
			handoff["small_version"] = "0.1"
			handoff["generated_at"] = time.Now().UTC().Format(time.RFC3339)

			currentPlan := make(map[string]interface{})
			if genAt, ok := planArtifact.Data["generated_at"].(string); ok {
				currentPlan["generated_at"] = genAt
			}
			if tasks, ok := planArtifact.Data["tasks"].([]interface{}); ok {
				// Sort tasks by id for deterministic output
				sortedTasks := make([]interface{}, len(tasks))
				copy(sortedTasks, tasks)
				sort.Slice(sortedTasks, func(i, j int) bool {
					taskI, okI := sortedTasks[i].(map[string]interface{})
					taskJ, okJ := sortedTasks[j].(map[string]interface{})
					if !okI || !okJ {
						return false
					}
					idI, _ := taskI["id"].(string)
					idJ, _ := taskJ["id"].(string)
					return idI < idJ
				})
				currentPlan["tasks"] = sortedTasks
			}
			handoff["current_plan"] = currentPlan

			var recentProgress []interface{}
			if entries, ok := progressArtifact.Data["entries"].([]interface{}); ok {
				// Sort entries by timestamp for deterministic output
				sortedEntries := make([]interface{}, len(entries))
				copy(sortedEntries, entries)
				sort.Slice(sortedEntries, func(i, j int) bool {
					entryI, okI := sortedEntries[i].(map[string]interface{})
					entryJ, okJ := sortedEntries[j].(map[string]interface{})
					if !okI || !okJ {
						return false
					}
					tsI, _ := entryI["timestamp"].(string)
					tsJ, _ := entryJ["timestamp"].(string)
					return tsI < tsJ
				})

				start := len(sortedEntries) - recent
				if start < 0 {
					start = 0
				}
				recentProgress = sortedEntries[start:]
			}
			handoff["recent_progress"] = recentProgress

			handoffData, err := marshalDeterministic(handoff)
			if err != nil {
				return fmt.Errorf("failed to marshal handoff: %w", err)
			}

			if err := small.SaveArtifact(artifactsDir, "handoff.small.yml", handoff); err != nil {
				return fmt.Errorf("failed to save handoff.small.yml: %w", err)
			}

			fmt.Printf("Generated handoff.small.yml with %d recent progress entries\n", len(recentProgress))
			fmt.Println(string(handoffData))
			return nil
		},
	}

	cmd.Flags().IntVar(&recent, "recent", 10, "Number of recent progress entries to include")
	cmd.Flags().StringVar(&dir, "dir", ".", "Directory containing .small/ artifacts")

	return cmd
}

func marshalDeterministic(data map[string]interface{}) ([]byte, error) {
	// Create a new map with sorted keys for deterministic output
	ordered := make(map[string]interface{})
	keys := make([]string, 0, len(data))
	for k := range data {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		ordered[k] = data[k]
	}

	return yaml.Marshal(ordered)
}
