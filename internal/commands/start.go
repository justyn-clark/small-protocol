package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/justyn-clark/small-protocol/internal/small"
	"github.com/justyn-clark/small-protocol/internal/small/fixers"
	"github.com/justyn-clark/small-protocol/internal/workspace"
	"github.com/spf13/cobra"
)

func startCmd() *cobra.Command {
	var (
		fixOrphanProgress bool
		summary           string
		dir               string
		workspaceFlag     string
	)

	cmd := &cobra.Command{
		Use:   "start",
		Short: "Initialize or repair run handoff state",
		RunE: func(cmd *cobra.Command, args []string) error {
			if dir == "" {
				dir = baseDir
			}
			artifactsDir := resolveArtifactsDir(dir)
			p := currentPrinter()

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
			if err := setWorkspaceRunReplayIDIfPresent(artifactsDir, handoff.ReplayId.Value); err != nil {
				return err
			}

			if err := writeHandoff(artifactsDir, handoff); err != nil {
				return err
			}

			if selfHeal {
				entry := map[string]any{
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

			strictCheck := true
			p.PrintInfo("Running strict check (validate, lint, verify)...")
			code, output, err := runCheck(artifactsDir, strictCheck, false, false, scope, false)
			if err != nil {
				return err
			}
			if code != ExitValid {
				if isOrphanProgressOnly(output) {
					if fixOrphanProgress {
						result, err := fixers.FixOrphanProgress(artifactsDir)
						if err != nil {
							return err
						}
						if err := recordOrphanProgressReconcileEntry(artifactsDir, result); err != nil {
							return err
						}
						p.PrintInfo("Applied orphan progress fix. Re-running strict check...")
						code, output, err = runCheck(artifactsDir, strictCheck, false, false, scope, false)
						if err != nil {
							return err
						}
						if code != ExitValid {
							return fmt.Errorf("check failed (validate=%s lint=%s verify=%s)", output.Validate.Status, output.Lint.Status, output.Verify.Status)
						}
						p.PrintInfo("Strict check passed. Start complete. Handoff ready.")
						return nil
					}
					lines := []string{
						"Why: orphan progress entries exist in the current replayId scope.",
						"Fix:",
						"small fix --orphan-progress",
						"small start --fix",
					}
					p.PrintError(p.FormatBlock("Strict check failed (orphan progress)", lines))
					return fmt.Errorf("check failed (orphan progress)")
				}
				return fmt.Errorf("check failed (validate=%s lint=%s verify=%s)", output.Validate.Status, output.Lint.Status, output.Verify.Status)
			}

			p.PrintInfo("Strict check passed. Start complete. Handoff ready.")
			return nil
		},
	}

	cmd.Flags().StringVar(&summary, "summary", "", "Summary description for the handoff")
	cmd.Flags().BoolVar(&fixOrphanProgress, "fix", false, "Auto-fix orphan progress if strict check fails")
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

func isOrphanProgressOnly(output checkOutput) bool {
	if output.Validate.Status != "ok" {
		return false
	}
	if output.Verify.Status != "ok" {
		return false
	}
	if output.Lint.Status != "failed" {
		return false
	}
	if len(output.Lint.Errors) == 0 {
		return false
	}
	for _, message := range output.Lint.Errors {
		if !strings.Contains(message, "strict invariant S2 failed") {
			return false
		}
	}
	return true
}
