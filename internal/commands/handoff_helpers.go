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
	defaultNextStepsLimit = 3
	defaultHandoffOwner   = "agent"
	defaultReplayIdSource = "auto"
)

type handoffStatus string

const (
	handoffStatusComplete   handoffStatus = "complete"
	handoffStatusBlocked    handoffStatus = "blocked"
	handoffStatusInProgress handoffStatus = "in_progress"
)

type handoffState struct {
	Status        handoffStatus
	Summary       string
	CurrentTaskID *string
	NextSteps     []string
}

type existingHandoff struct {
	Summary  string
	Links    []linkOut
	ReplayId *replayIdOut
}

func buildHandoff(artifactsDir string, summary string, manualReplayId string, links []linkOut, replayId *replayIdOut, run *runOut, nextStepsLimit int) (handoffOut, error) {
	_ = summary
	if nextStepsLimit <= 0 {
		nextStepsLimit = defaultNextStepsLimit
	}

	planPath := filepath.Join(artifactsDir, small.SmallDir, "plan.small.yml")
	plan, err := loadPlan(planPath)
	if err != nil {
		return handoffOut{}, fmt.Errorf("failed to load plan.small.yml: %w", err)
	}

	state, err := computeHandoffState(artifactsDir, plan)
	if err != nil {
		return handoffOut{}, err
	}

	resume := resumeOut{
		CurrentTaskID: state.CurrentTaskID,
		NextSteps:     state.NextSteps,
	}

	if links == nil {
		links = []linkOut{}
	}

	if replayId == nil {
		if strings.TrimSpace(manualReplayId) != "" {
			smallDir := filepath.Join(artifactsDir, small.SmallDir)
			generatedReplayId, err := generateReplayId(smallDir, manualReplayId)
			if err != nil {
				return handoffOut{}, fmt.Errorf("replayId error: %w", err)
			}
			replayId = generatedReplayId
		} else {
			workspaceReplayID, err := currentWorkspaceRunReplayID(artifactsDir)
			if err != nil {
				return handoffOut{}, err
			}
			if workspaceReplayID != "" {
				replayId = &replayIdOut{Value: workspaceReplayID, Source: defaultReplayIdSource}
			} else {
				smallDir := filepath.Join(artifactsDir, small.SmallDir)
				generatedReplayId, err := generateReplayId(smallDir, "")
				if err != nil {
					return handoffOut{}, fmt.Errorf("replayId error: %w", err)
				}
				replayId = generatedReplayId
			}
		}
	}

	return handoffOut{
		SmallVersion: small.ProtocolVersion,
		Owner:        defaultHandoffOwner,
		Summary:      state.Summary,
		Resume:       resume,
		Links:        links,
		ReplayId:     *replayId,
		Run:          run,
	}, nil
}

func writeHandoff(artifactsDir string, handoff handoffOut) error {
	smallDir := filepath.Join(artifactsDir, small.SmallDir)
	if err := os.MkdirAll(smallDir, 0o755); err != nil {
		return fmt.Errorf("failed to create .small directory: %w", err)
	}

	yml, err := small.MarshalYAMLWithQuotedVersion(handoff)
	if err != nil {
		return fmt.Errorf("failed to marshal handoff: %w", err)
	}

	outPath := filepath.Join(smallDir, "handoff.small.yml")
	if err := os.WriteFile(outPath, yml, 0o644); err != nil {
		return fmt.Errorf("failed to write %s: %w", outPath, err)
	}
	if err := touchWorkspaceUpdatedAt(artifactsDir); err != nil {
		return err
	}
	return nil
}

