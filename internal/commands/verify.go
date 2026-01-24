package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/justyn-clark/small-protocol/internal/small"
	"github.com/justyn-clark/small-protocol/internal/workspace"
	"github.com/spf13/cobra"
)

// Exit codes for verify command
const (
	ExitValid       = 0
	ExitInvalid     = 1
	ExitSystemError = 2
)

func verifyCmd() *cobra.Command {
	var strict bool
	var ci bool
	var dir string
	var workspaceFlag string

	cmd := &cobra.Command{
		Use:   "verify",
		Short: "CI/local enforcement gate for SMALL artifacts",
		Long: `Validates all .small/* artifacts for CI and local enforcement.

Performs:
  - Schema validation of all artifacts
  - Invariant enforcement (required files, ownership, format)
  - ReplayId validation (required in handoff.small.yml)

Exit codes:
  0 - All artifacts valid
  1 - Artifacts invalid (validation or invariant failures)
  2 - System error (missing directory, read errors, etc.)

Flags:
  --strict   Enable strict mode (strict invariants, secrets, insecure links)
  --ci       CI mode (minimal output, just errors)`,
		Run: func(cmd *cobra.Command, args []string) {
			p := currentPrinter()
			scope, err := workspace.ParseScope(workspaceFlag)
			if err != nil {
				p.PrintError(fmt.Sprintf("Invalid workspace scope: %v", err))
				os.Exit(ExitSystemError)
			}

			if dir == "" {
				dir = baseDir
			}

			exitCode := runVerify(dir, strict, ci, scope)
			os.Exit(exitCode)
		},
	}

	cmd.Flags().BoolVar(&strict, "strict", false, "Enable strict mode (strict invariants, secrets, insecure links)")
	cmd.Flags().BoolVar(&ci, "ci", false, "CI mode (minimal output)")
	cmd.Flags().StringVar(&dir, "dir", "", "Directory containing .small/ artifacts")
	cmd.Flags().StringVar(&workspaceFlag, "workspace", string(workspace.ScopeRoot), "Workspace scope (root, examples, or any)")

	return cmd
}

func runVerify(dir string, strict, ci bool, scope workspace.Scope) int {
	p := currentPrinter()
	smallDir := filepath.Join(dir, ".small")

	// Check if .small/ directory exists
	if _, err := os.Stat(smallDir); os.IsNotExist(err) {
		p.PrintError("Error: .small/ directory does not exist")
		if !ci {
			p.PrintError(fmt.Sprintf("Fix: small init --dir %q", dir))
		}
		return ExitSystemError
	}

	artifactsDir := resolveArtifactsDir(dir)
	if err := enforceWorkspaceScope(artifactsDir, scope); err != nil {
		p.PrintError(fmt.Sprintf("Workspace validation failed: %v", err))
		// Check if workspace.small.yml is missing
		wsPath := filepath.Join(smallDir, "workspace.small.yml")
		if !ci {
			if _, wsErr := os.Stat(wsPath); os.IsNotExist(wsErr) {
				p.PrintError(fmt.Sprintf("Fix: small init --dir %q --force", dir))
			}
		}
		return ExitInvalid
	}

	// Required files
	requiredFiles := []string{
		"intent.small.yml",
		"constraints.small.yml",
		"plan.small.yml",
		"progress.small.yml",
		"handoff.small.yml",
	}

	// Check required files exist
	var missingFiles []string
	for _, filename := range requiredFiles {
		path := filepath.Join(smallDir, filename)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			missingFiles = append(missingFiles, filename)
		}
	}

	if len(missingFiles) > 0 {
		p.PrintError("Missing required files:")
		for _, f := range missingFiles {
			p.PrintError(fmt.Sprintf("- %s", f))
		}
		if !ci {
			p.PrintError(fmt.Sprintf("Fix: small init --dir %q --force", dir))
		}
		return ExitInvalid
	}

	// Load artifacts
	artifacts, err := small.LoadAllArtifacts(artifactsDir)
	if err != nil {
		p.PrintError(fmt.Sprintf("Error loading artifacts: %v", err))
		return ExitSystemError
	}

	var allErrors []verifyError

	// Schema validation
	config := small.SchemaConfig{BaseDir: artifactsDir}
	schemaErrors := small.ValidateAllArtifactsWithConfig(artifacts, config)
	for _, err := range schemaErrors {
		allErrors = append(allErrors, verifyError{
			message: fmt.Sprintf("Schema: %v", err),
			fix:     "", // Schema errors need manual fixes
		})
	}

	// Invariant validation
	violations := small.CheckInvariants(artifacts, strict)
	for _, v := range violations {
		ve := verifyError{
			message: fmt.Sprintf("Invariant [%s]: %s", filepath.Base(v.File), v.Message),
		}
		// Add actionable fix for specific invariant violations
		ve.fix = suggestFixForInvariant(v, dir)
		allErrors = append(allErrors, ve)
	}

	// ReplayId validation (required in handoff)
	if handoff, ok := artifacts["handoff"]; ok {
		replayIdErrors := validateReplayIdWithFixes(handoff, dir)
		allErrors = append(allErrors, replayIdErrors...)
	}

	// Report results
	if len(allErrors) > 0 {
		if strict && !ci {
			report, remaining := buildStrictS2ReportFromVerifyErrors(allErrors)
			if report != nil {
				p.PrintError(p.FormatBlock("Strict S2 failed (current run only)", strictS2ReportLines(*report)))
				allErrors = remaining
			}
		}

		if len(allErrors) > 0 {
			if !ci {
				lines := make([]string, 0, len(allErrors)*2)
				for _, ve := range allErrors {
					lines = append(lines, fmt.Sprintf("- %s", ve.message))
					if ve.fix != "" {
						lines = append(lines, fmt.Sprintf("  Fix: %s", ve.fix))
					}
				}
				p.PrintError(p.FormatBlock(fmt.Sprintf("Verification failed (%d error(s))", len(allErrors)), lines))
			} else {
				for _, ve := range allErrors {
					p.PrintError(ve.message)
				}
			}
			return ExitInvalid
		}
		return ExitInvalid
	}

	if !ci {
		p.PrintInfo("Verification passed")
	}
	return ExitValid
}

