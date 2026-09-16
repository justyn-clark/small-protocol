package commands

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/justyn-clark/small-protocol/internal/sessionv2"
	"github.com/justyn-clark/small-protocol/internal/small"
	"github.com/justyn-clark/small-protocol/internal/workspace"
	"github.com/spf13/cobra"
)

func applyCmd() *cobra.Command {
	var (
		cmdArg             string
		handoff            bool
		taskID             string
		dryRun             bool
		autoProgress       bool
		autoCheckpoint     bool
		dir                string
		workspaceFlag      string
		jsonOutput         bool
		acceptanceEvidence string
		sessionID          string
	)

	cmd := &cobra.Command{
		Use:   "apply",
		Short: "Execute a command bounded by intent and constraints",
		Long: `Executes a user-provided shell command (or runs as dry-run).
Writes progress entries using signal-first mode by default.
Set SMALL_PROGRESS_MODE=audit to retain verbose apply telemetry.
Optionally generates a handoff at the end.

If no command is provided, defaults to dry-run mode.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if dir == "" {
				dir = baseDir
			}
			artifactsDir := resolveArtifactsDir(dir)
			smallDir := filepath.Join(artifactsDir, small.SmallDir)

			// Check if .small directory exists
			if _, err := os.Stat(smallDir); os.IsNotExist(err) {
				return fmt.Errorf(".small/ directory does not exist. Run 'small init' first")
			}
			if sessionv2.IsWorkspace(artifactsDir) {
				return runV2Apply(artifactsDir, sessionID, taskID, cmdArg, dryRun || cmdArg == "", autoCheckpoint, acceptanceEvidence, handoff, jsonOutput)
			}

			scope, err := workspace.ParseScope(workspaceFlag)
			if err != nil {
				return err
			}
			if scope == workspace.ScopeExamples {
				return fmt.Errorf("--workspace examples is not supported for apply (use --workspace any to bypass)")
			}
			if scope != workspace.ScopeAny {
				if err := enforceWorkspaceScope(artifactsDir, workspace.ScopeRoot); err != nil {
					return err
				}
			}

			// Check required artifacts exist

			if !small.ArtifactExists(artifactsDir, "progress.small.yml") {
				return fmt.Errorf("progress.small.yml not found. Run 'small init' first")
			}

			if autoCheckpoint && taskID == "" {
				return fmt.Errorf("--auto-checkpoint requires --task")
			}
			if autoCheckpoint && !small.ArtifactExists(artifactsDir, "plan.small.yml") {
				return fmt.Errorf("plan.small.yml not found. Run 'small init' first")
			}
			if autoCheckpoint && dryRun {
				return fmt.Errorf("--auto-checkpoint cannot be used with --dry-run")
			}
			if autoCheckpoint {
				requiresEvidence, err := taskRequiresAcceptanceEvidence(artifactsDir, taskID)
				if err != nil {
					return err
				}
				if requiresEvidence && strings.TrimSpace(acceptanceEvidence) == "" {
					return fmt.Errorf("--auto-checkpoint cannot accept task %s because it has acceptance criteria; provide --acceptance-evidence or run small checkpoint explicitly", taskID)
				}
			}
			if autoProgress && dryRun {
				return fmt.Errorf("--auto-progress cannot be used with --dry-run")
			}

			// Default to dry-run if no command provided
			if cmdArg == "" {
				dryRun = true
			}

			mode := resolveProgressMode()
			normalizedTaskID := normalizeTaskID(taskID)
			timestamp := formatProgressTimestamp(time.Now().UTC())

			if dryRun {
				if !jsonOutput {
					fmt.Println("Dry-run mode: no command will be executed")
					fmt.Println()
				}

				if !jsonOutput && cmdArg != "" {
					fmt.Printf("Would execute: %s\n", cmdArg)
				} else if !jsonOutput {
					fmt.Println("No command specified")
				}

				if !jsonOutput && taskID != "" {
					fmt.Printf("Would associate with task: %s\n", taskID)
				}

				if !jsonOutput && handoff {
					fmt.Println("Would generate handoff after execution")
				}

				// Record dry-run in progress
				entry := map[string]any{
					"timestamp": timestamp,
					"task_id":   normalizedTaskID,
					"status":    "pending",
					"evidence":  "Dry-run: no command executed",
					"notes":     "apply --dry-run",
				}

				emitDryRunProgress := shouldEmitProgress(progressEventApplyDryRun, normalizedTaskID, mode)
				if emitDryRunProgress && cmdArg != "" {
					summary, ref, sha, err := applyCommandMetadata(artifactsDir, timestamp, cmdArg)
					if err != nil {
						return err
					}
					entry["command"] = summary
					entry["command_summary"] = summary
					entry["command_ref"] = ref
					entry["command_sha256"] = sha
					entry["notes"] = fmt.Sprintf("apply --dry-run (cmd: %q)", summary)
				}

				if emitDryRunProgress {
					if err := appendProgressEntry(artifactsDir, entry); err != nil {
						return fmt.Errorf("failed to record progress: %w", err)
					}

					if !jsonOutput {
						fmt.Println()
						fmt.Println("Recorded dry-run in progress.small.yml")
					}
				}
				if jsonOutput {
					return printApplyJSON(applyOutput{Command: small.SummarizeCommand(cmdArg, small.DefaultCommandSummaryCap), TaskID: normalizedTaskID, ChildOutcome: "not_executed", EvidenceRecorded: emitDryRunProgress, DryRun: true})
				}
				return nil
			}

			emitStartProgress := shouldEmitProgress(progressEventApplyStart, normalizedTaskID, mode)
			if emitStartProgress {
				startEntry := map[string]any{
					"timestamp": timestamp,
					"task_id":   normalizedTaskID,
					"status":    "in_progress",
					"evidence":  "Apply started",
					"notes":     "apply: execution started",
				}

				if cmdArg != "" {
					summary, ref, sha, err := applyCommandMetadata(artifactsDir, timestamp, cmdArg)
					if err != nil {
						return err
					}
					startEntry["command"] = summary
					startEntry["command_summary"] = summary
					startEntry["command_ref"] = ref
					startEntry["command_sha256"] = sha
				}

				if err := appendProgressEntry(artifactsDir, startEntry); err != nil {
					return fmt.Errorf("failed to record start: %w", err)
				}
			}

			if !jsonOutput {
				fmt.Printf("Executing: %s\n", cmdArg)
				fmt.Println()
			}

			// Execute command using sh -lc for portability
			shellCmd := exec.Command("sh", "-lc", cmdArg)
			shellCmd.Dir = artifactsDir

			var outputBuffer bytes.Buffer
			if autoProgress || jsonOutput {
				shellCmd.Stdout = &outputBuffer
				shellCmd.Stderr = &outputBuffer
			} else {
				shellCmd.Stdout = os.Stdout
				shellCmd.Stderr = os.Stderr
			}

			cmdErr := shellCmd.Run()
			exitCode := 0
			outcome := "succeeded"

			if cmdErr != nil {
				if exitErr, ok := cmdErr.(*exec.ExitError); ok {
					exitCode = exitErr.ExitCode()
				} else {
					exitCode = 1
				}
				outcome = "failed"
			}

			emitEndProgress := shouldEmitProgress(progressEventApplyComplete, normalizedTaskID, mode)

			// Record completion entry
			endTimestamp := formatProgressTimestamp(time.Now().UTC())
			endEntry := map[string]any{
				"timestamp": endTimestamp,
				"task_id":   normalizedTaskID,
				"status":    "in_progress",
			}
			var recordErr error
			if emitEndProgress && cmdArg != "" {
				summary, ref, sha, err := applyCommandMetadata(artifactsDir, endTimestamp, cmdArg)
				if err != nil {
					recordErr = err
				} else {
					endEntry["command"] = summary
					endEntry["command_summary"] = summary
					endEntry["command_ref"] = ref
					endEntry["command_sha256"] = sha
				}
			}
			endEntry["evidence"] = buildExecutionEvidence(outputBuffer.String(), exitCode, autoProgress || jsonOutput)
			endEntry["notes"] = fmt.Sprintf("apply: command outcome %s; task acceptance unchanged", outcome)

			if emitEndProgress && recordErr == nil {
				if err := appendProgressEntry(artifactsDir, endEntry); err != nil {
					recordErr = err
				}
			}
			if recordErr != nil {
				result := applyOutput{Command: small.SummarizeCommand(cmdArg, small.DefaultCommandSummaryCap), TaskID: normalizedTaskID, ChildOutcome: outcome, ChildExitCode: exitCode, ChildExecuted: true, EvidenceRecorded: false, ErrorCode: "executed_but_unrecorded"}
				if jsonOutput {
					_ = printApplyJSON(result)
				}
				return fmt.Errorf("command executed with exit code %d but durable evidence recording failed; command was not retried: %w", exitCode, recordErr)
			}

			checkpointed := false
			if autoCheckpoint {
				if err := ensureCheckpointTask(taskID); err != nil {
					return err
				}
				checkpointStatus := "completed"
				if outcome != "succeeded" {
					checkpointStatus = "blocked"
				}
				checkpointEvidence := strings.TrimSpace(acceptanceEvidence)
				if checkpointEvidence == "" {
					checkpointEvidence = fmt.Sprintf("Explicit auto-checkpoint of captured command outcome: exit_code=%d", exitCode)
				}
				if err := runCheckpointApply(artifactsDir, taskID, checkpointStatus, checkpointEvidence); err != nil {
					return err
				}
				checkpointed = true
			}

			if !jsonOutput {
				fmt.Println()
			}
			if !jsonOutput && outcome == "succeeded" {
				fmt.Printf("Command completed successfully (exit code: %d)\n", exitCode)
			} else if !jsonOutput {
				fmt.Printf("Command failed (exit code: %d)\n", exitCode)
			}

			// Generate handoff if requested and command succeeded
			if handoff && outcome == "succeeded" {
				fmt.Println()
				fmt.Println("Generating handoff...")

				// Call handoff generation
				if err := generateHandoffFromApply(artifactsDir); err != nil {
					return fmt.Errorf("command succeeded but requested handoff persistence failed: %w", err)
				} else {
					fmt.Println("Handoff generated")
				}
			}
			if jsonOutput {
				if err := printApplyJSON(applyOutput{Command: small.SummarizeCommand(cmdArg, small.DefaultCommandSummaryCap), TaskID: normalizedTaskID, ChildOutcome: outcome, ChildExitCode: exitCode, ChildExecuted: true, EvidenceRecorded: emitEndProgress, Checkpointed: checkpointed}); err != nil {
					return err
				}
			}

			if cmdErr != nil {
				os.Exit(exitCode)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&cmdArg, "cmd", "", "Shell command to execute")
	cmd.Flags().BoolVar(&handoff, "handoff", false, "Generate handoff after successful execution")
	cmd.Flags().StringVar(&taskID, "task", "", "Associate this apply run with a specific task ID")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Do not execute, only record intent")
	cmd.Flags().BoolVar(&autoProgress, "auto-progress", false, "Capture output in progress evidence")
	cmd.Flags().BoolVar(&autoCheckpoint, "auto-checkpoint", false, "Checkpoint the task based on command result")
	cmd.Flags().StringVar(&acceptanceEvidence, "acceptance-evidence", "", "Explicit acceptance evidence for --auto-checkpoint")
	cmd.Flags().StringVar(&sessionID, "session", "", "v2 session id (defaults to local active selection)")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output child and persistence outcomes in JSON format")

	cmd.Flags().StringVar(&dir, "dir", ".", "Directory containing .small/ artifacts")
	cmd.Flags().StringVar(&workspaceFlag, "workspace", string(workspace.ScopeRoot), "Workspace scope (root or any)")

	return cmd
}

type applyOutput struct {
	Command          string `json:"command,omitempty"`
	TaskID           string `json:"task_id"`
	ChildOutcome     string `json:"child_outcome"`
	ChildExitCode    int    `json:"child_exit_code"`
	ChildExecuted    bool   `json:"child_executed"`
	EvidenceRecorded bool   `json:"evidence_recorded"`
	Checkpointed     bool   `json:"checkpointed"`
	DryRun           bool   `json:"dry_run,omitempty"`
	ErrorCode        string `json:"error_code,omitempty"`
}

func printApplyJSON(output applyOutput) error {
	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}

func buildExecutionEvidence(output string, exitCode int, includeOutput bool) map[string]any {
	evidence := map[string]any{"kind": "cli_execution", "strength": "cli_captured", "exit_code": exitCode, "outcome": "succeeded"}
	if exitCode != 0 {
		evidence["outcome"] = "failed"
	}
	if includeOutput {
		const maxLen = 4000
		trimmed := strings.TrimRight(output, "\n")
		if len(trimmed) > maxLen {
			trimmed = trimmed[:maxLen]
			evidence["output_truncated"] = true
			evidence["output_limit_bytes"] = maxLen
		}
		if trimmed != "" {
			evidence["bounded_output"] = trimmed
		}
	}
	return evidence
}

func taskRequiresAcceptanceEvidence(baseDir, taskID string) (bool, error) {
	plan, err := loadPlan(filepath.Join(baseDir, small.SmallDir, "plan.small.yml"))
	if err != nil {
		return false, err
	}
	task, _ := findTask(plan, strings.TrimSpace(taskID))
	if task == nil {
		return false, fmt.Errorf("task %s not found", taskID)
	}
	return len(task.Acceptance) > 0, nil
}

func normalizeTaskID(taskID string) string {
	if taskID == "" {
		return "apply"
	}
	return taskID
}

func generateHandoffFromApply(baseDir string) error {
	handoff, err := buildHandoff(baseDir, "Auto-generated handoff from apply command", "", nil, nil, nil, defaultNextStepsLimit)
	if err != nil {
		return err
	}
	if err := setWorkspaceRunReplayIDIfPresent(baseDir, handoff.ReplayId.Value); err != nil {
		return err
	}

	return writeHandoff(baseDir, handoff)
}

func applyCommandMetadata(baseDir, timestamp, command string) (string, string, string, error) {
	summary := small.SummarizeCommand(command, small.DefaultCommandSummaryCap)
	replayId, err := currentWorkspaceRunReplayID(baseDir)
	if err != nil {
		return "", "", "", err
	}
	if replayId == "" {
		existing, loadErr := loadExistingHandoff(baseDir)
		if loadErr == nil && existing != nil && existing.ReplayId != nil {
			replayId = strings.TrimSpace(existing.ReplayId.Value)
		}
	}
	if replayId == "" {
		return "", "", "", fmt.Errorf("cannot record command log: replayId missing (run small plan --add or small checkpoint)")
	}
	ref, sha, err := small.WriteCommandLog(baseDir, replayId, timestamp, command)
	if err != nil {
		return "", "", "", fmt.Errorf("failed to write command log: %w", err)
	}
	return summary, ref, sha, nil
}

func buildAutoProgressEvidence(output string, exitCode int) string {
	const maxLen = 4000
	trimmed := strings.TrimRight(output, "\n")
	truncated := false
	if len(trimmed) > maxLen {
		trimmed = trimmed[:maxLen]
		truncated = true
	}

	payload := fmt.Sprintf("exit_code=%d", exitCode)
	if strings.TrimSpace(trimmed) != "" {
		payload = payload + " output=\"" + trimmed + "\""
	}
	if truncated {
		payload = payload + fmt.Sprintf(" truncated=true limit=%d", maxLen)
	}
	return payload
}

func ensureCheckpointTask(taskID string) error {
	if taskID == "" {
		return fmt.Errorf("checkpoint requires --task")
	}
	return nil
}

func runCheckpointApply(baseDir, taskID, status string, evidence string) error {
	if status != "completed" && status != "blocked" {
		return fmt.Errorf("checkpoint status must be completed or blocked")
	}

	_, err := commitV1Checkpoint(baseDir, taskID, status, evidence, "apply --auto-checkpoint", "", "", false)
	return err
}

func validateCheckpointArtifacts(baseDir string) error {
	artifacts, err := small.LoadAllArtifacts(baseDir)
	if err != nil {
		return fmt.Errorf("failed to load artifacts: %w", err)
	}
	config := small.SchemaConfig{BaseDir: baseDir}
	errors := small.ValidateAllArtifactsWithConfig(artifacts, config)
	if len(errors) > 0 {
		return fmt.Errorf("validation failed: %v", errors[0])
	}
	violations := small.CheckInvariants(artifacts, false)
	if len(violations) > 0 {
		return fmt.Errorf("invariant violations found: %s", violations[0].Message)
	}
	return nil
}
