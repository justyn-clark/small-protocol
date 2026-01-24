package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/justyn-clark/small-protocol/internal/small"
	"github.com/justyn-clark/small-protocol/internal/small/fixers"
	"github.com/justyn-clark/small-protocol/internal/workspace"
	"github.com/spf13/cobra"
)

func fixCmd() *cobra.Command {
	var (
		fixVersions       bool
		fixOrphanProgress bool
		dir               string
		workspaceFlag     string
	)

	cmd := &cobra.Command{
		Use:   "fix",
		Short: "Normalize SMALL artifacts in-place",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !fixVersions && !fixOrphanProgress {
				return fmt.Errorf("no fix selected (use --versions or --orphan-progress)")
			}
			if dir == "" {
				dir = baseDir
			}
			artifactsDir := resolveArtifactsDir(dir)
			p := currentPrinter()

			scope, err := workspace.ParseScope(workspaceFlag)
			if err != nil {
				return err
			}
			if scope != workspace.ScopeAny {
				if err := enforceWorkspaceScope(artifactsDir, scope); err != nil {
					return err
				}
			}

			if fixVersions {
				changed, canonical, err := fixVersionFormatting(artifactsDir)
				if err != nil {
					return err
				}

				p.PrintInfo("small_version normalization complete.")
				if len(changed) > 0 {
					p.PrintInfo(fmt.Sprintf("Changed (%d):", len(changed)))
					for _, file := range changed {
						p.PrintInfo(fmt.Sprintf("- %s", file))
					}
				}
				if len(canonical) > 0 {
					p.PrintInfo(fmt.Sprintf("Already canonical (%d):", len(canonical)))
					for _, file := range canonical {
						p.PrintInfo(fmt.Sprintf("- %s", file))
					}
				}
			}

			if fixOrphanProgress {
				result, err := fixers.FixOrphanProgress(artifactsDir)
				if err != nil {
					return err
				}

				scopeLabel := strings.TrimSpace(result.ReplayID)
				if scopeLabel == "" {
					scopeLabel = "unknown"
				}

				if len(result.Rewrites) == 0 {
					lines := []string{
						fmt.Sprintf("ReplayId scope: %s", scopeLabel),
						"No orphan progress entries found.",
					}
					p.PrintInfo(p.FormatBlock("Orphan progress", lines))
				} else {
					byCategory := map[string][]string{
						"operational": {},
						"historical":  {},
						"unknown":     {},
					}
					for _, rewrite := range result.Rewrites {
						entry := fmt.Sprintf("%s -> %s", rewrite.OriginalTaskID, rewrite.NewTaskID)
						byCategory[rewrite.Category] = append(byCategory[rewrite.Category], entry)
					}

					lines := []string{
						fmt.Sprintf("ReplayId scope: %s", scopeLabel),
						fmt.Sprintf("Rewrote %d orphan progress entr(ies).", len(result.Rewrites)),
						fmt.Sprintf("Counts: operational=%d historical=%d unknown=%d", result.Counts.Operational, result.Counts.Historical, result.Counts.Unknown),
						"Rewrites by category:",
					}
					lines = append(lines, fmt.Sprintf("  %s", formatCountedList("operational", byCategory["operational"], defaultListCap)))
					lines = append(lines, fmt.Sprintf("  %s", formatCountedList("historical", byCategory["historical"], defaultListCap)))
					lines = append(lines, fmt.Sprintf("  %s", formatCountedList("unknown", byCategory["unknown"], defaultListCap)))
					if err := recordOrphanProgressReconcileEntry(artifactsDir, result); err != nil {
						return err
					}
					lines = append(lines, "", "Recorded progress entry: meta/reconcile-plan")
					lines = append(lines, "Fix:", "small check --strict")
					p.PrintInfo(p.FormatBlock("Orphan progress rewritten", lines))
				}
			}

			p.PrintInfo("Next: small check --strict")
			return nil
		},
	}

	cmd.Flags().BoolVar(&fixVersions, "versions", false, "Normalize small_version formatting (quoted string)")
	cmd.Flags().BoolVar(&fixOrphanProgress, "orphan-progress", false, "Rewrite orphan progress task_ids to meta names")
	cmd.Flags().StringVar(&dir, "dir", ".", "Directory containing .small/ artifacts")
	cmd.Flags().StringVar(&workspaceFlag, "workspace", string(workspace.ScopeRoot), "Workspace scope (root, examples, or any)")

	return cmd
}

func fixVersionFormatting(artifactsDir string) ([]string, []string, error) {
	var changed []string
	var canonical []string
	files := versionFormatFiles()

	for _, filename := range files {
		path := filepath.Join(artifactsDir, small.SmallDir, filename)
		data, err := os.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, nil, fmt.Errorf("failed to read %s: %w", path, err)
		}

		updated, wasCanonical, err := normalizeSmallVersionInFile(string(data))
		if err != nil {
			return nil, nil, fmt.Errorf("failed to normalize %s: %w", path, err)
		}
		if updated == "" {
			continue
		}
		if wasCanonical {
			canonical = append(canonical, filepath.Join(small.SmallDir, filename))
			continue
		}
		if updated != string(data) {
			if err := os.WriteFile(path, []byte(updated), 0644); err != nil {
				return nil, nil, fmt.Errorf("failed to write %s: %w", path, err)
			}
			changed = append(changed, filepath.Join(small.SmallDir, filename))
		} else {
			canonical = append(canonical, filepath.Join(small.SmallDir, filename))
		}
	}

	return changed, canonical, nil
}

func normalizeSmallVersionInFile(content string) (string, bool, error) {
	lines := strings.Split(content, "\n")
	changed := false
	found := false
	canonical := false

	for i, line := range lines {
		normalized, didChange := normalizeSmallVersionLine(line)
		if normalized == line && !didChange {
			if smallVersionLinePattern.MatchString(line) {
				found = true
				canonical = true
			}
			continue
		}
		if smallVersionLinePattern.MatchString(line) {
			found = true
		}
		lines[i] = normalized
		changed = changed || didChange
		canonical = false
		break
	}

	if !found {
		return "", false, nil
	}

	if !changed && canonical {
		return content, true, nil
	}

	result := strings.Join(lines, "\n")
	if strings.HasSuffix(content, "\n") && !strings.HasSuffix(result, "\n") {
		result += "\n"
	}
	return result, false, nil
}
