package commands

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/justyn-clark/small-protocol/internal/runstore"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

type runFileDiff struct {
	Name    string `json:"name"`
	Changed bool   `json:"changed"`
	Added   int    `json:"added"`
	Removed int    `json:"removed"`
	Diff    string `json:"diff,omitempty"`
}

type runProgressDiff struct {
	EntriesBefore  int      `json:"entries_before"`
	EntriesAfter   int      `json:"entries_after"`
	EntryDelta     int      `json:"entry_delta"`
	CompletedDelta []string `json:"completed_delta,omitempty"`
	LatestBefore   string   `json:"latest_before,omitempty"`
	LatestAfter    string   `json:"latest_after,omitempty"`
	Diff           string   `json:"diff,omitempty"`
}

type runDiffOutput struct {
	From     string          `json:"from"`
	To       string          `json:"to"`
	Files    []runFileDiff   `json:"files"`
	Progress runProgressDiff `json:"progress"`
}

func runDiffCmd(dir, storeFlag, workspaceFlag *string) *cobra.Command {
	var (
		full       bool
		jsonOutput bool
	)

	cmd := &cobra.Command{
		Use:   "diff <fromReplayId> <toReplayId>",
		Short: "Diff two run snapshots",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			artifactsDir, storeDir, err := resolveRunContext(*dir, *storeFlag, *workspaceFlag)
			if err != nil {
				return err
			}
			_ = artifactsDir

			fromSnapshot, err := runstore.LoadSnapshot(storeDir, args[0])
			if err != nil {
				return err
			}
			toSnapshot, err := runstore.LoadSnapshot(storeDir, args[1])
			if err != nil {
				return err
			}

			result, err := buildRunDiff(fromSnapshot, toSnapshot, full)
			if err != nil {
				return err
			}

			output, err := formatRunDiffOutput(result, jsonOutput)
			if err != nil {
				return err
			}
			fmt.Print(output)
			return nil
		},
	}

	cmd.Flags().BoolVar(&full, "full", false, "Include full progress diff")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output in JSON format")
	return cmd
}

func buildRunDiff(fromSnapshot, toSnapshot *runstore.Snapshot, full bool) (runDiffOutput, error) {
	files := []runFileDiff{}
	for _, filename := range diffFiles() {
		fromText, fromExists, err := readSnapshotFile(fromSnapshot.Dir, filename)
		if err != nil {
			return runDiffOutput{}, err
		}
		toText, toExists, err := readSnapshotFile(toSnapshot.Dir, filename)
		if err != nil {
			return runDiffOutput{}, err
		}
		if !fromExists && !toExists {
			continue
		}

		diffText, added, removed, changed := unifiedDiff(filename, fromText, toText, fromExists, toExists)
		fileDiff := runFileDiff{
			Name:    filename,
			Changed: changed,
			Added:   added,
			Removed: removed,
			Diff:    diffText,
		}
		files = append(files, fileDiff)
	}

	progressDiff, err := diffProgress(fromSnapshot.Dir, toSnapshot.Dir, full)
	if err != nil {
		return runDiffOutput{}, err
	}

	return runDiffOutput{
		From:     fromSnapshot.ReplayID,
		To:       toSnapshot.ReplayID,
		Files:    files,
		Progress: progressDiff,
	}, nil
}

