package commands

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/justyn-clark/small-protocol/internal/small"
	"github.com/justyn-clark/small-protocol/internal/workspace"
	"github.com/spf13/cobra"
)

func resetCmd() *cobra.Command {
	var yes bool
	var keepIntent bool
	var workspaceFlag string

	cmd := &cobra.Command{
		Use:   "reset",
		Short: "Reset .small/ for a new run while preserving audit history",
		Long: `Cleanly start a new run without destroying audit history.

Ephemeral files reset (recreated from templates):
  - intent.small.yml (unless --keep-intent)
  - plan.small.yml
  - handoff.small.yml

Preserved audit artifacts:
  - progress.small.yml (append-only audit trail)
  - constraints.small.yml (human-owned)`,
		RunE: func(cmd *cobra.Command, args []string) error {
			smallDir := filepath.Join(baseDir, ".small")
			var previousReplayID string
			if small.ArtifactExists(baseDir, "handoff.small.yml") {
				if existing, err := loadExistingHandoff(baseDir); err == nil && existing.ReplayId != nil {
					previousReplayID = existing.ReplayId.Value
				}
			}

			// Check if .small/ directory exists
			if _, err := os.Stat(smallDir); os.IsNotExist(err) {
				return fmt.Errorf(".small/ directory does not exist. Use 'small init' first")
			}

			scope, err := workspace.ParseScope(workspaceFlag)
			if err != nil {
				return err
			}
			if scope == workspace.ScopeExamples {
				return fmt.Errorf("--workspace examples is not supported for reset (use --workspace any to bypass)")
			}
			if scope != workspace.ScopeAny {
				if err := enforceWorkspaceScope(baseDir, workspace.ScopeRoot); err != nil {
					return err
				}
			}

			// Confirm unless --yes flag is provided

			if !yes {
				fmt.Print("This will reset ephemeral .small/ files for a new run.\n")
				fmt.Print("Progress and constraints will be preserved.\n")
				if keepIntent {
					fmt.Print("Intent will be preserved (--keep-intent).\n")
				}
				fmt.Print("\nContinue? [y/N]: ")

				reader := bufio.NewReader(os.Stdin)
				response, err := reader.ReadString('\n')
				if err != nil {
					return fmt.Errorf("failed to read response: %w", err)
				}

				response = strings.TrimSpace(strings.ToLower(response))
				if response != "y" && response != "yes" {
					fmt.Println("Reset cancelled.")
					return nil
				}
			}

			// Define which files to reset vs preserve
			ephemeralFiles := []string{
				"plan.small.yml",
				"handoff.small.yml",
			}

			if !keepIntent {
				ephemeralFiles = append([]string{"intent.small.yml"}, ephemeralFiles...)
			}

			// Templates for reset files
			templates := map[string]string{
				"intent.small.yml": intentTemplate,
				"plan.small.yml":   planTemplate,
			}

			// Reset ephemeral files
			for _, filename := range ephemeralFiles {
				path := filepath.Join(smallDir, filename)
				template, ok := templates[filename]
				if !ok {
					continue
				}

				if err := os.WriteFile(path, []byte(template), 0644); err != nil {
					return fmt.Errorf("failed to reset %s: %w", filename, err)
				}
				fmt.Printf("Reset: %s\n", filename)
			}

			handoffRun := &runOut{
				CreatedAt:        time.Now().UTC().Format(time.RFC3339Nano),
				TransitionReason: "reset",
				PreviousReplayID: previousReplayID,
			}
			handoff, err := buildHandoff(baseDir, "", "", nil, nil, handoffRun, defaultNextStepsLimit)
			if err != nil {
				return err
			}
			if err := setWorkspaceRunReplayIDIfPresent(baseDir, handoff.ReplayId.Value); err != nil {
				return err
			}
			if err := writeHandoff(baseDir, handoff); err != nil {
				return err
			}
			fmt.Println("Reset: handoff.small.yml")

			evidence := "Reset ephemeral .small artifacts"
			notes := "small reset"
			if keepIntent {
				evidence = "Reset plan and handoff (intent preserved)"
				notes = "small reset --keep-intent"
			}
			entry := map[string]any{
				"task_id":   "reset",
				"status":    "completed",
				"timestamp": formatProgressTimestamp(time.Now().UTC()),
				"evidence":  evidence,
				"notes":     notes,
			}
			if err := appendProgressEntry(baseDir, entry); err != nil {
				return fmt.Errorf("failed to record reset progress: %w", err)
			}

			fmt.Printf("\nSMALL v%s reset complete. Ready for new run.\n", small.ProtocolVersion)
			fmt.Println("Preserved: progress.small.yml, constraints.small.yml")
			if keepIntent {
				fmt.Println("Preserved: intent.small.yml (--keep-intent)")
			}

			return nil
		},
	}

	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Non-interactive mode (skip confirmation)")
	cmd.Flags().BoolVar(&keepIntent, "keep-intent", false, "Preserve intent.small.yml")
	cmd.Flags().StringVar(&workspaceFlag, "workspace", string(workspace.ScopeRoot), "Workspace scope (root or any)")

	return cmd
}
