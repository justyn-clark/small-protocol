package commands

import (
	"fmt"
	"os"

	"github.com/justyn-clark/small-protocol/internal/migratev2"
	"github.com/justyn-clark/small-protocol/internal/sessionv2"
	"github.com/spf13/cobra"
)

func migrateCmd() *cobra.Command {
	var dir, to, namespace, mode, applyPath, expected string
	var preview, jsonOutput, recoverOnly bool
	cmd := &cobra.Command{
		Use: "migrate", Short: "Preview or apply an explicit loss-preserving profile migration",
		RunE: func(cmd *cobra.Command, args []string) error {
			if dir == "" {
				dir = baseDir
			}
			dir = resolveArtifactsDir(dir)
			selected := 0
			if preview {
				selected++
			}
			if applyPath != "" {
				selected++
			}
			if recoverOnly {
				selected++
			}
			if selected != 1 {
				return fmt.Errorf("choose exactly one of --preview, --apply, or --recover")
			}
			if recoverOnly {
				if err := migratev2.Recover(dir); err != nil {
					return err
				}
				if jsonOutput {
					return writeJSONValue(map[string]any{"recovered": true})
				}
				fmt.Println("Migration recovery complete")
				return nil
			}
			if preview {
				if to != sessionv2.ProfileVersion {
					return fmt.Errorf("--to must be %s", sessionv2.ProfileVersion)
				}
				plan, err := migratev2.Preview(dir, namespace, mode)
				if err != nil {
					return err
				}
				if jsonOutput {
					return writeJSONValue(plan)
				}
				fmt.Printf("Migration %s -> %s; input %s; mode %s\n", plan.ImportID, plan.To, plan.ExpectedInputDigest, plan.Mode)
				return nil
			}
			if expected == "" {
				return fmt.Errorf("--expect-state is required with --apply")
			}
			data, err := os.ReadFile(applyPath)
			if err != nil {
				return err
			}
			var plan migratev2.Plan
			if err := sessionv2.DecodeStrict(data, &plan); err != nil {
				return fmt.Errorf("migration plan: %w", err)
			}
			if plan.ExpectedInputDigest != expected {
				return fmt.Errorf("expected state does not match reviewed migration plan")
			}
			result, err := migratev2.Apply(dir, plan)
			if err != nil {
				return err
			}
			if jsonOutput {
				return writeJSONValue(result)
			}
			fmt.Printf("Migrated to %s; import %s; frontier %s\n", sessionv2.ProfileVersion, result.ImportID, result.Frontier)
			return nil
		},
	}
	cmd.Flags().StringVar(&dir, "dir", "", "Project directory")
	cmd.Flags().StringVar(&to, "to", sessionv2.ProfileVersion, "Target profile")
	cmd.Flags().StringVar(&namespace, "namespace", "", "Explicit shared import namespace")
	cmd.Flags().StringVar(&mode, "mode", "solo", "Initial mode (solo or collaborative)")
	cmd.Flags().BoolVar(&preview, "preview", false, "Perform a read-only preview")
	cmd.Flags().StringVar(&applyPath, "apply", "", "Apply a reviewed migration plan JSON file")
	cmd.Flags().StringVar(&expected, "expect-state", "", "Required legacy input digest")
	cmd.Flags().BoolVar(&recoverOnly, "recover", false, "Recover an interrupted migration")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output JSON")
	return cmd
}