// verifyError holds an error message with an optional fix command
type verifyError struct {
	message string
	fix     string
}

// suggestFixForInvariant returns an actionable fix command for common invariant violations
func suggestFixForInvariant(v small.InvariantViolation, dir string) string {
	msg := v.Message

	// Progress timestamp format issues
	if strings.Contains(msg, "timestamp") && strings.Contains(msg, "RFC3339Nano") {
		return fmt.Sprintf("small progress migrate --dir %q", dir)
	}

	// Progress entries missing for completed tasks
	if strings.Contains(msg, "progress entries missing") && strings.Contains(msg, "completed plan tasks") {
		// Extract task IDs from message
		return fmt.Sprintf("small plan --done <task-id> --dir %q  # for each missing task", dir)
	}

	// Evidence missing
	if strings.Contains(msg, "must have at least one evidence field") {
		return "Add evidence, verification, command, test, link, or commit field to the progress entry"
	}

	return ""
}

// validateReplayIdWithFixes checks replayId and returns errors with actionable fixes
func validateReplayIdWithFixes(handoff *small.Artifact, dir string) []verifyError {
	var errors []verifyError

	root := handoff.Data
	if root == nil {
		return errors
	}

	replayId, ok := root["replayId"].(map[string]interface{})
	if !ok {
		// replayId is REQUIRED - fail if missing
		errors = append(errors, verifyError{
			message: "handoff.small.yml must include replayId",
			fix:     fmt.Sprintf("small handoff --dir %q", dir),
		})
		return errors
	}

	// If replayId exists, validate its structure
	value, hasValue := replayId["value"].(string)
	source, hasSource := replayId["source"].(string)

	if !hasValue || strings.TrimSpace(value) == "" {
		errors = append(errors, verifyError{
			message: "replayId.value must be a non-empty string",
			fix:     fmt.Sprintf("small handoff --dir %q", dir),
		})
	} else {
		// Validate SHA256 format (64 hex characters)
		sha256Pattern := regexp.MustCompile(`^[a-fA-F0-9]{64}$`)
		if !sha256Pattern.MatchString(value) {
			errors = append(errors, verifyError{
				message: fmt.Sprintf("replayId.value must be a valid SHA256 hash (64 hex chars), got: %s", value),
				fix:     fmt.Sprintf("small handoff --dir %q", dir),
			})
		}
	}

	if !hasSource || strings.TrimSpace(source) == "" {
		errors = append(errors, verifyError{
			message: "replayId.source must be a non-empty string",
			fix:     fmt.Sprintf("small handoff --dir %q", dir),
		})
	} else {
		validSources := map[string]bool{"auto": true, "manual": true}
		if !validSources[source] {
			errors = append(errors, verifyError{
				message: fmt.Sprintf("replayId.source must be one of [auto, manual], got: %s", source),
				fix:     fmt.Sprintf("small handoff --dir %q", dir),
			})
		}
	}

	return errors
}

// validateReplayId checks if replayId field exists and has valid format
// replayId is REQUIRED in handoff.small.yml for verify to pass
func validateReplayId(handoff *small.Artifact) []string {
	var errors []string

	root := handoff.Data
	if root == nil {
		return errors
	}

	replayId, ok := root["replayId"].(map[string]interface{})
	if !ok {
		// replayId is REQUIRED - fail if missing
		errors = append(errors, "handoff.small.yml must include replayId (use 'small handoff' to generate)")
		return errors
	}

	// If replayId exists, validate its structure
	value, hasValue := replayId["value"].(string)
	source, hasSource := replayId["source"].(string)

	if !hasValue || strings.TrimSpace(value) == "" {
		errors = append(errors, "replayId.value must be a non-empty string")
	} else {
		// Validate SHA256 format (64 hex characters)
		sha256Pattern := regexp.MustCompile(`^[a-fA-F0-9]{64}$`)
		if !sha256Pattern.MatchString(value) {
			errors = append(errors, fmt.Sprintf("replayId.value must be a valid SHA256 hash (64 hex chars), got: %s", value))
		}
	}

	if !hasSource || strings.TrimSpace(source) == "" {
		errors = append(errors, "replayId.source must be a non-empty string")
	} else {
		validSources := map[string]bool{"auto": true, "manual": true}
		if !validSources[source] {
			errors = append(errors, fmt.Sprintf("replayId.source must be one of [auto, manual], got: %s", source))
		}
	}

	return errors
}
