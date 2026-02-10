package commands

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/justyn-clark/small-protocol/internal/small"
	"github.com/justyn-clark/small-protocol/internal/workspace"
	"github.com/spf13/cobra"
)

func draftCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "draft",
		Short: "Create draft artifacts for human-owned files",
	}

	cmd.AddCommand(draftArtifactCmd("intent"))
	cmd.AddCommand(draftArtifactCmd("constraints"))

	return cmd
}

func draftArtifactCmd(kind string) *cobra.Command {
	var from string
	var dir string
	var workspaceFlag string

	cmd := &cobra.Command{
		Use:   fmt.Sprintf("%s --from <path|stdin>", kind),
		Short: fmt.Sprintf("Write a draft %s.small.yml", kind),
		RunE: func(cmd *cobra.Command, args []string) error {
			if strings.TrimSpace(from) == "" {
				return fmt.Errorf("--from is required")
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

			data, err := readDraftSource(from)
			if err != nil {
				return err
			}

			draftsDir := small.CacheDraftsDir(artifactsDir)
			if err := os.MkdirAll(draftsDir, 0755); err != nil {
				return fmt.Errorf("failed to create drafts directory: %w", err)
			}

			filename := fmt.Sprintf("%s.small.yml", kind)
			outPath := filepath.Join(draftsDir, filename)
			if err := os.WriteFile(outPath, data, 0644); err != nil {
				return fmt.Errorf("failed to write draft %s: %w", outPath, err)
			}

			relPath := filepath.Join(small.CacheDirName, "drafts", filename)
			fmt.Printf("Draft saved: %s\n", relPath)
			fmt.Printf("Next: small accept %s\n", kind)
			return nil
		},
	}

	cmd.Flags().StringVar(&from, "from", "", "Path to draft source file or stdin")
	cmd.Flags().StringVar(&dir, "dir", ".", "Directory containing .small/ artifacts")
	cmd.Flags().StringVar(&workspaceFlag, "workspace", string(workspace.ScopeRoot), "Workspace scope (root or any)")

	_ = cmd.MarkFlagRequired("from")

	return cmd
}

func readDraftSource(source string) ([]byte, error) {
	value := strings.TrimSpace(source)
	if strings.EqualFold(value, "stdin") || value == "-" {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return nil, fmt.Errorf("failed to read stdin: %w", err)
		}
		if len(bytes.TrimSpace(data)) == 0 {
			return nil, fmt.Errorf("stdin is empty")
		}
		return data, nil
	}
	data, err := os.ReadFile(value)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", value, err)
	}
	return data, nil
}
