package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/justyn-clark/small-protocol/internal/small"
	"github.com/justyn-clark/small-protocol/internal/workspace"
	"github.com/spf13/cobra"
)

func fixCmd() *cobra.Command {
	var (
		fixVersions   bool
		dir           string
		workspaceFlag string
	)

	cmd := &cobra.Command{
		Use:   "fix",
		Short: "Normalize SMALL artifacts in-place",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !fixVersions {
				return fmt.Errorf("no fix selected (use --versions)")
			}
			if dir == "" {
				dir = baseDir
			}
			artifactsDir := resolveArtifactsDir(dir)

			scope, err := workspace.ParseScope(workspaceFlag)
			if err != nil {
				return err
			}
			if scope != workspace.ScopeAny {
				if err := enforceWorkspaceScope(artifactsDir, scope); err != nil {
					return err
				}
			}

			changed, canonical, err := fixVersionFormatting(artifactsDir)
			if err != nil {
				return err
			}

			fmt.Println("small_version normalization complete.")
			if len(changed) > 0 {
				fmt.Printf("Changed (%d):\n", len(changed))
				for _, file := range changed {
					fmt.Printf("  - %s\n", file)
				}
			}
			if len(canonical) > 0 {
				fmt.Printf("Already canonical (%d):\n", len(canonical))
				for _, file := range canonical {
					fmt.Printf("  - %s\n", file)
				}
			}
			fmt.Println("Next: small check --strict")
			return nil
		},
	}

	cmd.Flags().BoolVar(&fixVersions, "versions", false, "Normalize small_version formatting (quoted string)")
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
