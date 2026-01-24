package commands

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/justyn-clark/small-protocol/internal/small"
)

type strictS2Report struct {
	Scope       string
	Operational []string
	Historical  []string
	Unknown     []string
}

const strictS2ListCap = defaultListCap

var strictS2Pattern = regexp.MustCompile(`strict invariant S2 failed \(replayId scope: ([^)]+)\): unknown progress task ids: (.+)$`)
var strictS2HistoricalPattern = regexp.MustCompile(`^task-\d+$`)

func buildStrictS2ReportFromViolations(violations []small.InvariantViolation) (*strictS2Report, []string) {
	var s2Messages []string
	var other []string
	for _, v := range violations {
		if isStrictS2Message(v.Message) {
			s2Messages = append(s2Messages, v.Message)
			continue
		}
		other = append(other, fmt.Sprintf("%s: %s", v.File, v.Message))
	}
	report, ok := buildStrictS2Report(s2Messages)
	if !ok {
		return nil, other
	}
	return &report, other
}

func buildStrictS2ReportFromVerifyErrors(errors []verifyError) (*strictS2Report, []verifyError) {
	var s2Messages []string
	var other []verifyError
	for _, ve := range errors {
		if isStrictS2Message(ve.message) {
			s2Messages = append(s2Messages, ve.message)
			continue
		}
		other = append(other, ve)
	}
	report, ok := buildStrictS2Report(s2Messages)
	if !ok {
		return nil, errors
	}
	return &report, other
}

func isStrictS2Message(message string) bool {
	return strings.Contains(message, "strict invariant S2 failed")
}

func buildStrictS2Report(messages []string) (strictS2Report, bool) {
	report := strictS2Report{}
	if len(messages) == 0 {
		return report, false
	}

	operational := map[string]struct{}{}
	historical := map[string]struct{}{}
	unknown := map[string]struct{}{}
	scope := ""

	for _, message := range messages {
		match := strictS2Pattern.FindStringSubmatch(message)
		if len(match) != 3 {
			continue
		}
		if scope == "" {
			scope = match[1]
		}
		items := parseStrictS2TaskList(match[2])
		for _, item := range items {
			switch classifyStrictS2Task(item) {
			case "operational":
				operational[item] = struct{}{}
			case "historical":
				historical[item] = struct{}{}
			default:
				unknown[item] = struct{}{}
			}
		}
	}

	report.Scope = scope
	report.Operational = sortedKeys(operational)
	report.Historical = sortedKeys(historical)
	report.Unknown = sortedKeys(unknown)

	return report, true
}

func strictS2ReportLines(report strictS2Report) []string {
	return []string{
		fmt.Sprintf("ReplayId scope: %s", scopeLabel(report.Scope)),
		"Why: progress entries reference task_ids not present in the current plan.",
		"",
		"Unknown task_ids:",
		fmt.Sprintf("  %s", formatCountedList("operational", report.Operational, strictS2ListCap)),
		fmt.Sprintf("  %s", formatCountedList("historical", report.Historical, strictS2ListCap)),
		fmt.Sprintf("  %s", formatCountedList("unknown", report.Unknown, strictS2ListCap)),
		"",
		"Fix:",
		"small fix --orphan-progress",
		"small check --strict",
	}
}

func parseStrictS2TaskList(raw string) []string {
	parts := strings.Split(raw, ";")
	items := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}
		if idx := strings.Index(trimmed, " (closest:"); idx != -1 {
			trimmed = strings.TrimSpace(trimmed[:idx])
		}
		if trimmed == "" {
			continue
		}
		items = append(items, trimmed)
	}
	return items
}

func classifyStrictS2Task(taskID string) string {
	switch taskID {
	case "reset", "init", "apply":
		return "operational"
	default:
	}
	if strictS2HistoricalPattern.MatchString(taskID) {
		return "historical"
	}
	return "unknown"
}

func sortedKeys(values map[string]struct{}) []string {
	items := make([]string, 0, len(values))
	for key := range values {
		items = append(items, key)
	}
	sort.Strings(items)
	return items
}

func scopeLabel(scope string) string {
	scope = strings.TrimSpace(scope)
	if scope == "" {
		return "unknown"
	}
	return scope
}
