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

func acceptCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "accept",
		Short: "Accept draft artifacts into canonical files",
	}

	cmd.AddCommand(acceptArtifactCmd("intent"))
	cmd.AddCommand(acceptArtifactCmd("constraints"))

	return cmd
}

func acceptArtifactCmd(kind string) *cobra.Command {
	var dir string
	var workspaceFlag string

	cmd := &cobra.Command{
		Use:   kind,
		Short: fmt.Sprintf("Accept draft %s.small.yml", kind),
		RunE: func(cmd *cobra.Command, args []string) error {
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

			smallDir := filepath.Join(artifactsDir, small.SmallDir)
			draftPath := filepath.Join(smallDir, "drafts", fmt.Sprintf("%s.small.yml", kind))
			data, err := os.ReadFile(draftPath)
			if err != nil {
				if os.IsNotExist(err) {
					return fmt.Errorf("draft not found: %s (run: small draft %s --from <path|stdin>)", draftPath, kind)
				}
				return fmt.Errorf("failed to read draft %s: %w", draftPath, err)
			}

			outPath := filepath.Join(smallDir, fmt.Sprintf("%s.small.yml", kind))
			if err := os.WriteFile(outPath, data, 0644); err != nil {
				return fmt.Errorf("failed to write %s: %w", outPath, err)
			}

			evidence := fmt.Sprintf("Accepted draft %s from %s", kind, filepath.Join(small.SmallDir, "drafts", fmt.Sprintf("%s.small.yml", kind)))
			entry := map[string]interface{}{
				"task_id":  fmt.Sprintf("meta/accept-%s", kind),
				"status":   "completed",
				"evidence": evidence,
				"notes":    strings.TrimSpace(fmt.Sprintf("small accept %s", kind)),
			}
			if err := appendProgressEntry(artifactsDir, entry); err != nil {
				return fmt.Errorf("failed to record accept progress: %w", err)
			}

			fmt.Printf("Accepted draft: %s -> %s\n", filepath.Join(small.SmallDir, "drafts", fmt.Sprintf("%s.small.yml", kind)), filepath.Join(small.SmallDir, fmt.Sprintf("%s.small.yml", kind)))
			fmt.Printf("Recorded progress entry: meta/accept-%s\n", kind)
			return nil
		},
	}

	cmd.Flags().StringVar(&dir, "dir", ".", "Directory containing .small/ artifacts")
	cmd.Flags().StringVar(&workspaceFlag, "workspace", string(workspace.ScopeRoot), "Workspace scope (root or any)")

	return cmd
}
