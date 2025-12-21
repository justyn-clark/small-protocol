package commands

import (
	"fmt"
	"os"

	"github.com/agentlegible/small-cli/internal/small"
	"github.com/spf13/cobra"
)

func validateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "validate",
		Short: "Validate all canonical SMALL artifacts",
		Long:  "Loads and validates all five canonical files against their JSON schemas.",
		RunE: func(cmd *cobra.Command, args []string) error {
			repoRoot, err := findRepoRoot()
			if err != nil {
				return err
			}

			artifacts, err := small.LoadAllArtifacts(baseDir)
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
}
