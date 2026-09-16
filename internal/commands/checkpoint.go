package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/justyn-clark/small-protocol/internal/sessionv2"
	"github.com/justyn-clark/small-protocol/internal/small"
	"github.com/justyn-clark/small-protocol/internal/workspace"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

type checkpointOutput struct {
	TaskID       string         `json:"task_id"`
	Status       string         `json:"status"`
	Progress     map[string]any `json:"progress_entry"`
	Files        []string       `json:"files"`
	Validated    bool           `json:"validated"`
	Workspace    string         `json:"workspace"`
	PlanStatus   string         `json:"plan_status"`
	Checkpoint   string         `json:"checkpoint"`
	CheckpointAt string         `json:"checkpoint_at"`
}

func checkpointCmd() *cobra.Command {
	var taskID, status, evidence, notes, timestampAt, timestampAfter, dir, workspaceFlag, sessionID string
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "checkpoint",
		Short: "Atomically accept a task lifecycle transition and its evidence",
		RunE: func(cmd *cobra.Command, args []string) error {
			if dir == "" {
				dir = baseDir
			}
			artifactsDir := resolveArtifactsDir(dir)
			if _, err := os.Stat(filepath.Join(artifactsDir, small.SmallDir)); os.IsNotExist(err) {
				return fmt.Errorf(".small/ directory does not exist. Run 'small init' first")
			}
			if sessionv2.IsWorkspace(artifactsDir) {
				return runV2Checkpoint(artifactsDir, sessionID, taskID, status, evidence, notes, jsonOutput)
			}
			scope, err := workspace.ParseScope(workspaceFlag)
			if err != nil {
				return err
			}
			if scope != workspace.ScopeAny {
				if err := enforceWorkspaceScope(artifactsDir, scope); err != nil {
					return err
				}
			}
			status = strings.ToLower(strings.TrimSpace(status))
			if status != "completed" && status != "blocked" {
				return fmt.Errorf("invalid status %q (must be completed or blocked)", status)
			}

			entry, err := commitV1Checkpoint(artifactsDir, taskID, status, evidence, notes, timestampAt, timestampAfter, true)
			if err != nil {
				return err
			}
			checkpointAt := stringVal(entry["timestamp"])
			output := checkpointOutput{
				TaskID: taskID, Status: status, Progress: entry,
				Files: []string{"plan.small.yml", "progress.small.yml"}, Validated: true,
				Workspace: artifactsDir, PlanStatus: status,
				Checkpoint:   fmt.Sprintf("checkpoint: %s -> %s at %s", taskID, status, checkpointAt),
				CheckpointAt: checkpointAt,
			}
			if jsonOutput {
				data, err := json.MarshalIndent(output, "", "  ")
				if err != nil {
					return err
				}
				fmt.Println(string(data))
				return nil
			}
			fmt.Println(output.Checkpoint)
			return nil
		},
	}

	cmd.Flags().StringVar(&taskID, "task", "", "Task ID for the checkpoint")
	cmd.Flags().StringVar(&status, "status", "", "Status (completed or blocked)")
	cmd.Flags().StringVar(&evidence, "evidence", "", "Acceptance evidence for the checkpoint")
	cmd.Flags().StringVar(&notes, "notes", "", "Additional notes for the checkpoint")
	cmd.Flags().StringVar(&timestampAt, "at", "", "Use exact RFC3339Nano timestamp (must be after last entry)")
	cmd.Flags().StringVar(&timestampAfter, "after", "", "Generate timestamp after provided RFC3339Nano time")
	cmd.Flags().StringVar(&dir, "dir", ".", "Directory containing .small/ artifacts")
	cmd.Flags().StringVar(&workspaceFlag, "workspace", string(workspace.ScopeRoot), "Workspace scope (root, examples, or any)")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output in JSON format")
	cmd.Flags().StringVar(&sessionID, "session", "", "v2 session id (defaults to local active selection)")
	_ = cmd.MarkFlagRequired("task")
	_ = cmd.MarkFlagRequired("status")
	return cmd
}

