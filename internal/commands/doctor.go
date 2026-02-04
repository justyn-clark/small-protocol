package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/justyn-clark/small-protocol/internal/small"
	"github.com/spf13/cobra"
)

type DiagnosticResult struct {
	Category   string
	Status     string // "ok", "warning", "error"
	Message    string
	Suggestion string
}

func doctorCmd() *cobra.Command {
	var dir string

	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Diagnose SMALL workspace issues and suggest fixes",
		Long: `Onboarding and debugging assistant for SMALL workspaces.

Detects:
  - Missing or malformed .small files
  - Outdated schemas
  - Invalid run states
  - Ownership violations

This command is read-only and never mutates state.
Provides actionable suggestions for resolving issues.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if dir == "" {
				dir = baseDir
			}

			results := runDoctor(dir)
			p := currentPrinter()
			printDiagnostics(results)
			maybePrintUpdateNotice(p, false)
			return nil
		},
	}

	cmd.Flags().StringVar(&dir, "dir", "", "Directory to diagnose")

	return cmd
}

func runDoctor(dir string) []DiagnosticResult {
	var results []DiagnosticResult

	smallDir := filepath.Join(dir, ".small")

	// Check 1: .small/ directory exists
	if _, err := os.Stat(smallDir); os.IsNotExist(err) {
		results = append(results, DiagnosticResult{
			Category:   "Workspace",
			Status:     "error",
			Message:    ".small/ directory does not exist",
			Suggestion: "Run: small init",
		})
		return results // Can't continue without .small/
	}

	results = append(results, DiagnosticResult{
		Category: "Workspace",
		Status:   "ok",
		Message:  ".small/ directory exists",
	})

	// Check 2: Required files
	requiredFiles := map[string]string{
		"intent.small.yml":      "human",
		"constraints.small.yml": "human",
		"plan.small.yml":        "agent",
		"progress.small.yml":    "agent",
		"handoff.small.yml":     "agent",
	}

	var missingFiles []string
	existingFiles := make(map[string]bool)

	for filename := range requiredFiles {
		path := filepath.Join(smallDir, filename)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			missingFiles = append(missingFiles, filename)
		} else {
			existingFiles[filename] = true
		}
	}

	if len(missingFiles) > 0 {
		results = append(results, DiagnosticResult{
			Category:   "Files",
			Status:     "error",
			Message:    fmt.Sprintf("Missing files: %s", strings.Join(missingFiles, ", ")),
			Suggestion: "Run: small init --force (to recreate all files)",
		})
	} else {
		results = append(results, DiagnosticResult{
			Category: "Files",
			Status:   "ok",
			Message:  "All required files present",
		})
	}

	// Only continue with validation if all files exist
	if len(missingFiles) > 0 {
		return results
	}

	// Load and validate artifacts
	artifactsDir := resolveArtifactsDir(dir)
	artifacts, err := small.LoadAllArtifacts(artifactsDir)
	if err != nil {
		results = append(results, DiagnosticResult{
			Category:   "Loading",
			Status:     "error",
			Message:    fmt.Sprintf("Failed to load artifacts: %v", err),
			Suggestion: "Check YAML syntax in .small/ files",
		})
		return results
	}

	// Check 3: Schema validation
	config := small.SchemaConfig{BaseDir: artifactsDir}
	schemaErrors := small.ValidateAllArtifactsWithConfig(artifacts, config)
	if len(schemaErrors) > 0 {
		for _, err := range schemaErrors {
			results = append(results, DiagnosticResult{
				Category:   "Schema",
				Status:     "error",
				Message:    err.Error(),
				Suggestion: "Fix the schema violation, then run: small validate",
			})
		}
	} else {
		results = append(results, DiagnosticResult{
			Category: "Schema",
			Status:   "ok",
			Message:  "All artifacts pass schema validation",
		})
	}

	// Check 4: Invariant violations
	violations := small.CheckInvariants(artifacts, false)
	if len(violations) > 0 {
		for _, v := range violations {
			results = append(results, DiagnosticResult{
				Category:   "Invariant",
				Status:     "error",
				Message:    fmt.Sprintf("[%s] %s", filepath.Base(v.File), v.Message),
				Suggestion: "Fix the invariant violation",
			})
		}
	} else {
		results = append(results, DiagnosticResult{
			Category: "Invariant",
			Status:   "ok",
			Message:  "All protocol invariants satisfied",
		})
	}

	// Check 5: Version consistency
	for artifactType, artifact := range artifacts {
		if version, ok := artifact.Data["small_version"].(string); ok {
			if version != small.ProtocolVersion {
				results = append(results, DiagnosticResult{
					Category:   "Version",
					Status:     "warning",
					Message:    fmt.Sprintf("%s has version %s, expected %s", artifactType, version, small.ProtocolVersion),
					Suggestion: fmt.Sprintf("Update small_version to %s in %s.small.yml", small.ProtocolVersion, artifactType),
				})
			}
		}
	}

	// Check 6: Run state analysis
	results = append(results, analyzeRunState(artifacts)...)

	return results
}

func analyzeRunState(artifacts map[string]*small.Artifact) []DiagnosticResult {
	var results []DiagnosticResult

	// Analyze plan tasks
	if plan, ok := artifacts["plan"]; ok {
		tasks, _ := plan.Data["tasks"].([]any)

		var pending, inProgress, completed, blocked int
		for _, t := range tasks {
			tm, _ := t.(map[string]any)
			status, _ := tm["status"].(string)
			switch status {
			case "pending", "":
				pending++
			case "in_progress":
				inProgress++
			case "completed":
				completed++
			case "blocked":
				blocked++
			}
		}

		total := len(tasks)
		if total > 0 {
			msg := fmt.Sprintf("Tasks: %d total, %d completed, %d in_progress, %d pending, %d blocked",
				total, completed, inProgress, pending, blocked)

			status := "ok"
			suggestion := ""

			if inProgress > 1 {
				status = "warning"
				suggestion = "Multiple tasks in_progress. Consider focusing on one at a time."
			}
			if blocked > 0 && inProgress == 0 && pending == 0 {
				status = "warning"
				suggestion = "All remaining tasks are blocked. Review blockers."
			}
			if completed == total {
				suggestion = "All tasks complete! Run: small handoff"
			}
			if pending > 0 && inProgress == 0 {
				suggestion = "Ready to start next task. Run: small status"
			}

			results = append(results, DiagnosticResult{
				Category:   "Run State",
				Status:     status,
				Message:    msg,
				Suggestion: suggestion,
			})
		}
	}

	// Analyze progress entries
	if progress, ok := artifacts["progress"]; ok {
		entries, _ := progress.Data["entries"].([]any)
		entryCount := len(entries)

		if entryCount == 0 {
			results = append(results, DiagnosticResult{
				Category:   "Progress",
				Status:     "ok",
				Message:    "No progress entries yet (fresh run)",
				Suggestion: "Start work and record with: small apply <command>",
			})
		} else {
			results = append(results, DiagnosticResult{
				Category: "Progress",
				Status:   "ok",
				Message:  fmt.Sprintf("%d progress entries recorded", entryCount),
			})
		}
	}

	// Analyze handoff
	if handoff, ok := artifacts["handoff"]; ok {
		resume, _ := handoff.Data["resume"].(map[string]any)
		nextSteps, _ := resume["next_steps"].([]any)

		if len(nextSteps) > 0 {
			results = append(results, DiagnosticResult{
				Category:   "Handoff",
				Status:     "ok",
				Message:    fmt.Sprintf("Handoff has %d next steps defined", len(nextSteps)),
				Suggestion: "Resume from handoff or run: small reset",
			})
		}
	}

	return results
}

func printDiagnostics(results []DiagnosticResult) {
	p := currentPrinter()
	if p == nil {
		return
	}
	p.PrintInfo("SMALL Doctor Report")
	p.PrintInfo("===================")
	p.PrintInfo("")

	// Group by category
	categories := []string{"Workspace", "Files", "Loading", "Schema", "Invariant", "Version", "Run State", "Progress", "Handoff"}
	grouped := make(map[string][]DiagnosticResult)

	for _, r := range results {
		grouped[r.Category] = append(grouped[r.Category], r)
	}

	hasErrors := false
	hasWarnings := false

	for _, cat := range categories {
		if results, ok := grouped[cat]; ok {
			for _, r := range results {
				var icon string
				switch r.Status {
				case "ok":
					icon = "[OK]"
				case "warning":
					icon = "[WARN]"
					hasWarnings = true
				case "error":
					icon = "[ERROR]"
					hasErrors = true
				}

				line := fmt.Sprintf("%s %s: %s", icon, r.Category, r.Message)
				suggestion := ""
				if r.Suggestion != "" {
					suggestion = fmt.Sprintf("     -> %s", r.Suggestion)
				}
				switch r.Status {
				case "warning":
					p.PrintWarn(line)
					if suggestion != "" {
						p.PrintWarn(suggestion)
					}
				case "error":
					p.PrintError(line)
					if suggestion != "" {
						p.PrintError(suggestion)
					}
				default:
					p.PrintInfo(line)
					if suggestion != "" {
						p.PrintInfo(suggestion)
					}
				}
			}
		}
	}

	p.PrintInfo("")
	if hasErrors {
		p.PrintError("Summary: Issues found. See suggestions above.")
	} else if hasWarnings {
		p.PrintWarn("Summary: Warnings present. Review suggestions above.")
	} else {
		p.PrintSuccess("Summary: All checks passed!")
	}
}
