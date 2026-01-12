package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/justyn-clark/small-protocol/internal/small"
	"github.com/justyn-clark/small-protocol/internal/workspace"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

func initCmd() *cobra.Command {
	var force bool
	var intentStr string
	var dir string

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize a new SMALL v" + small.ProtocolVersion + " project",
		Long:  "Creates .small/ directory and all five canonical files from templates.",
		RunE: func(cmd *cobra.Command, args []string) error {
			targetDir := baseDir
			if dir != "" {
				targetDir = dir
			}
			targetDir = resolveArtifactsDir(targetDir)

			smallDir := filepath.Join(targetDir, small.SmallDir)

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

				// Seed intent if provided
				if filename == "intent.small.yml" && strings.TrimSpace(intentStr) != "" {
					var data map[string]interface{}
					if err := yaml.Unmarshal([]byte(template), &data); err == nil {
						data["intent"] = intentStr
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

			if err := workspace.Save(targetDir, workspace.KindRepoRoot); err != nil {
				return err
			}

			fmt.Printf("Initialized SMALL v%s project in %s\n", small.ProtocolVersion, smallDir)
			return nil
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Overwrite existing .small/ directory")
	cmd.Flags().StringVar(&intentStr, "intent", "", "Intent string to seed in intent.small.yml")
	cmd.Flags().StringVar(&dir, "dir", "", "Target directory for the new workspace (default: current working directory)")

	return cmd
}
