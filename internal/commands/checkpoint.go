package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

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
	var taskID string
	var status string
	var evidence string
	var notes string
	var timestampAt string
	var timestampAfter string
	var dir string
	var workspaceFlag string
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "checkpoint",
		Short: "Update plan and progress in one step",
		RunE: func(cmd *cobra.Command, args []string) error {
			if dir == "" {
				dir = baseDir
			}
			artifactsDir := resolveArtifactsDir(dir)
			smallDir := filepath.Join(artifactsDir, small.SmallDir)

			if _, err := os.Stat(smallDir); os.IsNotExist(err) {
				return fmt.Errorf(".small/ directory does not exist. Run 'small init' first")
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

			planPath := filepath.Join(smallDir, "plan.small.yml")
			progressPath := filepath.Join(smallDir, "progress.small.yml")

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

			originalPlanData, err := yaml.Marshal(plan)
			if err != nil {
				return fmt.Errorf("failed to snapshot plan.small.yml: %w", err)
			}
			originalProgressData, err := yaml.Marshal(&progress)
			if err != nil {
				return fmt.Errorf("failed to snapshot progress.small.yml: %w", err)
			}

			if err := setTaskStatus(plan, taskID, status); err != nil {
				return err
			}

			entry := map[string]any{
				"task_id":   strings.TrimSpace(taskID),
				"status":    status,
				"timestamp": checkpointTimestamp,
			}
			if strings.TrimSpace(evidence) != "" {
				entry["evidence"] = evidence
			}
			if strings.TrimSpace(notes) != "" {
				entry["notes"] = notes
			}
			if strings.TrimSpace(evidence) == "" {
				entry["evidence"] = "Recorded checkpoint via small checkpoint"
			}

			if err := validateProgressEntry(entry); err != nil {
				return err
			}

			if _, err := ensureWorkspaceRunReplayID(artifactsDir); err != nil {
				return err
			}

			if err := appendProgressEntryWithData(artifactsDir, entry, progress); err != nil {
				_ = os.WriteFile(planPath, originalPlanData, 0o644)
				_ = os.WriteFile(progressPath, originalProgressData, 0o644)
				return err
			}

			if err := savePlan(planPath, plan); err != nil {
				_ = os.WriteFile(planPath, originalPlanData, 0o644)
				_ = os.WriteFile(progressPath, originalProgressData, 0o644)
				return err
			}

			if err := validateCheckpointArtifacts(artifactsDir); err != nil {
				_ = os.WriteFile(planPath, originalPlanData, 0o644)
				_ = os.WriteFile(progressPath, originalProgressData, 0o644)
				return err
			}

			createdProgress, err := loadProgressData(progressPath)
			if err != nil {
				return fmt.Errorf("failed to reload progress.small.yml: %w", err)
			}
			if len(createdProgress.Entries) > 0 {
				entry = createdProgress.Entries[len(createdProgress.Entries)-1]
			}

			checkpointTimestamp = stringVal(entry["timestamp"])
			output := checkpointOutput{
				TaskID:       taskID,
				Status:       status,
				Progress:     entry,
				Files:        []string{"plan.small.yml", "progress.small.yml"},
				Validated:    true,
				Workspace:    artifactsDir,
				PlanStatus:   status,
				Checkpoint:   fmt.Sprintf("checkpoint: %s -> %s at %s", taskID, status, checkpointTimestamp),
				CheckpointAt: checkpointTimestamp,
			}

			if jsonOutput {
				data, err := json.MarshalIndent(output, "", "  ")
				if err != nil {
					return err
				}
				fmt.Println(string(data))
				return nil
			}

			fmt.Printf("checkpoint: %s -> %s at %s\n", taskID, status, stringVal(entry["timestamp"]))
			return nil
		},
	}

	cmd.Flags().StringVar(&taskID, "task", "", "Task ID for the checkpoint")
	cmd.Flags().StringVar(&status, "status", "", "Status (completed or blocked)")
	cmd.Flags().StringVar(&evidence, "evidence", "", "Evidence for the checkpoint")
	cmd.Flags().StringVar(&notes, "notes", "", "Additional notes for the checkpoint")
	cmd.Flags().StringVar(&timestampAt, "at", "", "Use exact RFC3339Nano timestamp (must be after last entry)")
	cmd.Flags().StringVar(&timestampAfter, "after", "", "Generate timestamp after provided RFC3339Nano time")
	cmd.Flags().StringVar(&dir, "dir", ".", "Directory containing .small/ artifacts")
	cmd.Flags().StringVar(&workspaceFlag, "workspace", string(workspace.ScopeRoot), "Workspace scope (root, examples, or any)")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output in JSON format")

	_ = cmd.MarkFlagRequired("task")
	_ = cmd.MarkFlagRequired("status")

	return cmd
}
