package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/justyn-clark/small-protocol/internal/small"
	"github.com/justyn-clark/small-protocol/internal/workspace"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// ProgressData represents the progress.small.yml structure
// Entries use map[string]interface{} to preserve flexible schema fields.
type ProgressData struct {
	SmallVersion string                   `yaml:"small_version"`
	Owner        string                   `yaml:"owner"`
	Entries      []map[string]interface{} `yaml:"entries"`
}

var progressTimestampNow = time.Now

type progressAddResult struct {
	Entry            map[string]interface{}
	WorkspaceDir     string
	TaskExistsInPlan bool
}

type progressAddOutput struct {
	Workspace        string                 `json:"workspace"`
	Entry            map[string]interface{} `json:"entry"`
	TaskExistsInPlan bool                   `json:"task_exists_in_plan"`
}

func progressCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "progress",
		Short: "Manage progress.small.yml",
		Long:  "Utilities for maintaining progress.small.yml, including timestamp migration.",
	}

	cmd.AddCommand(progressAddCmd())
	cmd.AddCommand(progressMigrateCmd())
	return cmd
}

func progressAddCmd() *cobra.Command {
	var (
		taskID         string
		status         string
		evidence       string
		notes          string
		timestampAt    string
		timestampAfter string
		dir            string
		workspaceFlag  string
		jsonOutput     bool
	)

	cmd := &cobra.Command{
		Use:   "add",
		Short: "Append a progress entry",
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
			if !isValidProgressStatus(status) {
				return fmt.Errorf("invalid status %q (must be pending, in_progress, completed, blocked, or cancelled)", status)
			}

			progressPath := filepath.Join(artifactsDir, small.SmallDir, "progress.small.yml")
			progress, err := loadProgressData(progressPath)
			if err != nil {
				if os.IsNotExist(err) {
					progress = ProgressData{
						SmallVersion: small.ProtocolVersion,
						Owner:        "agent",
						Entries:      []map[string]interface{}{},
					}
				} else {
					return fmt.Errorf("failed to read progress.small.yml: %w", err)
				}
			}

			lastTimestamp, err := lastProgressTimestamp(progress.Entries)
			if err != nil {
				return fmt.Errorf("existing progress timestamps invalid: %w (run 'small progress migrate' to repair)", err)
			}

			timestamp, err := resolveProgressTimestamp(lastTimestamp, timestampAt, timestampAfter)
			if err != nil {
				return err
			}
			if timestamp == "" {
				timestamp = formatProgressTimestamp(progressTimestampNow().UTC())
			}

			entry := map[string]interface{}{
				"task_id":   strings.TrimSpace(taskID),
				"status":    status,
				"timestamp": timestamp,
			}
			if strings.TrimSpace(evidence) != "" {
				entry["evidence"] = evidence
			}
			if strings.TrimSpace(notes) != "" {
				entry["notes"] = notes
			}
			if strings.TrimSpace(evidence) == "" {
				entry["evidence"] = "Recorded progress via small progress add"
			}

			if err := validateProgressEntry(entry); err != nil {
				return err
			}

			originalContent, err := os.ReadFile(progressPath)
			if err != nil {
				if os.IsNotExist(err) {
					originalContent = nil
				} else {
					return fmt.Errorf("failed to read progress.small.yml: %w", err)
				}
			}

			if err := appendProgressEntryWithData(artifactsDir, entry, progress); err != nil {
				return fmt.Errorf("failed to append progress entry: %w", err)
			}

			createdProgress, err := loadProgressData(progressPath)
			if err != nil {
				return fmt.Errorf("failed to reload progress.small.yml: %w", err)
			}
			if len(createdProgress.Entries) > 0 {
				entry = createdProgress.Entries[len(createdProgress.Entries)-1]
			}

			if err := validateProgressArtifact(artifactsDir); err != nil {
				if originalContent == nil {
					_ = os.Remove(progressPath)
				} else {
					_ = os.WriteFile(progressPath, originalContent, 0o644)
				}
				return err
			}

			taskExists, err := taskExistsInPlan(artifactsDir, taskID)
			if err != nil {
				return err
			}

			result := progressAddResult{
				Entry:            entry,
				WorkspaceDir:     artifactsDir,
				TaskExistsInPlan: taskExists,
			}

			if jsonOutput {
				return outputProgressAddJSON(result)
			}

			if !taskExists {
				fmt.Printf("Warning: task %s not found in plan.small.yml\n", taskID)
			}

			fmt.Printf("progress added: %s %s %s\n", taskID, status, stringVal(entry["timestamp"]))
			return nil
		},
	}

	cmd.Flags().StringVar(&taskID, "task", "", "Task ID for the progress entry")
	cmd.Flags().StringVar(&status, "status", "", "Status (pending, in_progress, completed, blocked, cancelled)")
	cmd.Flags().StringVar(&evidence, "evidence", "", "Evidence for the progress entry")
	cmd.Flags().StringVar(&notes, "notes", "", "Additional notes for the progress entry")
	cmd.Flags().StringVar(&timestampAt, "at", "", "Use exact RFC3339Nano timestamp (must be after last entry)")
	cmd.Flags().StringVar(&timestampAfter, "after", "", "Generate timestamp after provided RFC3339Nano time")
	cmd.Flags().StringVar(&dir, "dir", ".", "Directory containing .small/ artifacts")
	cmd.Flags().StringVar(&workspaceFlag, "workspace", string(workspace.ScopeRoot), "Workspace scope (root, examples, or any)")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output in JSON format")

	_ = cmd.MarkFlagRequired("task")
	_ = cmd.MarkFlagRequired("status")

	return cmd
}

