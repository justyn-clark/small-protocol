package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/justyn-clark/small-protocol/internal/small"
	"github.com/justyn-clark/small-protocol/internal/workspace"
	"github.com/spf13/cobra"
)

func startCmd() *cobra.Command {
	var (
		strict        bool
		summary       string
		dir           string
		workspaceFlag string
	)

	cmd := &cobra.Command{
		Use:   "start",
		Short: "Initialize or repair run handoff state",
		RunE: func(cmd *cobra.Command, args []string) error {
			if dir == "" {
				dir = baseDir
			}
			artifactsDir := resolveArtifactsDir(dir)

			scope, err := workspace.ParseScope(workspaceFlag)
			if err != nil {
				return err
			}
			if scope == workspace.ScopeExamples {
				return fmt.Errorf("--workspace examples is not supported for start (use --workspace any to bypass)")
			}
			if scope != workspace.ScopeAny {
				if err := enforceWorkspaceScope(artifactsDir, workspace.ScopeRoot); err != nil {
					return err
				}
			}

			smallDir := filepath.Join(artifactsDir, small.SmallDir)
			if err := os.MkdirAll(smallDir, 0755); err != nil {
				return fmt.Errorf("failed to create .small directory: %w", err)
			}

			if err := ensureStartArtifacts(artifactsDir); err != nil {
				return err
			}

			existing, existingErr := loadExistingHandoff(artifactsDir)
			if existingErr != nil && !os.IsNotExist(existingErr) {
				return existingErr
			}

			handoffSummary := summary
			if handoffSummary == "" && existing != nil {
				handoffSummary = existing.Summary
			}

			replayId := (*replayIdOut)(nil)
			if existing != nil {
				replayId = existing.ReplayId
			}

			selfHeal := false
			if replayId == nil {
				valid := validateIntentAndPlan(artifactsDir)
				if valid {
					selfHeal = true
				}
			}

			transitionReason := "manual"
			if selfHeal {
				transitionReason = "self_heal"
			}

			handoffRun := &runOut{
				CreatedAt:        time.Now().UTC().Format(time.RFC3339Nano),
				TransitionReason: transitionReason,
			}

			handoff, err := buildHandoff(artifactsDir, handoffSummary, "", existingLinks(existing), replayId, handoffRun, defaultNextStepsLimit)
			if err != nil {
				return err
			}

			if err := writeHandoff(artifactsDir, handoff); err != nil {
				return err
			}

			if selfHeal {
				entry := map[string]interface{}{
					"task_id":   "meta/replayid-self-heal",
					"status":    "completed",
					"timestamp": formatProgressTimestamp(time.Now().UTC()),
					"evidence":  "Generated replayId for handoff during small start",
					"notes":     "small start self-heal",
				}
				if err := appendProgressEntry(artifactsDir, entry); err != nil {
					return fmt.Errorf("failed to record replayId self-heal: %w", err)
				}
			}

			code, output, err := runCheck(artifactsDir, strict, false, false, scope, false)
			if err != nil {
				return err
			}
			if code != ExitValid {
				return fmt.Errorf("check failed (validate=%s lint=%s verify=%s)", output.Validate.Status, output.Lint.Status, output.Verify.Status)
			}

			fmt.Println("Start complete. Handoff ready.")
			return nil
		},
	}

	cmd.Flags().BoolVar(&strict, "strict", false, "Enable strict mode checks")
	cmd.Flags().StringVar(&summary, "summary", "", "Summary description for the handoff")
	cmd.Flags().StringVar(&dir, "dir", ".", "Directory containing .small/ artifacts")
	cmd.Flags().StringVar(&workspaceFlag, "workspace", string(workspace.ScopeRoot), "Workspace scope (root or any)")

	return cmd
}

func ensureStartArtifacts(artifactsDir string) error {
	templates := map[string]string{
		"intent.small.yml":      intentTemplate,
		"constraints.small.yml": constraintsTemplate,
		"plan.small.yml":        planTemplate,
		"progress.small.yml":    progressTemplate,
	}

	for filename, template := range templates {
		if small.ArtifactExists(artifactsDir, filename) {
			continue
		}
		path := filepath.Join(artifactsDir, small.SmallDir, filename)
		if err := os.WriteFile(path, []byte(template), 0644); err != nil {
			return fmt.Errorf("failed to write %s: %w", path, err)
		}
	}

	workspacePath := filepath.Join(artifactsDir, small.SmallDir, "workspace.small.yml")
	if _, err := os.Stat(workspacePath); os.IsNotExist(err) {
		if err := workspace.Save(artifactsDir, workspace.KindRepoRoot); err != nil {
			return err
		}
	}
	return nil
}

func existingLinks(existing *existingHandoff) []linkOut {
	if existing == nil {
		return nil
	}
	return existing.Links
}

func validateIntentAndPlan(artifactsDir string) bool {
	intent, err := small.LoadArtifact(artifactsDir, "intent.small.yml")
	if err != nil {
		return false
	}
	plan, err := small.LoadArtifact(artifactsDir, "plan.small.yml")
	if err != nil {
		return false
	}

	config := small.SchemaConfig{BaseDir: artifactsDir}
	if err := small.ValidateArtifactWithConfig(intent, config); err != nil {
		return false
	}
	if err := small.ValidateArtifactWithConfig(plan, config); err != nil {
		return false
	}
	return true
}