func formatRunDiffOutput(result runDiffOutput, jsonOutput bool) (string, error) {
	if jsonOutput {
		data, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return "", err
		}
		return string(data) + "\n", nil
	}

	var buffer bytes.Buffer
	writeLine := func(format string, args ...interface{}) {
		_, _ = fmt.Fprintf(&buffer, format, args...)
	}

	writeLine("Run diff: %s -> %s\n", shortID(result.From, 16), shortID(result.To, 16))
	writeLine("\n")

	for _, file := range result.Files {
		if !file.Changed {
			continue
		}
		writeLine("diff %s\n", file.Name)
		writeLine("%s\n", file.Diff)
	}

	writeLine("Progress summary:\n")
	writeLine("  entries: %d -> %d (delta %+d)\n", result.Progress.EntriesBefore, result.Progress.EntriesAfter, result.Progress.EntryDelta)
	if len(result.Progress.CompletedDelta) > 0 {
		writeLine("  completed tasks added: %s\n", strings.Join(result.Progress.CompletedDelta, ", "))
	} else {
		writeLine("  completed tasks added: none\n")
	}
	if result.Progress.LatestBefore != "" || result.Progress.LatestAfter != "" {
		writeLine("  latest timestamps: %s -> %s\n", result.Progress.LatestBefore, result.Progress.LatestAfter)
	}
	if result.Progress.Diff != "" {
		writeLine("\n")
		writeLine("diff progress.small.yml\n")
		writeLine("%s\n", result.Progress.Diff)
	}

	return buffer.String(), nil
}

func diffFiles() []string {
	return []string{
		"intent.small.yml",
		"constraints.small.yml",
		"plan.small.yml",
		"handoff.small.yml",
	}
}

func diffProgress(fromDir, toDir string, full bool) (runProgressDiff, error) {
	fromEntries, err := loadProgressEntries(filepath.Join(fromDir, "progress.small.yml"))
	if err != nil {
		return runProgressDiff{}, err
	}
	toEntries, err := loadProgressEntries(filepath.Join(toDir, "progress.small.yml"))
	if err != nil {
		return runProgressDiff{}, err
	}

	completedDelta := completedTaskDelta(fromEntries, toEntries)
	latestBefore := latestProgressTimestamp(fromEntries)
	latestAfter := latestProgressTimestamp(toEntries)

	progressDiff := runProgressDiff{
		EntriesBefore:  len(fromEntries),
		EntriesAfter:   len(toEntries),
		EntryDelta:     len(toEntries) - len(fromEntries),
		CompletedDelta: completedDelta,
		LatestBefore:   latestBefore,
		LatestAfter:    latestAfter,
	}

	if full {
		fromText, fromExists, err := readSnapshotFile(fromDir, "progress.small.yml")
		if err != nil {
			return runProgressDiff{}, err
		}
		toText, toExists, err := readSnapshotFile(toDir, "progress.small.yml")
		if err != nil {
			return runProgressDiff{}, err
		}
		diffText, _, _, changed := unifiedDiff("progress.small.yml", fromText, toText, fromExists, toExists)
		if changed {
			progressDiff.Diff = diffText
		}
	}

	return progressDiff, nil
}

func loadProgressEntries(path string) ([]map[string]interface{}, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []map[string]interface{}{}, nil
		}
		return nil, fmt.Errorf("failed to read progress.small.yml: %w", err)
	}

	var progress ProgressData
	if err := yaml.Unmarshal(data, &progress); err != nil {
		return nil, fmt.Errorf("failed to parse progress.small.yml: %w", err)
	}
	if progress.Entries == nil {
		return []map[string]interface{}{}, nil
	}
	return progress.Entries, nil
}

func completedTaskDelta(before, after []map[string]interface{}) []string {
	beforeSet := make(map[string]struct{})
	for _, entry := range before {
		if status := strings.TrimSpace(stringVal(entry["status"])); status == "completed" {
			if taskID := strings.TrimSpace(stringVal(entry["task_id"])); taskID != "" {
				beforeSet[taskID] = struct{}{}
			}
		}
	}

	afterSet := make(map[string]struct{})
	for _, entry := range after {
		if status := strings.TrimSpace(stringVal(entry["status"])); status == "completed" {
			if taskID := strings.TrimSpace(stringVal(entry["task_id"])); taskID != "" {
				afterSet[taskID] = struct{}{}
			}
		}
	}

	var delta []string
	for taskID := range afterSet {
		if _, ok := beforeSet[taskID]; !ok {
			delta = append(delta, taskID)
		}
	}
	if len(delta) == 0 {
		return delta
	}
	return sortStrings(delta)
}