func progressMigrateCmd() *cobra.Command {
	var dir string
	var workspaceFlag string

	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "Rewrite progress timestamps to RFC3339Nano",
		Long: `Rewrites progress.small.yml timestamps to RFC3339Nano with fractional seconds.

This command is explicit and only runs when invoked. It preserves entry order
and content, adjusting timestamps only when needed to satisfy the strict
monotonic contract.`,
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

			progressPath := filepath.Join(artifactsDir, small.SmallDir, "progress.small.yml")
			if _, err := os.Stat(progressPath); os.IsNotExist(err) {
				return fmt.Errorf("progress.small.yml not found. Run 'small init' first")
			}

			changed, err := migrateProgressFile(progressPath)
			if err != nil {
				return err
			}

			if changed == 0 {
				fmt.Println("progress.small.yml already satisfies the timestamp contract")
				return nil
			}
			fmt.Printf("Rewrote %d progress timestamp(s) in %s\n", changed, progressPath)
			return nil
		},
	}

	cmd.Flags().StringVar(&dir, "dir", ".", "Directory containing .small/ artifacts")
	cmd.Flags().StringVar(&workspaceFlag, "workspace", string(workspace.ScopeRoot), "Workspace scope (root, examples, or any)")

	return cmd
}

func migrateProgressFile(progressPath string) (int, error) {
	progress, err := loadProgressData(progressPath)
	if err != nil {
		return 0, fmt.Errorf("failed to read progress file: %w", err)
	}

	changed, err := normalizeProgressEntries(progress.Entries)
	if err != nil {
		return 0, err
	}
	if changed == 0 {
		return 0, nil
	}

	progress.SmallVersion = small.ProtocolVersion
	progress.Owner = "agent"

	updated, err := small.MarshalYAMLWithQuotedVersion(&progress)
	if err != nil {
		return 0, fmt.Errorf("failed to marshal progress file: %w", err)
	}
	if err := os.WriteFile(progressPath, updated, 0o644); err != nil {
		return 0, fmt.Errorf("failed to write progress file: %w", err)
	}

	return changed, nil
}

func normalizeProgressEntries(entries []map[string]interface{}) (int, error) {
	var last time.Time
	changed := 0

	for i, entry := range entries {
		tsValue, _ := entry["timestamp"].(string)
		tsValue = strings.TrimSpace(tsValue)
		if tsValue == "" {
			return changed, fmt.Errorf("progress entry %d timestamp is required", i)
		}

		parsed, err := time.Parse(time.RFC3339Nano, tsValue)
		if err != nil {
			return changed, fmt.Errorf("progress entry %d timestamp %q is not parseable: %w", i, tsValue, err)
		}
		parsed = parsed.UTC()

		if !last.IsZero() && !parsed.After(last) {
			parsed = last.Add(time.Nanosecond)
		}

		normalized := formatProgressTimestamp(parsed)
		if tsValue != normalized {
			changed++
		}
		entry["timestamp"] = normalized
		last = parsed
	}

	return changed, nil
}

