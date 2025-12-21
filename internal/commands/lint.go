package commands

import (
	"fmt"
	"os"

	"github.com/justyn-clark/agent-legible-cms-spec/internal/small"
	"github.com/spf13/cobra"
)

func lintCmd() *cobra.Command {
	var strict bool
	var dir string

	cmd := &cobra.Command{
		Use:   "lint",
		Short: "Lint SMALL artifacts for invariant violations",
		Long:  "Checks invariants beyond schema validation (version, ownership, evidence, secrets).",
		RunE: func(cmd *cobra.Command, args []string) error {
			if dir == "" {
				dir = baseDir
			}

			artifactsDir := resolveArtifactsDir(dir)
			artifacts, err := small.LoadAllArtifacts(artifactsDir)
			if err != nil {
				return fmt.Errorf("failed to load artifacts: %w", err)
			}

			violations := small.CheckInvariants(artifacts, strict)
			if len(violations) > 0 {
				fmt.Fprintf(os.Stderr, "Invariant violations found:\n")
				for _, violation := range violations {
					fmt.Fprintf(os.Stderr, "  %s: %s\n", violation.File, violation.Message)
				}
				os.Exit(1)
			}

			fmt.Println("All invariants satisfied")
			return nil
		},
	}

	cmd.Flags().BoolVar(&strict, "strict", false, "Enable strict mode (includes secret detection)")
	cmd.Flags().StringVar(&dir, "dir", ".", "Directory containing .small/ artifacts")

	return cmd
}
