package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/justyn-clark/small-protocol/internal/small"
	"github.com/justyn-clark/small-protocol/internal/version"
	"github.com/justyn-clark/small-protocol/internal/workspace"
	"github.com/spf13/cobra"
)

type emitOutput struct {
	CliVersion  string                 `json:"cliVersion"`
	SpecVersion []string               `json:"specVersion"`
	Workspace   string                 `json:"workspace"`
	Paths       emitPaths              `json:"paths"`
	Status      emitStatusSummary      `json:"status,omitempty"`
	Progress    emitProgressSummary    `json:"progress,omitempty"`
	Intent      emitIntentSummary      `json:"intent,omitempty"`
	Constraints emitConstraintsSummary `json:"constraints,omitempty"`
	Plan        *emitPlanDetails       `json:"plan,omitempty"`
	Enforcement *emitEnforcement       `json:"enforcement,omitempty"`
}

type emitPaths struct {
	Root      string            `json:"root"`
	Workspace string            `json:"workspace"`
	Artifacts emitArtifactPaths `json:"artifacts"`
}

type emitArtifactPaths struct {
	Intent      string `json:"intent"`
	Constraints string `json:"constraints"`
	Plan        string `json:"plan"`
	Progress    string `json:"progress"`
	Handoff     string `json:"handoff"`
}

type emitStatusSummary struct {
	TotalTasks     int            `json:"totalTasks"`
	TasksByStatus  map[string]int `json:"tasksByStatus"`
	NextActionable []string       `json:"nextActionable"`
}

type emitProgressSummary struct {
	LastTimestamp string                   `json:"lastTimestamp,omitempty"`
	Recent        []ProgressEntry          `json:"recent"`
	Entries       []map[string]interface{} `json:"entries,omitempty"`
}

type emitIntentSummary struct {
	SmallVersion         string                 `json:"smallVersion,omitempty"`
	Owner                string                 `json:"owner,omitempty"`
	Intent               string                 `json:"intent,omitempty"`
	Scope                emitIntentScope        `json:"scope,omitempty"`
	SuccessCriteriaCount int                    `json:"successCriteriaCount,omitempty"`
	Artifact             map[string]interface{} `json:"artifact,omitempty"`
}

type emitIntentScope struct {
	IncludeCount int `json:"includeCount,omitempty"`
	ExcludeCount int `json:"excludeCount,omitempty"`
}

type emitConstraintsSummary struct {
	Present       bool                   `json:"present"`
	ConstraintIds []string               `json:"constraintIds,omitempty"`
	Artifact      map[string]interface{} `json:"artifact,omitempty"`
}

type emitPlanDetails struct {
	SmallVersion string     `json:"smallVersion"`
	Owner        string     `json:"owner"`
	Tasks        []PlanTask `json:"tasks"`
}

type emitEnforcement struct {
	Validate checkStageResult `json:"validate"`
	Lint     checkStageResult `json:"lint"`
	Verify   checkStageResult `json:"verify"`
	ExitCode int              `json:"exitCode"`
	Failures []string         `json:"failures,omitempty"`
}

type emitInclude map[string]bool

func (e emitInclude) Has(key string) bool {
	if e == nil {
		return false
	}
	return e[key]
}

