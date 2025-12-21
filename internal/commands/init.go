package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

func initCmd() *cobra.Command {
	var force bool
	var projectName string

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize a new SMALL project",
		Long:  "Creates .small/ directory and all five canonical files from templates.",
		RunE: func(cmd *cobra.Command, args []string) error {
			smallDir := filepath.Join(baseDir, ".small")

			if !force {
				if _, err := os.Stat(smallDir); err == nil {
					return fmt.Errorf(".small/ directory already exists. Use --force to overwrite")
				}
			}

			if err := os.MkdirAll(smallDir, 0755); err != nil {
				return fmt.Errorf("failed to create .small directory: %w", err)
			}

			templates := map[string]string{
				"intent.small.yml":      intentTemplate,
				"constraints.small.yml": constraintsTemplate,
				"plan.small.yml":        planTemplate,
				"progress.small.yml":    progressTemplate,
				"handoff.small.yml":     handoffTemplate,
			}

			for filename, template := range templates {
				content := template

				if filename == "intent.small.yml" && projectName != "" {
					var data map[string]interface{}
					if err := yaml.Unmarshal([]byte(template), &data); err == nil {
						data["project_name"] = projectName
						updated, err := yaml.Marshal(data)
						if err == nil {
							content = string(updated)
						}
					}
				}

				path := filepath.Join(smallDir, filename)
				if err := os.WriteFile(path, []byte(content), 0644); err != nil {
					return fmt.Errorf("failed to write %s: %w", path, err)
				}
			}

			fmt.Printf("Initialized SMALL project in %s\n", smallDir)
			return nil
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Overwrite existing .small/ directory")
	cmd.Flags().StringVar(&projectName, "name", "", "Project name to seed in intent.small.yml")

	return cmd
}