func computeHandoffState(artifactsDir string, plan *PlanData) (handoffState, error) {
	strictOK, strictSummary, err := strictReadyForComplete(artifactsDir)
	if err != nil {
		return handoffState{}, err
	}

	explicitBlockedTaskID := firstTaskIDByStatus(plan, "blocked")
	preferredTaskID := explicitBlockedTaskID
	if preferredTaskID == "" {
		preferredTaskID = nextIncompleteTaskID(plan)
	}

	if !strictOK || explicitBlockedTaskID != "" {
		state := handoffState{Status: handoffStatusBlocked}
		currentTaskID := preferredTaskID
		if currentTaskID == "" {
			currentTaskID = "meta/blocker"
		}
		state.CurrentTaskID = stringPtr(currentTaskID)

		var nextSteps []string
		if currentTaskID != "meta/blocker" {
			nextSteps = append(nextSteps, fmt.Sprintf("Unblock %s", currentTaskID))
		}
		if !strictOK {
			if strictSummary == "" {
				nextSteps = append(nextSteps, "Resolve strict check failures")
			} else {
				nextSteps = append(nextSteps, fmt.Sprintf("Resolve strict check failures: %s", strictSummary))
			}
			nextSteps = append(nextSteps, "Run: small check --strict")
		}
		if len(nextSteps) == 0 {
			nextSteps = append(nextSteps, "Add plan tasks via small plan --add")
		}
		state.NextSteps = nextSteps

		if currentTaskID != "meta/blocker" {
			state.Summary = fmt.Sprintf("Run blocked on %s.", currentTaskID)
		} else {
			state.Summary = "Run blocked. strict check failed."
		}
		return state, nil
	}

	if allPlanTasksCompleted(plan) {
		return handoffState{
			Status:    handoffStatusComplete,
			Summary:   "Run complete. strict check passed. All plan tasks completed.",
			NextSteps: []string{},
		}, nil
	}

	nextTaskID := firstTaskIDByStatus(plan, "in_progress")
	if nextTaskID == "" {
		nextTaskID = nextIncompleteTaskID(plan)
	}

	state := handoffState{
		Status:  handoffStatusInProgress,
		Summary: fmt.Sprintf("Run in progress. Next task: %s.", nextTaskID),
	}
	if strings.TrimSpace(nextTaskID) != "" {
		state.CurrentTaskID = stringPtr(nextTaskID)
		state.NextSteps = []string{fmt.Sprintf("Continue with %s", nextTaskID)}
	} else {
		state.NextSteps = []string{}
	}
	return state, nil
}

func strictReadyForComplete(artifactsDir string) (bool, string, error) {
	filenames := []string{
		"intent.small.yml",
		"constraints.small.yml",
		"plan.small.yml",
		"progress.small.yml",
	}
	if small.ArtifactExists(artifactsDir, "handoff.small.yml") {
		filenames = append(filenames, "handoff.small.yml")
	}

	artifacts := map[string]*small.Artifact{}
	for _, filename := range filenames {
		artifact, err := small.LoadArtifact(artifactsDir, filename)
		if err != nil {
			return false, "", fmt.Errorf("failed to load %s for handoff status: %w", filename, err)
		}
		artifacts[artifact.Type] = artifact
	}

	schemaErrs := small.ValidateAllArtifactsWithConfig(artifacts, small.SchemaConfig{BaseDir: artifactsDir})
	if len(schemaErrs) > 0 {
		return false, strings.TrimSpace(schemaErrs[0].Error()), nil
	}

	violations := small.CheckInvariants(artifacts, true)
	if len(violations) > 0 {
		return false, strings.TrimSpace(violations[0].Message), nil
	}

	layoutViolations, err := small.StrictSmallLayoutViolations(artifactsDir, "")
	if err != nil {
		return false, "", err
	}
	if len(layoutViolations) > 0 {
		return false, strings.TrimSpace(layoutViolations[0].Message), nil
	}

	return true, "", nil
}

func firstTaskIDByStatus(plan *PlanData, status string) string {
	if plan == nil {
		return ""
	}
	target := strings.TrimSpace(status)
	for _, task := range plan.Tasks {
		if normalizePlanStatus(task.Status) == target {
			return strings.TrimSpace(task.ID)
		}
	}
	return ""
}

func nextIncompleteTaskID(plan *PlanData) string {
	if plan == nil {
		return ""
	}
	for _, task := range plan.Tasks {
		status := normalizePlanStatus(task.Status)
		if status == "completed" {
			continue
		}
		return strings.TrimSpace(task.ID)
	}
	return ""
}

func allPlanTasksCompleted(plan *PlanData) bool {
	if plan == nil || len(plan.Tasks) == 0 {
		return false
	}
	for _, task := range plan.Tasks {
		if normalizePlanStatus(task.Status) != "completed" {
			return false
		}
	}
	return true
}

func stringPtr(value string) *string {
	v := strings.TrimSpace(value)
	if v == "" {
		return nil
	}
	return &v
}

func normalizePlanStatus(status string) string {
	if strings.TrimSpace(status) == "" {
		return "pending"
	}
	return strings.TrimSpace(status)
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