func emitCmd() *cobra.Command {
	var dir string
	var workspaceFlag string
	var recent int
	var tasks int
	var include string
	var runCheckFlag bool

	cmd := &cobra.Command{
		Use:   "emit",
		Short: "Emit structured SMALL state in JSON",
		Long: `Emits structured JSON describing current SMALL state.

Emit is read-only unless --check is used, which runs small check and
includes enforcement results in the JSON output. The output is JSON only.`,
		Run: func(cmd *cobra.Command, args []string) {
			if dir == "" {
				dir = baseDir
			}
			artifactsDir := resolveArtifactsDir(dir)
			smallDir := filepath.Join(artifactsDir, small.SmallDir)

			if _, err := os.Stat(smallDir); os.IsNotExist(err) {
				fmt.Fprintln(os.Stderr, "Error: .small/ directory does not exist")
				os.Exit(ExitSystemError)
			}

			scope, err := workspace.ParseScope(workspaceFlag)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Invalid workspace scope: %v\n", err)
				os.Exit(ExitSystemError)
			}
			if scope != workspace.ScopeAny {
				if err := enforceWorkspaceScope(artifactsDir, scope); err != nil {
					fmt.Fprintf(os.Stderr, "Workspace validation failed: %v\n", err)
					os.Exit(ExitInvalid)
				}
			}

			includeSet, err := parseEmitInclude(include)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Invalid include value: %v\n", err)
				os.Exit(ExitSystemError)
			}

			rootPath, err := filepath.Abs(dir)
			if err != nil {
				rootPath = dir
			}

			output, exitCode, err := buildEmitOutput(rootPath, artifactsDir, includeSet, scope, recent, tasks, runCheckFlag)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(ExitSystemError)
			}

			data, err := json.MarshalIndent(output, "", "  ")
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(ExitSystemError)
			}
			fmt.Println(string(data))

			if runCheckFlag {
				os.Exit(exitCode)
			}
		},
	}

	cmd.Flags().StringVar(&dir, "dir", ".", "Directory containing .small/ artifacts")
	cmd.Flags().StringVar(&workspaceFlag, "workspace", string(workspace.ScopeRoot), "Workspace scope (root, examples, or any)")
	cmd.Flags().IntVar(&recent, "recent", 5, "Number of recent progress entries to include")
	cmd.Flags().IntVar(&tasks, "tasks", 3, "Number of next actionable tasks to include")
	cmd.Flags().StringVar(&include, "include", "", "Comma-separated sections to include (status, intent, constraints, plan, progress, paths, enforcement)")
	cmd.Flags().BoolVar(&runCheckFlag, "check", false, "Run small check and include enforcement results")

	return cmd
}

func parseEmitInclude(value string) (emitInclude, error) {
	includeSet := emitInclude{}
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return includeSet, nil
	}

	allowed := map[string]bool{
		"status":      true,
		"intent":      true,
		"constraints": true,
		"plan":        true,
		"progress":    true,
		"paths":       true,
		"enforcement": true,
	}

	parts := strings.Split(trimmed, ",")
	for _, part := range parts {
		p := strings.ToLower(strings.TrimSpace(part))
		if p == "" {
			continue
		}
		if !allowed[p] {
			return nil, fmt.Errorf("unknown include section %q", p)
		}
		includeSet[p] = true
	}

	return includeSet, nil
}

