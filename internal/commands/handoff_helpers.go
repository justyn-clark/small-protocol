package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/justyn-clark/small-protocol/internal/small"
	"gopkg.in/yaml.v3"
)

const (
	defaultHandoffSummary  = "Handoff generated from current plan state"
	defaultNextStepMessage = "No actionable tasks. Add plan tasks via small plan --add."
	defaultNextStepsLimit  = 3
	defaultHandoffOwner    = "agent"
	defaultReplayIdSource  = "auto"
)

type existingHandoff struct {
	Summary  string
	Links    []linkOut
	ReplayId *replayIdOut
}

func buildHandoff(artifactsDir string, summary string, manualReplayId string, links []linkOut, replayId *replayIdOut, run *runOut, nextStepsLimit int) (handoffOut, error) {
	if summary == "" {
		summary = defaultHandoffSummary
	}
	if nextStepsLimit <= 0 {
		nextStepsLimit = defaultNextStepsLimit
	}

	planPath := filepath.Join(artifactsDir, small.SmallDir, "plan.small.yml")
	plan, err := loadPlan(planPath)
	if err != nil {
		return handoffOut{}, fmt.Errorf("failed to load plan.small.yml: %w", err)
	}

	resume := computeHandoffResume(plan, nextStepsLimit)

	if links == nil {
		links = []linkOut{}
	}

	if replayId == nil {
		smallDir := filepath.Join(artifactsDir, small.SmallDir)
		generatedReplayId, err := generateReplayId(smallDir, manualReplayId)
		if err != nil {
			return handoffOut{}, fmt.Errorf("replayId error: %w", err)
		}
		replayId = generatedReplayId
	}

	return handoffOut{
		SmallVersion: small.ProtocolVersion,
		Owner:        defaultHandoffOwner,
		Summary:      summary,
		Resume:       resume,
		Links:        links,
		ReplayId:     *replayId,
		Run:          run,
	}, nil
}

func writeHandoff(artifactsDir string, handoff handoffOut) error {
	smallDir := filepath.Join(artifactsDir, small.SmallDir)
	if err := os.MkdirAll(smallDir, 0755); err != nil {
		return fmt.Errorf("failed to create .small directory: %w", err)
	}

	yml, err := small.MarshalYAMLWithQuotedVersion(handoff)
	if err != nil {
		return fmt.Errorf("failed to marshal handoff: %w", err)
	}

	outPath := filepath.Join(smallDir, "handoff.small.yml")
	if err := os.WriteFile(outPath, yml, 0644); err != nil {
		return fmt.Errorf("failed to write %s: %w", outPath, err)
	}
	return nil
}

func computeHandoffResume(plan *PlanData, maxSteps int) resumeOut {
	if maxSteps <= 0 {
		maxSteps = defaultNextStepsLimit
	}

	taskStatuses := make(map[string]string)
	taskByID := make(map[string]PlanTask)
	currentTaskID := ""

	for _, task := range plan.Tasks {
		status := normalizePlanStatus(task.Status)
		taskStatuses[task.ID] = status
		taskByID[task.ID] = task
		if status == "in_progress" && currentTaskID == "" {
			currentTaskID = task.ID
		}
	}

	nextSteps := make([]string, 0, maxSteps)
	used := make(map[string]bool)

	if currentTaskID != "" {
		if task, ok := taskByID[currentTaskID]; ok {
			label := planTaskLabel(task)
			if label != "" {
				nextSteps = append(nextSteps, label)
				used[label] = true
			}
		}
	}

	for _, task := range plan.Tasks {
		if len(nextSteps) >= maxSteps {
			break
		}

		status := normalizePlanStatus(task.Status)
		if status != "pending" {
			continue
		}

		depsSatisfied := true
		for _, depID := range task.Dependencies {
			depStatus, exists := taskStatuses[depID]
			if !exists || depStatus != "completed" {
				depsSatisfied = false
				break
			}
		}
		if !depsSatisfied {
			continue
		}

		label := planTaskLabel(task)
		if label == "" || used[label] {
			continue
		}
		nextSteps = append(nextSteps, label)
		used[label] = true
	}

	if len(nextSteps) == 0 {
		nextSteps = []string{defaultNextStepMessage}
	}

	return resumeOut{
		CurrentTaskID: currentTaskID,
		NextSteps:     nextSteps,
	}
}

func normalizePlanStatus(status string) string {
	if strings.TrimSpace(status) == "" {
		return "pending"
	}
	return status
}

func planTaskLabel(task PlanTask) string {
	if strings.TrimSpace(task.Title) != "" {
		return task.Title
	}
	return strings.TrimSpace(task.ID)
}

func loadExistingHandoff(artifactsDir string) (*existingHandoff, error) {
	path := filepath.Join(artifactsDir, small.SmallDir, "handoff.small.yml")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var payload map[string]any
	if err := yaml.Unmarshal(data, &payload); err != nil {
		return nil, fmt.Errorf("failed to parse handoff.small.yml: %w", err)
	}

	return &existingHandoff{
		Summary:  stringVal(payload["summary"]),
		Links:    parseLinks(payload["links"]),
		ReplayId: parseReplayId(payload["replayId"]),
	}, nil
}

func parseLinks(raw any) []linkOut {
	items, ok := raw.([]any)
	if !ok {
		return nil
	}
	links := make([]linkOut, 0, len(items))
	for _, item := range items {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		link := linkOut{
			URL:         stringVal(m["url"]),
			Description: stringVal(m["description"]),
		}
		if link.URL == "" && link.Description == "" {
			continue
		}
		links = append(links, link)
	}
	return links
}

func parseReplayId(raw any) *replayIdOut {
	m, ok := raw.(map[string]any)
	if !ok {
		return nil
	}
	value := strings.TrimSpace(stringVal(m["value"]))
	if !replayIdPattern.MatchString(value) {
		return nil
	}
	source := strings.TrimSpace(stringVal(m["source"]))
	if source == "" {
		source = defaultReplayIdSource
	}
	return &replayIdOut{
		Value:  strings.ToLower(value),
		Source: source,
	}
}
