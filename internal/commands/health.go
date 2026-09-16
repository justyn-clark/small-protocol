package commands

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"github.com/justyn-clark/small-protocol/internal/runstore"
	"github.com/justyn-clark/small-protocol/internal/sessionv2"
	"github.com/justyn-clark/small-protocol/internal/small"
	"github.com/justyn-clark/small-protocol/internal/workspace"
	"github.com/spf13/cobra"
)

type healthOutput struct {
	Workspaces []healthWorkspace `json:"workspaces"`
}

type healthWorkspace struct {
	Path                         string      `json:"path"`
	SmallDirExists               bool        `json:"small_dir_exists"`
	StrictStatus                 string      `json:"strict_status"`
	StrictErrors                 []string    `json:"strict_errors,omitempty"`
	ReplayID                     string      `json:"replay_id,omitempty"`
	ArtifactDigest               string      `json:"artifact_digest,omitempty"`
	SnapshotCount                int         `json:"snapshot_count"`
	LatestSnapshotReplayID       string      `json:"latest_snapshot_replay_id,omitempty"`
	LatestSnapshotArtifactDigest string      `json:"latest_snapshot_artifact_digest,omitempty"`
	LatestSnapshotStale          bool        `json:"latest_snapshot_stale"`
	Plan                         *PlanStatus `json:"plan,omitempty"`
}

