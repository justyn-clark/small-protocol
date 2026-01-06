package commands

import (
	"fmt"
	"os"

	"github.com/justyn-clark/small-protocol/internal/small"
	"github.com/spf13/cobra"
)

func validateCmd() *cobra.Command {
	var dir string
	var specDir string

	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate all canonical SMALL artifacts",
		Long: `Loads and validates all five canonical files against their JSON schemas.

Schema Resolution:
  1. If --spec-dir is set, load schemas from that directory
  2. Else if on-disk schemas found (dev mode in small-protocol repo), use those
  3. Else use embedded v1.0.0 schemas (default for installed CLI)

The embedded schemas allow validation to work in any repository without
requiring the small-protocol spec directory to be present.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if dir == "" {
				dir = baseDir
			}

			artifactsDir := resolveArtifactsDir(dir)

			artifacts, err := small.LoadAllArtifacts(artifactsDir)
			if err != nil {
				return fmt.Errorf("failed to load artifacts: %w", err)
			}

			// Build schema config
			config := small.SchemaConfig{
				SpecDir: specDir,
				BaseDir: artifactsDir,
			}

			// Show which schemas are being used (verbose info)
			if cmd.Flags().Changed("spec-dir") || os.Getenv("SMALL_SPEC_DIR") != "" {
				fmt.Printf("Schema resolution: %s\n", small.DescribeSchemaResolution(config))
			}

			errors := small.ValidateAllArtifactsWithConfig(artifacts, config)
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
	cmd.Flags().StringVar(&specDir, "spec-dir", os.Getenv("SMALL_SPEC_DIR"),
		"Directory containing spec/ (e.g., path/to/small-protocol). Falls back to $SMALL_SPEC_DIR")

	return cmd
}
