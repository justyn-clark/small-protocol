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
		fixAll            bool
	)

	cmd := &cobra.Command{
		Use:   "fix",
		Short: "Normalize SMALL artifacts in-place",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !fixVersions && !fixOrphanProgress && !fixAll {
				return fmt.Errorf("no fix selected (use --versions, --orphan-progress, or --all)")
			}

			// --all includes workspace fix
			if fixAll {
				fixVersions = true
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

			// --all includes workspace fix
			if fixAll {
				wsResult, err := workspace.Fix(artifactsDir, workspace.KindRepoRoot, false)
				if err != nil {
					return err
				}

				var wsLines []string
				if wsResult.Created {
					wsLines = append(wsLines, "Created workspace.small.yml")
				} else {
					wsLines = append(wsLines, "Checked workspace.small.yml")
				}

				var fixes []string
				if wsResult.AddedOwner {
					fixes = append(fixes, "owner")
				}
				if wsResult.AddedCreatedAt {
					fixes = append(fixes, "created_at")
				}
				if wsResult.AddedUpdatedAt {
					fixes = append(fixes, "updated_at")
				}
				if wsResult.NormalizedFormat {
					fixes = append(fixes, "format")
				}

				if len(fixes) > 0 {
					wsLines = append(wsLines, fmt.Sprintf("Fixed: %s", strings.Join(fixes, ", ")))
				} else if !wsResult.Created {
					wsLines = append(wsLines, "No changes needed")
				}

				p.PrintInfo(p.FormatBlock("Workspace fix", wsLines))
			}

			p.PrintInfo("Next: small check --strict")
			return nil
		},
	}

	cmd.Flags().BoolVar(&fixVersions, "versions", false, "Normalize small_version formatting (quoted string)")
	cmd.Flags().BoolVar(&fixOrphanProgress, "orphan-progress", false, "Rewrite orphan progress task_ids to meta names")
	cmd.Flags().BoolVar(&fixAll, "all", false, "Run all fixers (includes workspace)")
	cmd.Flags().StringVar(&dir, "dir", ".", "Directory containing .small/ artifacts")
	cmd.Flags().StringVar(&workspaceFlag, "workspace", string(workspace.ScopeRoot), "Workspace scope (root, examples, or any)")

	cmd.AddCommand(fixWorkspaceCmd())

	return cmd
}

func fixWorkspaceCmd() *cobra.Command {
	var (
		dir   string
		kind  string
		force bool
	)

	cmd := &cobra.Command{
		Use:   "workspace",
		Short: "Create or repair workspace.small.yml",
		Long: `Creates or repairs .small/workspace.small.yml.

If the file is missing, it will be created with default values.
If the file exists but has missing or invalid timestamps, they will be repaired.

Fields:
  small_version: "1.0.0"
  owner: "agent"
  kind: "repo-root" (or "examples")
  created_at: UTC RFC3339 timestamp (set only if missing)
  updated_at: UTC RFC3339 timestamp (set to now)

Use --force to overwrite all fields including kind.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if dir == "" {
				dir = baseDir
			}
			artifactsDir := resolveArtifactsDir(dir)
			p := currentPrinter()

			wsKind := workspace.KindRepoRoot
			if kind == "examples" {
				wsKind = workspace.KindExamples
			}

			result, err := workspace.Fix(artifactsDir, wsKind, force)
			if err != nil {
				return err
			}

			var lines []string
			if result.Created {
				lines = append(lines, "Created workspace.small.yml")
			} else {
				lines = append(lines, "Repaired workspace.small.yml")
			}

			var fixes []string
			if result.AddedOwner {
				fixes = append(fixes, "owner")
			}
			if result.AddedCreatedAt {
				fixes = append(fixes, "created_at")
			}
			if result.AddedUpdatedAt {
				fixes = append(fixes, "updated_at")
			}
			if result.NormalizedFormat {
				fixes = append(fixes, "format")
			}

			if len(fixes) > 0 {
				lines = append(lines, fmt.Sprintf("Fixed: %s", strings.Join(fixes, ", ")))
			} else if !result.Created {
				lines = append(lines, "No changes needed")
			}

			p.PrintSuccess(p.FormatBlock("Workspace fix", lines))
			p.PrintInfo("Next: small check --strict")
			return nil
		},
	}

	cmd.Flags().StringVar(&dir, "dir", ".", "Directory containing .small/ artifacts")
	cmd.Flags().StringVar(&kind, "kind", "repo-root", "Workspace kind (repo-root or examples)")
	cmd.Flags().BoolVar(&force, "force", false, "Overwrite all fields including kind")

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