func attachProgressReplayID(baseDir string, entry map[string]interface{}) {
	if entry == nil {
		return
	}
	if _, ok := entry["replayId"]; ok {
		return
	}
	existing, err := loadExistingHandoff(baseDir)
	if err != nil || existing == nil || existing.ReplayId == nil {
		return
	}
	value := strings.TrimSpace(existing.ReplayId.Value)
	if value == "" {
		return
	}
	entry["replayId"] = value
}

func appendProgressEntry(baseDir string, entry map[string]interface{}) error {
	progressPath := filepath.Join(baseDir, small.SmallDir, "progress.small.yml")

	progress, err := loadProgressData(progressPath)
	if err != nil {
		return fmt.Errorf("failed to read progress file: %w", err)
	}

	lastTimestamp, err := lastProgressTimestamp(progress.Entries)
	if err != nil {
		return fmt.Errorf("existing progress timestamps invalid: %w (run 'small progress migrate' to repair)", err)
	}

	if _, err := normalizeEntryTimestamp(entry, lastTimestamp); err != nil {
		return err
	}

	attachProgressReplayID(baseDir, entry)
	progress.Entries = append(progress.Entries, entry)
	progress.SmallVersion = small.ProtocolVersion
	progress.Owner = "agent"

	yamlData, err := small.MarshalYAMLWithQuotedVersion(&progress)
	if err != nil {
		return fmt.Errorf("failed to marshal progress: %w", err)
	}

	if err := os.WriteFile(progressPath, yamlData, 0o644); err != nil {
		return fmt.Errorf("failed to write progress file: %w", err)
	}

	return nil
}

func appendProgressEntryWithData(baseDir string, entry map[string]interface{}, progress ProgressData) error {
	progressPath := filepath.Join(baseDir, small.SmallDir, "progress.small.yml")

	lastTimestamp, err := lastProgressTimestamp(progress.Entries)
	if err != nil {
		return fmt.Errorf("existing progress timestamps invalid: %w (run 'small progress migrate' to repair)", err)
	}

	if _, err := normalizeEntryTimestamp(entry, lastTimestamp); err != nil {
		return err
	}

	attachProgressReplayID(baseDir, entry)
	progress.Entries = append(progress.Entries, entry)
	progress.SmallVersion = small.ProtocolVersion
	progress.Owner = "agent"

	yamlData, err := small.MarshalYAMLWithQuotedVersion(&progress)
	if err != nil {
		return fmt.Errorf("failed to marshal progress: %w", err)
	}

	if err := os.WriteFile(progressPath, yamlData, 0o644); err != nil {
		return fmt.Errorf("failed to write progress file: %w", err)
	}

	return nil
}

func lastProgressTimestamp(entries []map[string]interface{}) (time.Time, error) {
	if len(entries) == 0 {
		return time.Time{}, nil
	}

	last := entries[len(entries)-1]
	tsValue, _ := last["timestamp"].(string)
	if strings.TrimSpace(tsValue) == "" {
		return time.Time{}, fmt.Errorf("last progress entry timestamp is missing")
	}

	parsed, err := small.ParseProgressTimestamp(tsValue)
	if err != nil {
		return time.Time{}, err
	}

	return parsed.UTC(), nil
}

func normalizeEntryTimestamp(entry map[string]interface{}, last time.Time) (time.Time, error) {
	sValue, _ := entry["timestamp"].(string)
	sValue = strings.TrimSpace(sValue)
	if sValue == "" {
		sValue = formatProgressTimestamp(progressTimestampNow().UTC())
	}

	parsed, err := time.Parse(time.RFC3339Nano, sValue)
	if err != nil {
		return time.Time{}, fmt.Errorf("timestamp %q must be RFC3339Nano: %w", sValue, err)
	}
	parsed = parsed.UTC()

	if !last.IsZero() && !parsed.After(last) {
		parsed = last.Add(time.Nanosecond)
	}

	entry["timestamp"] = formatProgressTimestamp(parsed)
	return parsed, nil
}

