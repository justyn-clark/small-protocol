package fixers

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"

	"github.com/justyn-clark/small-protocol/internal/small"
)

type OrphanProgressRewrite struct {
	Index          int
	OriginalTaskID string
	NewTaskID      string
	Category       string
	Hash           string
}

type OrphanProgressCounts struct {
	Operational int
	Historical  int
	Unknown     int
}

type OrphanProgressFixResult struct {
	ReplayID string
	Rewrites []OrphanProgressRewrite
	Counts   OrphanProgressCounts
}

var historicalTaskPattern = regexp.MustCompile(`^task-\d+$`)

func FixOrphanProgress(baseDir string) (OrphanProgressFixResult, error) {
	plan, err := small.LoadArtifact(baseDir, "plan.small.yml")
	if err != nil {
		return OrphanProgressFixResult{}, err
	}
	progress, err := small.LoadArtifact(baseDir, "progress.small.yml")
	if err != nil {
		return OrphanProgressFixResult{}, err
	}
	handoff, err := small.LoadArtifact(baseDir, "handoff.small.yml")
	if err != nil {
		return OrphanProgressFixResult{}, err
	}

	planTaskIDs := extractPlanTaskIDs(plan)
	entries, ok := progress.Data["entries"].([]any)
	if !ok {
		return OrphanProgressFixResult{}, fmt.Errorf("progress.entries must be an array")
	}

	replayID := extractReplayID(handoff)
	result := OrphanProgressFixResult{ReplayID: replayID}

	for i, entry := range entries {
		entryMap, ok := entry.(map[string]any)
		if !ok {
			continue
		}

		taskID := strings.TrimSpace(stringVal(entryMap["task_id"]))
		if taskID == "" {
			continue
		}
		if strings.HasPrefix(taskID, "meta/") {
			continue
		}
		if !isReplayIDInScope(entryMap, replayID) {
			continue
		}
		if _, ok := planTaskIDs[taskID]; ok {
			continue
		}

		category, newTaskID := classifyOrphanTaskID(taskID)
		entryMap["task_id"] = newTaskID

		result.Rewrites = append(result.Rewrites, OrphanProgressRewrite{
			Index:          i,
			OriginalTaskID: taskID,
			NewTaskID:      newTaskID,
			Category:       category,
			Hash:           hashProgressEntry(taskID, entryMap),
		})

		switch category {
		case "operational":
			result.Counts.Operational++
		case "historical":
			result.Counts.Historical++
		case "unknown":
			result.Counts.Unknown++
		}
	}

	if len(result.Rewrites) == 0 {
		return result, nil
	}

	if err := small.SaveArtifact(baseDir, "progress.small.yml", progress.Data); err != nil {
		return OrphanProgressFixResult{}, err
	}

	return result, nil
}

func extractPlanTaskIDs(plan *small.Artifact) map[string]struct{} {
	ids := map[string]struct{}{}
	if plan == nil || plan.Data == nil {
		return ids
	}
	tasks, ok := plan.Data["tasks"].([]any)
	if !ok {
		return ids
	}
	for _, raw := range tasks {
		taskMap, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		id := strings.TrimSpace(stringVal(taskMap["id"]))
		if id == "" {
			continue
		}
		ids[id] = struct{}{}
	}
	return ids
}

func extractReplayID(handoff *small.Artifact) string {
	if handoff == nil || handoff.Data == nil {
		return ""
	}
	metadata, ok := handoff.Data["replayId"].(map[string]any)
	if !ok {
		return ""
	}
	return strings.TrimSpace(stringVal(metadata["value"]))
}

func isReplayIDInScope(entry map[string]any, replayID string) bool {
	if replayID == "" {
		return true
	}
	entryReplayID := strings.TrimSpace(stringVal(entry["replayId"]))
	if entryReplayID == "" {
		return false
	}
	return entryReplayID == replayID
}

func classifyOrphanTaskID(taskID string) (string, string) {
	switch taskID {
	case "reset", "init", "apply":
		return "operational", "meta/" + taskID
	default:
	}

	if historicalTaskPattern.MatchString(taskID) {
		return "historical", "meta/historical/" + taskID
	}

	safe := sanitizeTaskID(taskID)
	return "unknown", "meta/historical/unknown/" + safe
}

func sanitizeTaskID(taskID string) string {
	value := strings.TrimSpace(taskID)
	value = strings.ReplaceAll(value, "/", "-")
	value = strings.ReplaceAll(value, " ", "-")
	if value == "" {
		return "unknown"
	}
	return value
}

func hashProgressEntry(taskID string, entry map[string]any) string {
	parts := []string{
		taskID,
		strings.TrimSpace(stringVal(entry["timestamp"])),
		strings.TrimSpace(stringVal(entry["status"])),
	}
	sum := sha256.Sum256([]byte(strings.Join(parts, "|")))
	return hex.EncodeToString(sum[:])
}

func stringVal(value any) string {
	if value == nil {
		return ""
	}
	if s, ok := value.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", value)
}
