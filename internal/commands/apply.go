package commands

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/justyn-clark/small-protocol/internal/small"
	"github.com/justyn-clark/small-protocol/internal/workspace"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

func applyCmd() *cobra.Command {
	var (
		cmdArg         string
		handoff        bool
		taskID         string
		dryRun         bool
		autoProgress   bool
		autoCheckpoint bool
		dir            string
		workspaceFlag  string
	)

	cmd := &cobra.Command{
		Use:   "apply",
		Short: "Execute a command bounded by intent and constraints",
		Long: `Executes a user-provided shell command (or runs as dry-run).
Appends progress entries before and after execution.
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
			if autoProgress && dryRun {
				return fmt.Errorf("--auto-progress cannot be used with --dry-run")
			}

			// Default to dry-run if no command provided
			if cmdArg == "" {
				dryRun = true
			}

			timestamp := formatProgressTimestamp(time.Now().UTC())

			if dryRun {
				fmt.Println("Dry-run mode: no command will be executed")
				fmt.Println()

				if cmdArg != "" {
					fmt.Printf("Would execute: %s\n", cmdArg)
				} else {
					fmt.Println("No command specified")
				}

				if taskID != "" {
					fmt.Printf("Would associate with task: %s\n", taskID)
				}

				if handoff {
					fmt.Println("Would generate handoff after execution")
				}

				// Record dry-run in progress
				entry := map[string]interface{}{
					"timestamp": timestamp,
					"task_id":   normalizeTaskID(taskID),
					"status":    "pending",
					"evidence":  "Dry-run: no command executed",
					"notes":     fmt.Sprintf("apply --dry-run (cmd: %q)", cmdArg),
				}

				if err := appendProgressEntry(artifactsDir, entry); err != nil {
					return fmt.Errorf("failed to record progress: %w", err)
				}

				fmt.Println()
				fmt.Println("Recorded dry-run in progress.small.yml")
				return nil
			}

			// Record start entry
			startEntry := map[string]interface{}{
				"timestamp": timestamp,
				"task_id":   normalizeTaskID(taskID),
				"status":    "in_progress",
				"evidence":  "Apply started",
				"command":   cmdArg,
				"notes":     "apply: execution started",
			}

			if err := appendProgressEntry(artifactsDir, startEntry); err != nil {
				return fmt.Errorf("failed to record start: %w", err)
			}

			fmt.Printf("Executing: %s\n", cmdArg)
			fmt.Println()

			// Execute command using sh -lc for portability
			shellCmd := exec.Command("sh", "-lc", cmdArg)
			shellCmd.Dir = artifactsDir

			var outputBuffer bytes.Buffer
			if autoProgress {
				shellCmd.Stdout = &outputBuffer
				shellCmd.Stderr = &outputBuffer
			} else {
				shellCmd.Stdout = os.Stdout
				shellCmd.Stderr = os.Stderr
			}

			cmdErr := shellCmd.Run()
			exitCode := 0
			status := "completed"

			if cmdErr != nil {
				if exitErr, ok := cmdErr.(*exec.ExitError); ok {
					exitCode = exitErr.ExitCode()
				} else {
					exitCode = 1
				}
				status = "blocked"
			}

			// Record completion entry
			endTimestamp := formatProgressTimestamp(time.Now().UTC())
			endEntry := map[string]interface{}{
				"timestamp": endTimestamp,
				"task_id":   normalizeTaskID(taskID),
				"status":    status,
				"command":   cmdArg,
			}

			if autoProgress {
				endEntry["evidence"] = buildAutoProgressEvidence(outputBuffer.String(), exitCode)
				endEntry["notes"] = fmt.Sprintf("apply: exit code %d", exitCode)
			} else if status == "completed" {
				endEntry["evidence"] = "Command completed successfully"
				endEntry["notes"] = fmt.Sprintf("apply: exit code %d", exitCode)
			} else {
				endEntry["evidence"] = fmt.Sprintf("Command failed with exit code %d", exitCode)
				endEntry["notes"] = fmt.Sprintf("apply: failed with exit code %d", exitCode)
			}

			if err := appendProgressEntry(artifactsDir, endEntry); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to record completion: %v\n", err)
			}

			if autoCheckpoint {
				if err := ensureCheckpointTask(taskID); err != nil {
					return err
				}
				checkpointStatus := "completed"
				if status != "completed" {
					checkpointStatus = "blocked"
				}
				checkpointEvidence := buildAutoProgressEvidence(outputBuffer.String(), exitCode)
				if err := runCheckpointApply(artifactsDir, taskID, checkpointStatus, checkpointEvidence); err != nil {
					return err
				}
			}

			fmt.Println()
			if status == "completed" {
				fmt.Printf("Command completed successfully (exit code: %d)\n", exitCode)
			} else {
				fmt.Printf("Command failed (exit code: %d)\n", exitCode)
			}

			// Generate handoff if requested and command succeeded
			if handoff && status == "completed" {
				fmt.Println()
				fmt.Println("Generating handoff...")

				// Call handoff generation
				if err := generateHandoffFromApply(artifactsDir); err != nil {
					fmt.Fprintf(os.Stderr, "Warning: failed to generate handoff: %v\n", err)
				} else {
					fmt.Println("Handoff generated")
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

	cmd.Flags().StringVar(&dir, "dir", ".", "Directory containing .small/ artifacts")
	cmd.Flags().StringVar(&workspaceFlag, "workspace", string(workspace.ScopeRoot), "Workspace scope (root or any)")

	return cmd
}

func normalizeTaskID(taskID string) string {
	if taskID == "" {
		return "apply"
	}
	return taskID
}

func generateHandoffFromApply(baseDir string) error {
	// Load plan to derive next steps
	planArtifact, err := small.LoadArtifact(baseDir, "plan.small.yml")
	if err != nil {
		return fmt.Errorf("failed to load plan: %w", err)
	}

	// Build next_steps from pending tasks
	var nextSteps []interface{}
	var currentTaskID string
	if tasks, ok := planArtifact.Data["tasks"].([]interface{}); ok {
		for _, t := range tasks {
			if tm, ok := t.(map[string]interface{}); ok {
				status, _ := tm["status"].(string)
				taskID, _ := tm["id"].(string)
				title, _ := tm["title"].(string)
				if status == "in_progress" && currentTaskID == "" {
					currentTaskID = taskID
				}
				if status == "pending" || status == "in_progress" {
					if title != "" {
						nextSteps = append(nextSteps, title)
					} else if taskID != "" {
						nextSteps = append(nextSteps, taskID)
					}
				}
			}
		}
	}

	// Build handoff structure matching v1.0.0 schema
	handoffData := map[string]interface{}{
		"small_version": small.ProtocolVersion,
		"owner":         "agent",
		"summary":       "Auto-generated handoff from apply command",
		"resume": map[string]interface{}{
			"current_task_id": currentTaskID,
			"next_steps":      nextSteps,
		},
		"links": []interface{}{},
	}

	// Write handoff
	yamlData, err := yaml.Marshal(handoffData)
	if err != nil {
		return fmt.Errorf("failed to marshal handoff: %w", err)
	}

	handoffPath := filepath.Join(baseDir, small.SmallDir, "handoff.small.yml")
	if err := os.WriteFile(handoffPath, yamlData, 0644); err != nil {
		return fmt.Errorf("failed to write handoff: %w", err)
	}

	return nil
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

	entry := map[string]interface{}{
		"task_id":   taskID,
		"status":    status,
		"timestamp": formatProgressTimestamp(time.Now().UTC()),
	}
	if strings.TrimSpace(evidence) != "" {
		entry["evidence"] = evidence
	}
	if err := validateProgressEntry(entry); err != nil {
		return err
	}

	if err := appendProgressEntryWithData(baseDir, entry, progress); err != nil {
		return err
	}

	if err := savePlan(planPath, plan); err != nil {
		_ = os.WriteFile(planPath, originalPlanData, 0o644)
		_ = os.WriteFile(progressPath, originalProgressData, 0o644)
		return err
	}

	if err := validateCheckpointArtifacts(baseDir); err != nil {
		_ = os.WriteFile(planPath, originalPlanData, 0o644)
		_ = os.WriteFile(progressPath, originalProgressData, 0o644)
		return err
	}

	return nil
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
