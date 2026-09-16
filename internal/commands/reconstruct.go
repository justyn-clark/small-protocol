package commands

import (
	"bytes"
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/justyn-clark/small-protocol/internal/sessionv2"
	"github.com/justyn-clark/small-protocol/internal/small"
	"github.com/justyn-clark/small-protocol/internal/workspace"
	"github.com/spf13/cobra"
)

type reconstructOutput struct {
	Workspace       string             `json:"workspace"`
	Task            *reconstructTask   `json:"task,omitempty"`
	Handoff         reconstructHandoff `json:"handoff"`
	MatchedEntries  int                `json:"matched_entries"`
	ReturnedEntries int                `json:"returned_entries"`
	Truncated       bool               `json:"truncated"`
	Entries         []reconstructEntry `json:"entries"`
}

type reconstructTask struct {
	ID           string   `json:"id"`
	Title        string   `json:"title"`
	Status       string   `json:"status,omitempty"`
	Steps        []string `json:"steps,omitempty"`
	Acceptance   []string `json:"acceptance,omitempty"`
	Dependencies []string `json:"dependencies,omitempty"`
}

type reconstructHandoff struct {
	Summary       string   `json:"summary,omitempty"`
	CurrentTaskID string   `json:"current_task_id,omitempty"`
	NextSteps     []string `json:"next_steps,omitempty"`
	ReplayID      string   `json:"replay_id,omitempty"`
}

type reconstructEntry struct {
	Index          int    `json:"index"`
	Timestamp      string `json:"timestamp"`
	TaskID         string `json:"task_id"`
	Status         string `json:"status"`
	Evidence       string `json:"evidence,omitempty"`
	Notes          string `json:"notes,omitempty"`
	CommandSummary string `json:"command_summary,omitempty"`
	CommandRef     string `json:"command_ref,omitempty"`
	CommandSha256  string `json:"command_sha256,omitempty"`
	ReplayID       string `json:"replay_id,omitempty"`
}