func latestProgressTimestamp(entries []map[string]interface{}) string {
	var latest time.Time
	var latestRaw string
	for _, entry := range entries {
		value := strings.TrimSpace(stringVal(entry["timestamp"]))
		if value == "" {
			continue
		}
		parsed, err := time.Parse(time.RFC3339Nano, value)
		if err != nil {
			if latestRaw == "" {
				latestRaw = value
			}
			continue
		}
		if parsed.After(latest) {
			latest = parsed
			latestRaw = value
		}
	}
	return latestRaw
}

func sortStrings(values []string) []string {
	copyValues := append([]string{}, values...)
	sort.Strings(copyValues)
	return copyValues
}

func readSnapshotFile(dir, filename string) (string, bool, error) {
	path := filepath.Join(dir, filename)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", false, nil
		}
		return "", false, fmt.Errorf("failed to read %s: %w", filename, err)
	}
	return string(data), true, nil
}

type diffOp struct {
	kind diffKind
	line string
}

type diffKind int

const (
	diffEqual diffKind = iota
	diffAdd
	diffDelete
)

func unifiedDiff(name, fromText, toText string, fromExists, toExists bool) (string, int, int, bool) {
	if fromExists && toExists && fromText == toText {
		return "", 0, 0, false
	}

	fromLines := splitLines(fromText)
	toLines := splitLines(toText)
	ops := diffOps(fromLines, toLines)
	if len(ops) == 0 && !fromExists && !toExists {
		return "", 0, 0, false
	}

	var buffer bytes.Buffer
	_, _ = fmt.Fprintf(&buffer, "--- a/%s\n", name)
	_, _ = fmt.Fprintf(&buffer, "+++ b/%s\n", name)
	_, _ = fmt.Fprintf(&buffer, "@@ -1,%d +1,%d @@\n", len(fromLines), len(toLines))

	added := 0
	removed := 0
	for _, op := range ops {
		switch op.kind {
		case diffEqual:
			buffer.WriteString(" " + op.line + "\n")
		case diffAdd:
			added++
			buffer.WriteString("+" + op.line + "\n")
		case diffDelete:
			removed++
			buffer.WriteString("-" + op.line + "\n")
		}
	}

	return buffer.String(), added, removed, true
}

func splitLines(text string) []string {
	lines := strings.Split(text, "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		return lines[:len(lines)-1]
	}
	return lines
}

func diffOps(fromLines, toLines []string) []diffOp {
	fromLen := len(fromLines)
	toLen := len(toLines)
	dp := make([][]int, fromLen+1)
	for i := range dp {
		dp[i] = make([]int, toLen+1)
	}

	for i := fromLen - 1; i >= 0; i-- {
		for j := toLen - 1; j >= 0; j-- {
			if fromLines[i] == toLines[j] {
				dp[i][j] = dp[i+1][j+1] + 1
			} else if dp[i+1][j] >= dp[i][j+1] {
				dp[i][j] = dp[i+1][j]
			} else {
				dp[i][j] = dp[i][j+1]
			}
		}
	}

	i := 0
	j := 0
	var ops []diffOp
	for i < fromLen || j < toLen {
		if i < fromLen && j < toLen && fromLines[i] == toLines[j] {
			ops = append(ops, diffOp{kind: diffEqual, line: fromLines[i]})
			i++
			j++
			continue
		}
		if j < toLen && (i == fromLen || dp[i][j+1] >= dp[i+1][j]) {
			ops = append(ops, diffOp{kind: diffAdd, line: toLines[j]})
			j++
			continue
		}
		if i < fromLen {
			ops = append(ops, diffOp{kind: diffDelete, line: fromLines[i]})
			i++
		}
	}

	return ops
}
