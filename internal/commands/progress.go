package commands

import (
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

func progressCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "progress",
		Short: "Manage progress.small.yml",
		Long:  "Utilities for maintaining progress.small.yml, including timestamp migration.",
	}

	cmd.AddCommand(progressMigrateCmd())
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
	data, err := os.ReadFile(progressPath)
	if err != nil {
		return 0, fmt.Errorf("failed to read progress file: %w", err)
	}

	var progress ProgressData
	if err := yaml.Unmarshal(data, &progress); err != nil {
		return 0, fmt.Errorf("failed to parse progress file: %w", err)
	}
	if progress.Entries == nil {
		progress.Entries = []map[string]interface{}{}
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

	updated, err := yaml.Marshal(&progress)
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

func appendProgressEntry(baseDir string, entry map[string]interface{}) error {
	progressPath := filepath.Join(baseDir, small.SmallDir, "progress.small.yml")

	data, err := os.ReadFile(progressPath)
	if err != nil {
		return fmt.Errorf("failed to read progress file: %w", err)
	}

	var progress ProgressData
	if err := yaml.Unmarshal(data, &progress); err != nil {
		return fmt.Errorf("failed to parse progress file: %w", err)
	}
	if progress.Entries == nil {
		progress.Entries = []map[string]interface{}{}
	}

	lastTimestamp, err := lastProgressTimestamp(progress.Entries)
	if err != nil {
		return fmt.Errorf("existing progress timestamps invalid: %w (run 'small progress migrate' to repair)", err)
	}

	if _, err := normalizeEntryTimestamp(entry, lastTimestamp); err != nil {
		return err
	}

	progress.Entries = append(progress.Entries, entry)
	progress.SmallVersion = small.ProtocolVersion
	progress.Owner = "agent"

	yamlData, err := yaml.Marshal(&progress)
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
	tsValue, _ := entry["timestamp"].(string)
	tsValue = strings.TrimSpace(tsValue)
	if tsValue == "" {
		tsValue = formatProgressTimestamp(time.Now().UTC())
	}

	parsed, err := time.Parse(time.RFC3339Nano, tsValue)
	if err != nil {
		return time.Time{}, fmt.Errorf("timestamp %q must be RFC3339Nano: %w", tsValue, err)
	}
	parsed = parsed.UTC()

	if !last.IsZero() && !parsed.After(last) {
		parsed = last.Add(time.Nanosecond)
	}

	entry["timestamp"] = formatProgressTimestamp(parsed)
	return parsed, nil
}

func formatProgressTimestamp(ts time.Time) string {
	return ts.UTC().Format("2006-01-02T15:04:05.000000000Z")
}
