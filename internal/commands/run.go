package commands

import (
	"bytes"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"github.com/justyn-clark/small-protocol/internal/runstore"
	"github.com/justyn-clark/small-protocol/internal/workspace"
	"github.com/spf13/cobra"
)

type runListItem struct {
	CreatedAt string `json:"created_at"`
	ReplayID  string `json:"replayId"`
	Summary   string `json:"summary,omitempty"`
	GitSHA    string `json:"git_sha,omitempty"`
	GitDirty  bool   `json:"git_dirty"`
	Branch    string `json:"branch,omitempty"`
}

type runListOutput struct {
	Runs []runListItem `json:"runs"`
}

type runShowOutput struct {
	ReplayID  string        `json:"replayId"`
	Meta      runstore.Meta `json:"meta"`
	Artifacts []string      `json:"artifacts"`
	Summary   string        `json:"summary,omitempty"`
	NextSteps []string      `json:"next_steps,omitempty"`
	Dir       string        `json:"dir"`
}

func runCmd() *cobra.Command {
	var (
		dir           string
		storeFlag     string
		workspaceFlag string
	)

	cmd := &cobra.Command{
		Use:   "run",
		Short: "Manage run history snapshots",
	}

	cmd.PersistentFlags().StringVar(&dir, "dir", ".", "Directory containing .small/ artifacts")
	cmd.PersistentFlags().StringVar(&storeFlag, "store", "", "Run store directory (default: <workspace>/.small-runs)")
	cmd.PersistentFlags().StringVar(&workspaceFlag, "workspace", string(workspace.ScopeRoot), "Workspace scope (root, examples, or any)")

	cmd.AddCommand(runSnapshotCmd(&dir, &storeFlag, &workspaceFlag))
	cmd.AddCommand(runListCmd(&dir, &storeFlag, &workspaceFlag))
	cmd.AddCommand(runShowCmd(&dir, &storeFlag, &workspaceFlag))
	cmd.AddCommand(runDiffCmd(&dir, &storeFlag, &workspaceFlag))
	cmd.AddCommand(runCheckoutCmd(&dir, &storeFlag, &workspaceFlag))

	return cmd
}

func runSnapshotCmd(dir, storeFlag, workspaceFlag *string) *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "snapshot",
		Short: "Snapshot current workspace into the run store",
		RunE: func(cmd *cobra.Command, args []string) error {
			artifactsDir, storeDir, err := resolveRunContext(*dir, *storeFlag, *workspaceFlag)
			if err != nil {
				return err
			}

			snapshot, err := runstore.WriteSnapshot(artifactsDir, storeDir, force)
			if err != nil {
				return err
			}

			fmt.Printf("Snapshot saved: %s\n", snapshot.Dir)
			fmt.Printf("replayId: %s\n", snapshot.ReplayID)
			return nil
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Overwrite existing snapshot directory")
	return cmd
}

func runListCmd(dir, storeFlag, workspaceFlag *string) *cobra.Command {
	var (
		limit      int
		jsonOutput bool
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List run snapshots",
		RunE: func(cmd *cobra.Command, args []string) error {
			artifactsDir, storeDir, err := resolveRunContext(*dir, *storeFlag, *workspaceFlag)
			if err != nil {
				return err
			}
			_ = artifactsDir

			snapshots, err := runstore.ListSnapshots(storeDir)
			if err != nil {
				return err
			}
			if limit > 0 && len(snapshots) > limit {
				snapshots = snapshots[:limit]
			}

			output, err := formatRunListOutput(snapshots, jsonOutput)
			if err != nil {
				return err
			}
			fmt.Print(output)
			return nil
		},
	}

	cmd.Flags().IntVar(&limit, "limit", 20, "Maximum number of snapshots to show")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output in JSON format")
	return cmd
}

func runShowCmd(dir, storeFlag, workspaceFlag *string) *cobra.Command {
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "show <replayId>",
		Short: "Show snapshot details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			artifactsDir, storeDir, err := resolveRunContext(*dir, *storeFlag, *workspaceFlag)
			if err != nil {
				return err
			}
			_ = artifactsDir

			snapshot, err := runstore.LoadSnapshot(storeDir, args[0])
			if err != nil {
				return err
			}

			output, err := formatRunShowOutput(snapshot, jsonOutput)
			if err != nil {
				return err
			}
			fmt.Print(output)
			return nil
		},
	}

	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output in JSON format")
	return cmd
}

func runCheckoutCmd(dir, storeFlag, workspaceFlag *string) *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "checkout <replayId>",
		Short: "Restore a snapshot into the live workspace",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			artifactsDir, storeDir, err := resolveRunContext(*dir, *storeFlag, *workspaceFlag)
			if err != nil {
				return err
			}

			if err := runstore.CheckoutSnapshot(artifactsDir, storeDir, args[0], force); err != nil {
				return err
			}

			fmt.Printf("Restored snapshot %s into %s\n", shortID(args[0], 16), filepath.Join(artifactsDir, ".small"))
			return nil
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Overwrite local .small changes")
	return cmd
}

