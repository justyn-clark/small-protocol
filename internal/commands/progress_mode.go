package commands

import (
	"os"
	"regexp"
	"strings"
)

const progressModeEnvVar = "SMALL_PROGRESS_MODE"

type progressMode string

const (
	progressModeSignal progressMode = "signal"
	progressModeAudit  progressMode = "audit"
)

type progressEventKind string

const (
	progressEventApplyDryRun   progressEventKind = "apply_dry_run"
	progressEventApplyStart    progressEventKind = "apply_start"
	progressEventApplyComplete progressEventKind = "apply_complete"
)

var planTaskIDPattern = regexp.MustCompile("^task-[0-9]+$")

func resolveProgressMode() progressMode {
	value := strings.ToLower(strings.TrimSpace(os.Getenv(progressModeEnvVar)))
	switch progressMode(value) {
	case progressModeAudit:
		return progressModeAudit
	case progressModeSignal:
		return progressModeSignal
	default:
		return progressModeSignal
	}
}

func shouldEmitProgress(kind progressEventKind, taskID string, mode progressMode) bool {
	if mode == progressModeAudit {
		return true
	}

	switch kind {
	case progressEventApplyStart:
		return false
	case progressEventApplyDryRun, progressEventApplyComplete:
		return isPlanTaskID(taskID)
	default:
		return true
	}
}

func isPlanTaskID(taskID string) bool {
	return planTaskIDPattern.MatchString(strings.TrimSpace(taskID))
}

func isSignalProgressEntry(entry ProgressEntry) bool {
	taskID := strings.TrimSpace(entry.TaskID)
	if taskID == "" {
		return false
	}

	notes := strings.ToLower(strings.TrimSpace(entry.Notes))
	if notes == "apply: execution started" {
		return false
	}

	if taskID == "apply" || strings.HasPrefix(taskID, "meta/apply") {
		return false
	}

	return true
}