func reconstructCmd() *cobra.Command {
	var (
		taskID        string
		limit         int
		since         string
		jsonOutput    bool
		dir           string
		workspaceFlag string
		sessionID     string
		resume        bool
		maxBytes      int
	)

	cmd := &cobra.Command{
		Use:   "reconstruct",
		Short: "Reconstruct task or run evidence from SMALL state",
		Long: `Reconstructs a deterministic audit timeline from plan.small.yml,
progress.small.yml, and handoff.small.yml. This command reads state only; it
does not infer facts from chat history or terminal scrollback, and it is not a
replacement resume entrypoint for handoff.small.yml.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if dir == "" {
				dir = baseDir
			}
			artifactsDir := resolveArtifactsDir(dir)
			if sessionv2.IsWorkspace(artifactsDir) {
				output, err := buildV2ReconstructOutput(artifactsDir, sessionID, taskID, resume, limit, maxBytes)
				if err != nil {
					return err
				}
				if jsonOutput {
					return writeJSONValue(output)
				}
				fmt.Printf("Profile: %s  Mode: %s\nFrontier: %s\nEvents: %d of %d  Conflicts: %d  Truncated: %t\n", output.Profile, output.Mode, output.Frontier, output.ReturnedEvents, output.MatchedEvents, len(output.Conflicts), output.Truncated)
				for _, handoff := range output.Handoffs {
					fmt.Printf("Handoff %s: %s\n", handoff.EventID, stringVal(handoff.Payload["summary"]))
				}
				return nil
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

			output, err := buildReconstructOutput(artifactsDir, taskID, since, limit)
			if err != nil {
				return err
			}

			if jsonOutput {
				data, err := json.MarshalIndent(output, "", "  ")
				if err != nil {
					return err
				}
				fmt.Println(string(data))
				return nil
			}

			fmt.Print(formatReconstructText(output))
			return nil
		},
	}

	cmd.Flags().StringVar(&taskID, "task", "", "Only include progress for this task id")
	cmd.Flags().IntVar(&limit, "limit", 50, "Maximum number of progress entries to include (0 for all)")
	cmd.Flags().StringVar(&since, "since", "", "Only include entries at or after this RFC3339Nano timestamp")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output in JSON format")
	cmd.Flags().StringVar(&dir, "dir", ".", "Directory containing .small/ artifacts")
	cmd.Flags().StringVar(&workspaceFlag, "workspace", string(workspace.ScopeRoot), "Workspace scope (root or any)")
	cmd.Flags().StringVar(&sessionID, "session", "", "v2 session id")
	cmd.Flags().BoolVar(&resume, "resume", false, "Build a bounded v2 resume packet")
	cmd.Flags().IntVar(&maxBytes, "max-bytes", 65536, "Maximum encoded event bytes in a v2 resume packet")
	return cmd
}

type v2ReconstructOutput struct {
	Profile            string                           `json:"profile"`
	ProjectID          string                           `json:"project_id"`
	LineageID          string                           `json:"lineage_id"`
	Mode               string                           `json:"mode"`
	Frontier           string                           `json:"frontier"`
	SessionID          string                           `json:"session_id,omitempty"`
	Task               *sessionv2.TaskState             `json:"task,omitempty"`
	Obligations        []sessionv2.TaskState            `json:"obligations"`
	Handoffs           []sessionv2.Event                `json:"handoffs"`
	Conflicts          []sessionv2.Conflict             `json:"conflicts"`
	Evidence           []sessionv2.EvidenceVerification `json:"evidence"`
	Events             []sessionv2.Event                `json:"events"`
	MatchedEvents      int                              `json:"matched_events"`
	ReturnedEvents     int                              `json:"returned_events"`
	ReturnedEventBytes int                              `json:"returned_event_bytes"`
	Limit              int                              `json:"limit"`
	MaxBytes           int                              `json:"max_bytes"`
	Truncated          bool                             `json:"truncated"`
}

func buildV2ReconstructOutput(baseDir, explicitSession, taskRef string, resume bool, limit, maxBytes int) (v2ReconstructOutput, error) {
	store, err := sessionv2.Load(baseDir)
	if err != nil {
		return v2ReconstructOutput{}, err
	}
	state, err := sessionv2.Reduce(store)
	if err != nil {
		return v2ReconstructOutput{}, err
	}
	output := v2ReconstructOutput{Profile: sessionv2.ProfileVersion, ProjectID: store.Profile.ProjectID, LineageID: store.Profile.LineageID, Mode: state.Profile.Mode, Frontier: state.Frontier, Handoffs: state.Handoffs, Conflicts: state.Conflicts, Limit: limit, MaxBytes: maxBytes, Obligations: []sessionv2.TaskState{}, Events: []sessionv2.Event{}}
	if explicitSession != "" || resume {
		output.SessionID, err = sessionv2.ResolveSession(baseDir, explicitSession)
		if err != nil {
			return v2ReconstructOutput{}, err
		}
		if _, ok := store.Sessions[output.SessionID]; !ok {
			return v2ReconstructOutput{}, fmt.Errorf("session %s not found", output.SessionID)
		}
	}
	if strings.TrimSpace(taskRef) != "" {
		task, err := resolveV2Task(state, taskRef)
		if err != nil {
			return v2ReconstructOutput{}, err
		}
		output.Task = &task
	}
	for _, id := range sortedTaskIDs(state.Tasks) {
		task := state.Tasks[id]
		if task.Status != "completed" && task.Status != "cancelled" {
			output.Obligations = append(output.Obligations, task)
		}
	}
	output.Evidence, err = sessionv2.VerifyEvidence(baseDir)
	if err != nil {
		return v2ReconstructOutput{}, err
	}
	events := make([]sessionv2.Event, 0, len(store.Events))
	for _, event := range store.Events {
		if output.SessionID != "" && !resume && event.SessionID != output.SessionID {
			continue
		}
		if output.Task != nil && event.TaskID != "" && event.TaskID != output.Task.ID {
			continue
		}
		events = append(events, event)
	}
	sort.Slice(events, func(i, j int) bool {
		if events[i].SessionID == events[j].SessionID {
			return events[i].Sequence < events[j].Sequence
		}
		return events[i].SessionID < events[j].SessionID
	})
	output.MatchedEvents = len(events)
	if limit < 0 {
		return v2ReconstructOutput{}, fmt.Errorf("--limit must be non-negative")
	}
	if maxBytes < 0 {
		return v2ReconstructOutput{}, fmt.Errorf("--max-bytes must be non-negative")
	}
	start := 0
	if limit > 0 && len(events) > limit {
		start = len(events) - limit
		output.Truncated = true
	}
	events = events[start:]
	for i := len(events) - 1; i >= 0; i-- {
		data, _ := json.Marshal(events[i])
		if maxBytes > 0 && output.ReturnedEventBytes+len(data) > maxBytes {
			output.Truncated = true
			continue
		}
		output.Events = append(output.Events, events[i])
		output.ReturnedEventBytes += len(data)
	}
	for i, j := 0, len(output.Events)-1; i < j; i, j = i+1, j-1 {
		output.Events[i], output.Events[j] = output.Events[j], output.Events[i]
	}
	output.ReturnedEvents = len(output.Events)
	if output.ReturnedEvents < output.MatchedEvents {
		output.Truncated = true
	}
	return output, nil
}

func buildReconstructOutput(artifactsDir, taskID, since string, limit int) (reconstructOutput, error) {
	output := reconstructOutput{Workspace: artifactsDir}

	planPath := filepath.Join(artifactsDir, small.SmallDir, "plan.small.yml")
	if plan, err := loadPlan(planPath); err == nil && strings.TrimSpace(taskID) != "" {
		if task, _ := findTask(plan, strings.TrimSpace(taskID)); task != nil {
			output.Task = &reconstructTask{
				ID:           task.ID,
				Title:        task.Title,
				Status:       task.Status,
				Steps:        task.Steps,
				Acceptance:   task.Acceptance,
				Dependencies: task.Dependencies,
			}
		}
	}

	output.Handoff = loadReconstructHandoff(artifactsDir)

	entries, matched, err := loadReconstructEntries(artifactsDir, taskID, since, limit)
	if err != nil {
		return reconstructOutput{}, err
	}
	output.Entries = entries
	output.MatchedEntries = matched
	output.ReturnedEntries = len(entries)
	output.Truncated = matched > len(entries)
	return output, nil
}

// loadReconstructEntries returns the entries to display along with the total
// number that matched the filters before any --limit truncation was applied.
func loadReconstructEntries(artifactsDir, taskID, since string, limit int) ([]reconstructEntry, int, error) {
	progressPath := filepath.Join(artifactsDir, small.SmallDir, "progress.small.yml")
	progress, err := loadProgressData(progressPath)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to load progress.small.yml: %w", err)
	}

	var sinceTime time.Time
	if strings.TrimSpace(since) != "" {
		parsed, err := small.ParseProgressTimestamp(strings.TrimSpace(since))
		if err != nil {
			return nil, 0, fmt.Errorf("invalid --since timestamp: %w", err)
		}
		sinceTime = parsed
	}

	filterTaskID := strings.TrimSpace(taskID)
	entries := []reconstructEntry{}
	for idx, raw := range progress.Entries {
		entryTaskID := strings.TrimSpace(stringVal(raw["task_id"]))
		if filterTaskID != "" && entryTaskID != filterTaskID {
			continue
		}

		timestamp := strings.TrimSpace(stringVal(raw["timestamp"]))
		if !sinceTime.IsZero() {
			parsed, err := small.ParseProgressTimestamp(timestamp)
			if err != nil || parsed.Before(sinceTime) {
				continue
			}
		}

		commandSummary := strings.TrimSpace(stringVal(raw["command_summary"]))
		if commandSummary == "" {
			commandSummary = strings.TrimSpace(stringVal(raw["command"]))
		}

		entries = append(entries, reconstructEntry{
			Index:          idx,
			Timestamp:      timestamp,
			TaskID:         entryTaskID,
			Status:         strings.TrimSpace(stringVal(raw["status"])),
			Evidence:       strings.TrimSpace(stringVal(raw["evidence"])),
			Notes:          strings.TrimSpace(stringVal(raw["notes"])),
			CommandSummary: commandSummary,
			CommandRef:     strings.TrimSpace(stringVal(raw["command_ref"])),
			CommandSha256:  strings.TrimSpace(stringVal(raw["command_sha256"])),
			ReplayID:       strings.TrimSpace(stringVal(raw["replayId"])),
		})
	}

	matched := len(entries)
	if limit > 0 && len(entries) > limit {
		entries = entries[len(entries)-limit:]
	}
	return entries, matched, nil
}

func loadReconstructHandoff(artifactsDir string) reconstructHandoff {
	artifact, err := small.LoadArtifact(artifactsDir, "handoff.small.yml")
	if err != nil || artifact == nil || artifact.Data == nil {
		return reconstructHandoff{}
	}

	handoff := reconstructHandoff{Summary: strings.TrimSpace(stringVal(artifact.Data["summary"]))}
	if resume, ok := artifact.Data["resume"].(map[string]any); ok {
		handoff.CurrentTaskID = strings.TrimSpace(stringVal(resume["current_task_id"]))
		if rawSteps, ok := resume["next_steps"].([]any); ok {
			for _, raw := range rawSteps {
				step := strings.TrimSpace(stringVal(raw))
				if step != "" {
					handoff.NextSteps = append(handoff.NextSteps, step)
				}
			}
		}
	}
	if replay, ok := artifact.Data["replayId"].(map[string]any); ok {
		handoff.ReplayID = strings.TrimSpace(stringVal(replay["value"]))
	}
	return handoff
}

func formatReconstructText(output reconstructOutput) string {
	var buffer bytes.Buffer
	_, _ = fmt.Fprintf(&buffer, "Workspace: %s\n", output.Workspace)
	if output.Task != nil {
		_, _ = fmt.Fprintf(&buffer, "Task: %s - %s", output.Task.ID, output.Task.Title)
		if output.Task.Status != "" {
			_, _ = fmt.Fprintf(&buffer, " (%s)", output.Task.Status)
		}
		_, _ = fmt.Fprintln(&buffer)
	}
	if output.Handoff.Summary != "" {
		_, _ = fmt.Fprintf(&buffer, "Handoff: %s\n", output.Handoff.Summary)
	}
	if output.Handoff.ReplayID != "" {
		_, _ = fmt.Fprintf(&buffer, "ReplayId: %s\n", output.Handoff.ReplayID)
	}
	if len(output.Handoff.NextSteps) > 0 {
		_, _ = fmt.Fprintln(&buffer, "Next steps:")
		for _, step := range output.Handoff.NextSteps {
			_, _ = fmt.Fprintf(&buffer, "  - %s\n", step)
		}
	}

	if output.Truncated {
		_, _ = fmt.Fprintf(&buffer, "Progress entries: %d of %d (showing latest %d, oldest omitted by --limit)\n", len(output.Entries), output.MatchedEntries, len(output.Entries))
	} else {
		_, _ = fmt.Fprintf(&buffer, "Progress entries: %d\n", len(output.Entries))
	}
	for _, entry := range output.Entries {
		_, _ = fmt.Fprintf(&buffer, "  [%d] %s %s %s", entry.Index, entry.Timestamp, entry.TaskID, entry.Status)
		if entry.Evidence != "" {
			_, _ = fmt.Fprintf(&buffer, " - %s", entry.Evidence)
		}
		_, _ = fmt.Fprintln(&buffer)
		if entry.Notes != "" {
			_, _ = fmt.Fprintf(&buffer, "      notes: %s\n", entry.Notes)
		}
		if entry.CommandSummary != "" {
			_, _ = fmt.Fprintf(&buffer, "      command: %s\n", entry.CommandSummary)
		}
		if entry.CommandRef != "" {
			_, _ = fmt.Fprintf(&buffer, "      command_ref: %s\n", entry.CommandRef)
		}
	}

	return buffer.String()
}