func healthCmd() *cobra.Command {
	var (
		jsonOutput    bool
		storeFlag     string
		workspaceFlag string
	)

	cmd := &cobra.Command{
		Use:   "health [workspace ...]",
		Short: "Report SMALL health across one or more workspaces",
		Long: `Reports strict status, replay lineage, current artifact digest, and run snapshot freshness.

With no workspace arguments, health inspects the current directory. Additional
arguments may point at repository roots or .small directories. This command is
read-only and does not modify .small/ or run stores.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				args = []string{baseDir}
			}

			scope, err := workspace.ParseScope(workspaceFlag)
			if err != nil {
				return err
			}

			result := healthOutput{Workspaces: make([]healthWorkspace, 0, len(args))}
			for _, arg := range args {
				workspaceHealth := inspectWorkspaceHealth(arg, storeFlag, scope)
				result.Workspaces = append(result.Workspaces, workspaceHealth)
			}

			if jsonOutput {
				data, err := json.MarshalIndent(result, "", "  ")
				if err != nil {
					return err
				}
				fmt.Println(string(data))
				return nil
			}

			fmt.Print(formatHealthText(result))
			return nil
		},
	}

	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output in JSON format")
	cmd.Flags().StringVar(&storeFlag, "store", "", "Run store directory (default: <workspace>/.small-runs)")
	cmd.Flags().StringVar(&workspaceFlag, "workspace", string(workspace.ScopeAny), "Workspace scope (root or any)")
	return cmd
}

func inspectWorkspaceHealth(dir, storeFlag string, scope workspace.Scope) healthWorkspace {
	artifactsDir := resolveArtifactsDir(dir)
	result := healthWorkspace{Path: artifactsDir, StrictStatus: "unknown"}

	if _, err := os.Stat(filepath.Join(artifactsDir, small.SmallDir)); err != nil {
		result.SmallDirExists = false
		if os.IsNotExist(err) {
			result.StrictStatus = "missing"
			result.StrictErrors = []string{".small/ directory does not exist"}
			return result
		}
		result.StrictStatus = "error"
		result.StrictErrors = []string{err.Error()}
		return result
	}
	result.SmallDirExists = true

	if scope != workspace.ScopeAny {
		if err := enforceWorkspaceScope(artifactsDir, scope); err != nil {
			result.StrictStatus = "error"
			result.StrictErrors = []string{err.Error()}
			return result
		}
	}

	code, checkResult, err := runCheck(artifactsDir, true, true, false, workspace.ScopeAny, false)
	if err != nil {
		result.StrictStatus = "error"
		result.StrictErrors = []string{err.Error()}
	} else if code == ExitValid {
		result.StrictStatus = "ok"
	} else {
		result.StrictStatus = "failed"
		result.StrictErrors = flattenCheckErrors(checkResult)
	}

	if replayID := currentHealthReplayID(artifactsDir); replayID != "" {
		result.ReplayID = replayID
	}

	if digest, _, err := runstore.ComputeArtifactDigests(artifactsDir); err == nil {
		result.ArtifactDigest = digest
	} else {
		result.StrictErrors = append(result.StrictErrors, err.Error())
		if result.StrictStatus == "ok" {
			result.StrictStatus = "error"
		}
	}

	if small.ArtifactExists(artifactsDir, "plan.small.yml") {
		if planStatus, err := analyzePlan(artifactsDir, 5); err == nil {
			result.Plan = planStatus
		}
	}

	storeDir := runstore.ResolveStoreDir(artifactsDir, storeFlag)
	if _, statErr := os.Stat(storeDir); statErr != nil {
		// A missing run store is the normal "no snapshots yet" case, not an
		// error. Only surface real read failures (permissions, corruption).
		if !os.IsNotExist(statErr) {
			result.StrictErrors = append(result.StrictErrors, fmt.Sprintf("run store unreadable: %v", statErr))
			if result.StrictStatus == "ok" {
				result.StrictStatus = "error"
			}
		}
		return result
	}

	snapshots, err := runstore.ListSnapshots(storeDir)
	if err != nil {
		result.StrictErrors = append(result.StrictErrors, fmt.Sprintf("failed to read run store: %v", err))
		if result.StrictStatus == "ok" {
			result.StrictStatus = "error"
		}
		return result
	}

	result.SnapshotCount = len(snapshots)
	if len(snapshots) > 0 {
		latest := snapshots[0]
		result.LatestSnapshotReplayID = latest.ReplayID
		result.LatestSnapshotArtifactDigest = latest.Meta.ArtifactDigest
		result.LatestSnapshotStale = latest.Meta.ArtifactDigest != "" && result.ArtifactDigest != "" && latest.Meta.ArtifactDigest != result.ArtifactDigest
	}

	return result
}

func flattenCheckErrors(output checkOutput) []string {
	errors := []string{}
	errors = append(errors, output.Validate.Errors...)
	errors = append(errors, output.Lint.Errors...)
	errors = append(errors, output.Verify.Errors...)
	if len(errors) == 0 && output.Verify.Status == "failed" {
		errors = append(errors, "verify failed")
	}
	return errors
}

func currentHealthReplayID(artifactsDir string) string {
	if sessionv2.IsWorkspace(artifactsDir) {
		if store, err := sessionv2.Load(artifactsDir); err == nil {
			return sessionv2.Frontier(store.Events)
		}
	}
	if replayID, err := workspace.RunReplayID(artifactsDir); err == nil && strings.TrimSpace(replayID) != "" {
		return strings.TrimSpace(replayID)
	}

	artifact, err := small.LoadArtifact(artifactsDir, "handoff.small.yml")
	if err != nil || artifact == nil || artifact.Data == nil {
		return ""
	}
	replay, _ := artifact.Data["replayId"].(map[string]any)
	if replay == nil {
		return ""
	}
	return strings.TrimSpace(stringVal(replay["value"]))
}

func formatHealthText(output healthOutput) string {
	if len(output.Workspaces) == 0 {
		return "no workspaces inspected\n"
	}

	var buffer bytes.Buffer
	writer := tabwriter.NewWriter(&buffer, 0, 4, 2, 32, 0)
	_, _ = fmt.Fprintln(writer, "strict\tsnapshots\tstale\treplayId\tartifact_digest\tnext_task\tpath")
	for _, workspace := range output.Workspaces {
		nextTask := "-"
		if workspace.Plan != nil {
			nextTask = resolveNextTask(workspace.Plan, "")
			if nextTask == "" {
				nextTask = "-"
			}
		}
		_, _ = fmt.Fprintf(writer, "%s\t%d\t%t\t%s\t%s\t%s\t%s\n",
			workspace.StrictStatus,
			workspace.SnapshotCount,
			workspace.LatestSnapshotStale,
			shortID(workspace.ReplayID, 8),
			shortID(workspace.ArtifactDigest, 8),
			nextTask,
			workspace.Path,
		)
	}
	_ = writer.Flush()

	for _, workspace := range output.Workspaces {
		if len(workspace.StrictErrors) == 0 {
			continue
		}
		_, _ = fmt.Fprintf(&buffer, "\n%s errors:\n", workspace.Path)
		for _, msg := range workspace.StrictErrors {
			_, _ = fmt.Fprintf(&buffer, "  - %s\n", msg)
		}
	}

	return buffer.String()
}
