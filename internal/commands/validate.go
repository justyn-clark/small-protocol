package commands

import (
	"fmt"
	"os"

	"github.com/justyn-clark/agent-legible-cms-spec/internal/small"
	"github.com/spf13/cobra"
)

func validateCmd() *cobra.Command {
	var dir string

	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate all canonical SMALL artifacts",
		Long:  "Loads and validates all five canonical files against their JSON schemas.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if dir == "" {
				dir = baseDir
			}

			artifactsDir := resolveArtifactsDir(dir)
			repoRoot, err := findRepoRoot(artifactsDir)
			if err != nil {
				return err
			}

			artifacts, err := small.LoadAllArtifacts(artifactsDir)
			if err != nil {
				return fmt.Errorf("failed to load artifacts: %w", err)
			}

			errors := small.ValidateAllArtifacts(artifacts, repoRoot)
			if len(errors) > 0 {
				fmt.Fprintf(os.Stderr, "Validation failed:\n")
				for _, err := range errors {
					fmt.Fprintf(os.Stderr, "  %v\n", err)
				}
				os.Exit(1)
			}

			fmt.Println("All artifacts are valid")
			return nil
		},
	}

	cmd.Flags().StringVar(&dir, "dir", ".", "Directory containing .small/ artifacts")

	return cmd
}