func resolveProgressTimestamp(last time.Time, atValue string, afterValue string) (string, error) {
	if strings.TrimSpace(atValue) != "" && strings.TrimSpace(afterValue) != "" {
		return "", fmt.Errorf("only one of --at or --after may be set")
	}

	if strings.TrimSpace(atValue) != "" {
		parsed, err := small.ParseProgressTimestamp(atValue)
		if err != nil {
			return "", fmt.Errorf("--at timestamp %q invalid: %w", atValue, err)
		}
		parsed = parsed.UTC()
		if !last.IsZero() && !parsed.After(last) {
			return "", fmt.Errorf("--at timestamp must be after last progress entry %s", formatProgressTimestamp(last))
		}
		return formatProgressTimestamp(parsed), nil
	}

	if strings.TrimSpace(afterValue) != "" {
		parsed, err := small.ParseProgressTimestamp(afterValue)
		if err != nil {
			return "", fmt.Errorf("--after timestamp %q invalid: %w", afterValue, err)
		}
		parsed = parsed.UTC()
		candidate := parsed.Add(time.Nanosecond)
		if !last.IsZero() && !candidate.After(last) {
			candidate = last.Add(time.Nanosecond)
		}
		if !candidate.After(parsed) {
			candidate = parsed.Add(time.Nanosecond)
		}
		return formatProgressTimestamp(candidate), nil
	}

	return "", nil
}

func formatProgressTimestamp(ts time.Time) string {
	return ts.UTC().Format("2006-01-02T15:04:05.000000000Z")
}

func isValidProgressStatus(status string) bool {
	switch status {
	case "pending", "in_progress", "completed", "blocked", "cancelled":
		return true
	default:
		return false
	}
}

func taskExistsInPlan(baseDir, taskID string) (bool, error) {
	if strings.TrimSpace(taskID) == "" {
		return false, nil
	}
	planPath := filepath.Join(baseDir, small.SmallDir, "plan.small.yml")
	if _, err := os.Stat(planPath); os.IsNotExist(err) {
		return false, nil
	}
	plan, err := loadPlan(planPath)
	if err != nil {
		return false, fmt.Errorf("failed to load plan.small.yml: %w", err)
	}

	_, index := findTask(plan, taskID)
	return index >= 0, nil
}

func validateProgressArtifact(baseDir string) error {
	progressArtifact, err := small.LoadArtifact(baseDir, "progress.small.yml")
	if err != nil {
		return fmt.Errorf("failed to load progress.small.yml: %w", err)
	}
	config := small.SchemaConfig{BaseDir: baseDir}
	if err := small.ValidateArtifactWithConfig(progressArtifact, config); err != nil {
		return fmt.Errorf("validation failed: %v", err)
	}
	violations := small.CheckInvariants(map[string]*small.Artifact{"progress": progressArtifact}, false)
	if len(violations) > 0 {
		return fmt.Errorf("invariant violations found: %s", violations[0].Message)
	}
	return nil
}

func loadProgressData(progressPath string) (ProgressData, error) {
	data, err := os.ReadFile(progressPath)
	if err != nil {
		return ProgressData{}, err
	}

	var progress ProgressData
	if err := yaml.Unmarshal(data, &progress); err != nil {
		return ProgressData{}, err
	}
	if progress.Entries == nil {
		progress.Entries = []map[string]interface{}{}
	}
	return progress, nil
}

func outputProgressAddJSON(result progressAddResult) error {
	payload := progressAddOutput{
		Workspace:        result.WorkspaceDir,
		Entry:            result.Entry,
		TaskExistsInPlan: result.TaskExistsInPlan,
	}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}

func validateProgressEntry(entry map[string]interface{}) error {
	if strings.TrimSpace(stringVal(entry["task_id"])) == "" {
		return fmt.Errorf("task_id is required")
	}
	status := strings.TrimSpace(stringVal(entry["status"]))
	if status != "" && !isValidProgressStatus(status) {
		return fmt.Errorf("invalid status %q", status)
	}

	if !small.ProgressEntryHasValidEvidence(entry) {
		return fmt.Errorf("entry must include evidence or notes")
	}

	timestamp := strings.TrimSpace(stringVal(entry["timestamp"]))
	if timestamp != "" {
		if _, err := small.ParseProgressTimestamp(timestamp); err != nil {
			return fmt.Errorf("timestamp %q invalid: %w", timestamp, err)
		}
	}

	return nil
}