// commitV1Checkpoint is the only v1 path that accepts a task lifecycle result.
// The plan and progress update are validated in memory, then journaled and
// published as one recoverable transaction while the local writer lock is held.
func commitV1Checkpoint(baseDir, taskID, status, evidence, notes, timestampAt, timestampAfter string, defaultEvidence bool) (map[string]any, error) {
	var created map[string]any
	err := small.WithStateLock(baseDir, func() error {
		planPath := filepath.Join(baseDir, small.SmallDir, "plan.small.yml")
		progressPath := filepath.Join(baseDir, small.SmallDir, "progress.small.yml")
		plan, err := loadPlan(planPath)
		if err != nil {
			return fmt.Errorf("failed to load plan.small.yml: %w", err)
		}
		progress, err := loadProgressData(progressPath)
		if err != nil {
			return fmt.Errorf("failed to load progress.small.yml: %w", err)
		}
		lastTimestamp, err := lastProgressTimestamp(progress.Entries)
		if err != nil {
			return fmt.Errorf("existing progress timestamps invalid: %w (run 'small progress migrate' to repair)", err)
		}
		checkpointTimestamp, err := resolveProgressTimestamp(lastTimestamp, timestampAt, timestampAfter)
		if err != nil {
			return err
		}
		if checkpointTimestamp == "" {
			checkpointTimestamp = formatProgressTimestamp(progressTimestampNow().UTC())
		}
		if err := setTaskStatus(plan, strings.TrimSpace(taskID), status); err != nil {
			return err
		}
		entry := map[string]any{"task_id": strings.TrimSpace(taskID), "status": status, "timestamp": checkpointTimestamp}
		if strings.TrimSpace(evidence) != "" {
			entry["evidence"] = evidence
		} else if defaultEvidence {
			entry["evidence"] = "Recorded checkpoint via small checkpoint"
		}
		if strings.TrimSpace(notes) != "" {
			entry["notes"] = notes
		}
		if err := validateProgressEntry(entry); err != nil {
			return err
		}
		if _, ok := entry["evidence"]; !ok {
			return fmt.Errorf("checkpoint evidence is required")
		}
		if _, err := ensureWorkspaceRunReplayID(baseDir); err != nil {
			return err
		}
		attachProgressReplayID(baseDir, entry)
		progress.Entries = append(progress.Entries, entry)
		progress.SmallVersion = small.ProtocolVersion
		progress.Owner = "agent"

		planData, err := small.MarshalYAMLWithQuotedVersion(plan)
		if err != nil {
			return err
		}
		progressData, err := small.MarshalYAMLWithQuotedVersion(&progress)
		if err != nil {
			return err
		}
		if err := validateCheckpointCandidate(baseDir, planData, progressData); err != nil {
			return err
		}
		_, err = small.WriteStateFilesLocked(baseDir, []small.StateFile{
			{Path: filepath.Join(small.SmallDir, "plan.small.yml"), Data: planData, Mode: 0o644},
			{Path: filepath.Join(small.SmallDir, "progress.small.yml"), Data: progressData, Mode: 0o644},
		})
		if err != nil {
			return err
		}
		created = entry
		return nil
	})
	return created, err
}

func validateCheckpointCandidate(baseDir string, planData, progressData []byte) error {
	artifacts, err := small.LoadAllArtifacts(baseDir)
	if err != nil {
		return fmt.Errorf("failed to load artifacts for candidate validation: %w", err)
	}
	for kind, data := range map[string][]byte{"plan": planData, "progress": progressData} {
		var decoded map[string]any
		if err := yaml.Unmarshal(data, &decoded); err != nil {
			return fmt.Errorf("decode candidate %s: %w", kind, err)
		}
		artifacts[kind] = &small.Artifact{Type: kind, Data: decoded, Path: filepath.Join(baseDir, small.SmallDir, kind+".small.yml")}
	}
	if schemaErrs := small.ValidateAllArtifactsWithConfig(artifacts, small.SchemaConfig{BaseDir: baseDir}); len(schemaErrs) > 0 {
		return fmt.Errorf("candidate validation failed: %v", schemaErrs[0])
	}
	if violations := small.CheckInvariants(artifacts, false); len(violations) > 0 {
		return fmt.Errorf("candidate invariant violation: %s", violations[0].Message)
	}
	return nil
}