func resolveRunContext(dirFlag, storeFlag, workspaceFlag string) (string, string, error) {
	if dirFlag == "" {
		dirFlag = baseDir
	}
	artifactsDir := resolveArtifactsDir(dirFlag)

	scope, err := workspace.ParseScope(workspaceFlag)
	if err != nil {
		return "", "", err
	}
	if scope != workspace.ScopeAny {
		if err := enforceWorkspaceScope(artifactsDir, scope); err != nil {
			return "", "", err
		}
	}

	storeDir := runstore.ResolveStoreDir(artifactsDir, storeFlag)
	return artifactsDir, storeDir, nil
}

func formatRunListOutput(snapshots []runstore.Snapshot, jsonOutput bool) (string, error) {
	if jsonOutput {
		items := make([]runListItem, 0, len(snapshots))
		for _, snapshot := range snapshots {
			items = append(items, runListItem{
				CreatedAt: snapshot.Meta.CreatedAt,
				ReplayID:  snapshot.ReplayID,
				Summary:   snapshot.HandoffSummary,
				GitSHA:    snapshot.Meta.GitSHA,
				GitDirty:  snapshot.Meta.GitDirty,
				Branch:    snapshot.Meta.Branch,
			})
		}
		payload := runListOutput{Runs: items}
		data, err := json.MarshalIndent(payload, "", "  ")
		if err != nil {
			return "", err
		}
		return string(data) + "\n", nil
	}

	if len(snapshots) == 0 {
		return "no run snapshots found\n", nil
	}

	var buffer bytes.Buffer
	writer := tabwriter.NewWriter(&buffer, 0, 4, 2, ' ', 0)
	_, _ = fmt.Fprintln(writer, "created_at\treplayId\tsummary\tgit_sha\tgit_dirty")
	for _, snapshot := range snapshots {
		summary := snapshot.HandoffSummary
		if summary == "" {
			summary = "-"
		}
		gitSHA := shortID(snapshot.Meta.GitSHA, 8)
		if gitSHA == "" {
			gitSHA = "-"
		}
		_, _ = fmt.Fprintf(writer, "%s\t%s\t%s\t%s\t%t\n",
			snapshot.Meta.CreatedAt,
			shortID(snapshot.ReplayID, 8),
			summary,
			gitSHA,
			snapshot.Meta.GitDirty,
		)
	}
	_ = writer.Flush()
	return buffer.String(), nil
}

func formatRunShowOutput(snapshot *runstore.Snapshot, jsonOutput bool) (string, error) {
	if jsonOutput {
		payload := runShowOutput{
			ReplayID:  snapshot.ReplayID,
			Meta:      snapshot.Meta,
			Artifacts: snapshot.Artifacts,
			Summary:   snapshot.HandoffSummary,
			NextSteps: snapshot.HandoffNextSteps,
			Dir:       snapshot.Dir,
		}
		data, err := json.MarshalIndent(payload, "", "  ")
		if err != nil {
			return "", err
		}
		return string(data) + "\n", nil
	}

	var buffer bytes.Buffer
	writeLine := func(format string, args ...interface{}) {
		_, _ = fmt.Fprintf(&buffer, format, args...)
	}

	writeLine("ReplayId: %s\n", snapshot.ReplayID)
	writeLine("Created at: %s\n", snapshot.Meta.CreatedAt)
	writeLine("Store: %s\n", snapshot.Dir)
	writeLine("CLI version: %s\n", snapshot.Meta.CLIVersion)
	writeLine("Workspace kind: %s\n", snapshot.Meta.WorkspaceKind)
	if snapshot.Meta.SourceDir != "" {
		writeLine("Source dir: %s\n", snapshot.Meta.SourceDir)
	}
	if snapshot.Meta.GitSHA != "" || snapshot.Meta.Branch != "" {
		writeLine("Git: %s (dirty: %t)\n", shortID(snapshot.Meta.GitSHA, 8), snapshot.Meta.GitDirty)
		if snapshot.Meta.Branch != "" {
			writeLine("Branch: %s\n", snapshot.Meta.Branch)
		}
	}

	writeLine("Artifacts:\n")
	for _, artifact := range snapshot.Artifacts {
		writeLine("  - %s\n", artifact)
	}

	if snapshot.HandoffSummary != "" {
		writeLine("Summary: %s\n", snapshot.HandoffSummary)
	}
	if len(snapshot.HandoffNextSteps) > 0 {
		writeLine("Next steps:\n")
		for _, step := range snapshot.HandoffNextSteps {
			writeLine("  - %s\n", step)
		}
	} else {
		writeLine("Next steps: none\n")
	}

	return buffer.String(), nil
}

func shortID(value string, length int) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	if len(trimmed) <= length {
		return trimmed
	}
	return trimmed[:length] + "..."
}
