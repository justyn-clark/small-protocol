package commands

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/justyn-clark/small-protocol/internal/small"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// ProgressData represents the progress.small.yml structure
type ProgressData struct {
	SmallVersion string                   `yaml:"small_version"`
	Owner        string                   `yaml:"owner"`
	Entries      []map[string]interface{} `yaml:"entries"`
}

func applyCmd() *cobra.Command {
	var (
		cmdArg  string
		handoff bool
		taskID  string
		dryRun  bool
		dir     string
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

			// Check required artifacts exist
			if !small.ArtifactExists(artifactsDir, "progress.small.yml") {
				return fmt.Errorf("progress.small.yml not found. Run 'small init' first")
			}

			// Default to dry-run if no command provided
			if cmdArg == "" {
				dryRun = true
			}

			timestamp := time.Now().UTC().Format(time.RFC3339)

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
			shellCmd.Stdout = os.Stdout
			shellCmd.Stderr = os.Stderr
			shellCmd.Dir = artifactsDir

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
			endTimestamp := time.Now().UTC().Format(time.RFC3339)
			endEntry := map[string]interface{}{
				"timestamp": endTimestamp,
				"task_id":   normalizeTaskID(taskID),
				"status":    status,
				"command":   cmdArg,
			}

			if status == "completed" {
				endEntry["evidence"] = "Command completed successfully"
				endEntry["notes"] = fmt.Sprintf("apply: exit code %d", exitCode)
			} else {
				endEntry["evidence"] = fmt.Sprintf("Command failed with exit code %d", exitCode)
				endEntry["notes"] = fmt.Sprintf("apply: failed with exit code %d", exitCode)
			}

			if err := appendProgressEntry(artifactsDir, endEntry); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to record completion: %v\n", err)
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
	cmd.Flags().StringVar(&dir, "dir", ".", "Directory containing .small/ artifacts")

	return cmd
}

func normalizeTaskID(taskID string) string {
	if taskID == "" {
		return "apply"
	}
	return taskID
}

func appendProgressEntry(baseDir string, entry map[string]interface{}) error {
	progressPath := filepath.Join(baseDir, small.SmallDir, "progress.small.yml")

	// Load existing progress
	data, err := os.ReadFile(progressPath)
	if err != nil {
		return fmt.Errorf("failed to read progress file: %w", err)
	}

	var progress ProgressData
	if err := yaml.Unmarshal(data, &progress); err != nil {
		return fmt.Errorf("failed to parse progress file: %w", err)
	}

	// Append new entry
	progress.Entries = append(progress.Entries, entry)

	// Write back
	yamlData, err := yaml.Marshal(&progress)
	if err != nil {
		return fmt.Errorf("failed to marshal progress: %w", err)
	}

	if err := os.WriteFile(progressPath, yamlData, 0644); err != nil {
		return fmt.Errorf("failed to write progress file: %w", err)
	}

	return nil
}

func generateHandoffFromApply(baseDir string) error {
	// Load plan
	planArtifact, err := small.LoadArtifact(baseDir, "plan.small.yml")
	if err != nil {
		return fmt.Errorf("failed to load plan: %w", err)
	}

	// Load progress
	progressArtifact, err := small.LoadArtifact(baseDir, "progress.small.yml")
	if err != nil {
		return fmt.Errorf("failed to load progress: %w", err)
	}

	// Build handoff structure (simplified version of handoff command)
	handoffData := map[string]interface{}{
		"small_version": "0.1",
		"generated_at":  time.Now().UTC().Format(time.RFC3339),
		"current_plan": map[string]interface{}{
			"tasks": planArtifact.Data["tasks"],
		},
	}

	// Add recent progress (last 10 entries)
	if entries, ok := progressArtifact.Data["entries"].([]interface{}); ok && len(entries) > 0 {
		recent := entries
		if len(entries) > 10 {
			recent = entries[len(entries)-10:]
		}
		handoffData["recent_progress"] = recent
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