func buildEmitOutput(rootDir, artifactsDir string, include emitInclude, scope workspace.Scope, recent, tasks int, runCheckFlag bool) (emitOutput, int, error) {
	workspaceInfo, err := workspace.Load(artifactsDir)
	if err != nil {
		return emitOutput{}, ExitSystemError, err
	}

	output := emitOutput{
		CliVersion:  version.GetVersion(),
		SpecVersion: []string{small.ProtocolVersion},
		Workspace:   string(workspaceInfo.Kind),
	}

	pathsPayload := emitPaths{
		Root:      rootDir,
		Workspace: artifactsDir,
		Artifacts: emitArtifactPaths{
			Intent:      filepath.Join(artifactsDir, small.SmallDir, "intent.small.yml"),
			Constraints: filepath.Join(artifactsDir, small.SmallDir, "constraints.small.yml"),
			Plan:        filepath.Join(artifactsDir, small.SmallDir, "plan.small.yml"),
			Progress:    filepath.Join(artifactsDir, small.SmallDir, "progress.small.yml"),
			Handoff:     filepath.Join(artifactsDir, small.SmallDir, "handoff.small.yml"),
		},
	}

	if include.Has("paths") || len(include) == 0 {
		output.Paths = pathsPayload
	}

	if include.Has("status") || len(include) == 0 {
		output.Status = emitStatusSummary{
			TasksByStatus:  map[string]int{},
			NextActionable: []string{},
		}
		if small.ArtifactExists(artifactsDir, "plan.small.yml") {
			planStatus, err := analyzePlan(artifactsDir, tasks)
			if err == nil {
				output.Status.TotalTasks = planStatus.TotalTasks
				output.Status.TasksByStatus = planStatus.TasksByStatus
				output.Status.NextActionable = planStatus.NextActionable
			}
		}
	}

	if include.Has("plan") {
		if small.ArtifactExists(artifactsDir, "plan.small.yml") {
			planData, err := loadPlan(filepath.Join(artifactsDir, small.SmallDir, "plan.small.yml"))
			if err != nil {
				return emitOutput{}, ExitSystemError, err
			}
			output.Plan = &emitPlanDetails{
				SmallVersion: planData.SmallVersion,
				Owner:        planData.Owner,
				Tasks:        planData.Tasks,
			}
		}
	}

	if include.Has("progress") || len(include) == 0 {
		output.Progress = emitProgressSummary{Recent: []ProgressEntry{}}
		if small.ArtifactExists(artifactsDir, "progress.small.yml") {
			recentEntries, err := getRecentProgress(artifactsDir, recent)
			if err != nil {
				return emitOutput{}, ExitSystemError, err
			}
			output.Progress.Recent = recentEntries

			progressData, err := loadProgressData(filepath.Join(artifactsDir, small.SmallDir, "progress.small.yml"))
			if err != nil {
				return emitOutput{}, ExitSystemError, err
			}
			lastTimestamp, err := lastProgressTimestamp(progressData.Entries)
			if err != nil {
				return emitOutput{}, ExitSystemError, err
			}
			if !lastTimestamp.IsZero() {
				output.Progress.LastTimestamp = formatProgressTimestamp(lastTimestamp)
			}
			if include.Has("progress") {
				output.Progress.Entries = progressData.Entries
			}
		}
	}

	if include.Has("intent") || len(include) == 0 {
		if small.ArtifactExists(artifactsDir, "intent.small.yml") {
			intentArtifact, err := small.LoadArtifact(artifactsDir, "intent.small.yml")
			if err != nil {
				return emitOutput{}, ExitSystemError, err
			}
			output.Intent = buildIntentSummary(intentArtifact)
			if include.Has("intent") {
				output.Intent.Artifact = intentArtifact.Data
			}
		}
	}

	if include.Has("constraints") || len(include) == 0 {
		if !small.ArtifactExists(artifactsDir, "constraints.small.yml") {
			output.Constraints = emitConstraintsSummary{Present: false}
		} else {
			constraintsArtifact, err := small.LoadArtifact(artifactsDir, "constraints.small.yml")
			if err != nil {
				return emitOutput{}, ExitSystemError, err
			}
			output.Constraints = buildConstraintsSummary(constraintsArtifact)
			if include.Has("constraints") {
				output.Constraints.Artifact = constraintsArtifact.Data
			}
		}
	}

	if runCheckFlag {
		checkCode, checkOutput, err := runCheck(artifactsDir, false, true, true, scope)
		if err != nil {
			return emitOutput{}, ExitSystemError, err
		}
		output.Enforcement = buildEnforcementSummary(checkOutput)
		return output, checkCode, nil
	}

	return output, ExitValid, nil
}

func buildIntentSummary(artifact *small.Artifact) emitIntentSummary {
	summary := emitIntentSummary{}
	if artifact == nil || artifact.Data == nil {
		return summary
	}

	summary.SmallVersion = stringVal(artifact.Data["small_version"])
	summary.Owner = stringVal(artifact.Data["owner"])
	summary.Intent = stringVal(artifact.Data["intent"])

	if scope, ok := artifact.Data["scope"].(map[string]interface{}); ok {
		if include, ok := scope["include"].([]interface{}); ok {
			summary.Scope.IncludeCount = len(include)
		}
		if exclude, ok := scope["exclude"].([]interface{}); ok {
			summary.Scope.ExcludeCount = len(exclude)
		}
	}
	if criteria, ok := artifact.Data["success_criteria"].([]interface{}); ok {
		summary.SuccessCriteriaCount = len(criteria)
	}

	return summary
}

func buildConstraintsSummary(artifact *small.Artifact) emitConstraintsSummary {
	summary := emitConstraintsSummary{}
	if artifact == nil || artifact.Data == nil {
		return summary
	}

	summary.Present = true
	constraints, ok := artifact.Data["constraints"].([]interface{})
	if !ok {
		return summary
	}
	for _, item := range constraints {
		m, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		id := strings.TrimSpace(stringVal(m["id"]))
		if id != "" {
			summary.ConstraintIds = append(summary.ConstraintIds, id)
		}
	}
	if len(summary.ConstraintIds) > 0 {
		sort.Strings(summary.ConstraintIds)
	}

	return summary
}

func buildEnforcementSummary(output checkOutput) *emitEnforcement {
	enforcement := &emitEnforcement{
		Validate: output.Validate,
		Lint:     output.Lint,
		Verify:   output.Verify,
		ExitCode: output.ExitCode,
	}

	failures := append([]string{}, output.Validate.Errors...)
	failures = append(failures, output.Lint.Errors...)
	failures = append(failures, output.Verify.Errors...)
	if len(failures) > 0 {
		enforcement.Failures = failures
	}

	return enforcement
}
