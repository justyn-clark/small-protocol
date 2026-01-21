package commands

import (
	"fmt"
	"os"

	"github.com/justyn-clark/small-protocol/internal/small"
	"github.com/spf13/cobra"
)

func lintCmd() *cobra.Command {
	var strict bool
	var dir string
	var specDir string
	var formatStrict bool

	cmd := &cobra.Command{
		Use:   "lint",
		Short: "Lint SMALL artifacts for invariant violations",
		Long: `Checks invariants beyond schema validation (version, ownership, evidence, secrets).

Also warns when small_version is not a quoted string. Fix with: small fix --versions
Use --format-strict to treat formatting drift as an error.

Schema Resolution (for any validation performed):
  1. If --spec-dir is set, load schemas from that directory
  2. Else if on-disk schemas found (dev mode in small-protocol repo), use those
  3. Else use embedded v1.0.0 schemas (default for installed CLI)`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if dir == "" {
				dir = baseDir
			}

			artifactsDir := resolveArtifactsDir(dir)
			violations, err := runLintArtifacts(artifactsDir, strict)
			if err != nil {
				return fmt.Errorf("failed to load artifacts: %w", err)
			}

			if len(violations) > 0 {
				fmt.Fprintf(os.Stderr, "Invariant violations found:\n")
				for _, violation := range violations {
					fmt.Fprintf(os.Stderr, "  %s: %s\n", violation.File, violation.Message)
				}
				os.Exit(1)
			}

			warnings, err := findVersionFormatWarnings(artifactsDir)
			if err != nil {
				return err
			}
			if len(warnings) > 0 {
				for _, warning := range warnings {
					fmt.Fprintf(os.Stderr, "WARN %s: small_version should be a quoted string. Fix: small fix --versions\n", warning)
				}
				if formatStrict {
					os.Exit(1)
				}
			}

			fmt.Println("All invariants satisfied")
			return nil
		},
	}

	cmd.Flags().BoolVar(&strict, "strict", false, "Enable strict mode (strict invariants, secrets, insecure links)")
	cmd.Flags().BoolVar(&formatStrict, "format-strict", false, "Treat small_version formatting drift as an error")
	cmd.Flags().StringVar(&dir, "dir", ".", "Directory containing .small/ artifacts")
	cmd.Flags().StringVar(&specDir, "spec-dir", os.Getenv("SMALL_SPEC_DIR"),
		"Directory containing spec/ (e.g., path/to/small-protocol). Falls back to $SMALL_SPEC_DIR")
	// Mark as unused for now since lint doesn't do schema validation
	_ = specDir

	return cmd
}

func runLintArtifacts(baseDir string, strict bool) ([]small.InvariantViolation, error) {
	artifacts, err := small.LoadAllArtifacts(baseDir)
	if err != nil {
		return nil, err
	}
	violations := small.CheckInvariants(artifacts, strict)
	return violations, nil
}
