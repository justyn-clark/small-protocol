package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/justyn-clark/small-protocol/internal/small"
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

	cmd := &cobra.Command{
		Use:   "verify",
		Short: "CI/local enforcement gate for SMALL artifacts",
		Long: `Validates all .small/* artifacts for CI and local enforcement.

Performs:
  - Schema validation of all artifacts
  - Invariant enforcement (required files, ownership, format)
  - ReplayId format validation (if present)

Exit codes:
  0 - All artifacts valid
  1 - Artifacts invalid (validation or invariant failures)
  2 - System error (missing directory, read errors, etc.)

Flags:
  --strict   Enable strict mode (check for secrets, insecure links)
  --ci       CI mode (minimal output, just errors)`,
		Run: func(cmd *cobra.Command, args []string) {
			if dir == "" {
				dir = baseDir
			}

			exitCode := runVerify(dir, strict, ci)
			os.Exit(exitCode)
		},
	}

	cmd.Flags().BoolVar(&strict, "strict", false, "Enable strict mode (secrets, insecure links)")
	cmd.Flags().BoolVar(&ci, "ci", false, "CI mode (minimal output)")
	cmd.Flags().StringVar(&dir, "dir", "", "Directory containing .small/ artifacts")

	return cmd
}

func runVerify(dir string, strict, ci bool) int {
	smallDir := filepath.Join(dir, ".small")

	// Check if .small/ directory exists
	if _, err := os.Stat(smallDir); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Error: .small/ directory does not exist\n")
		return ExitSystemError
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
		fmt.Fprintf(os.Stderr, "Missing required files:\n")
		for _, f := range missingFiles {
			fmt.Fprintf(os.Stderr, "  - %s\n", f)
		}
		return ExitInvalid
	}

	// Load artifacts
	artifactsDir := resolveArtifactsDir(dir)
	artifacts, err := small.LoadAllArtifacts(artifactsDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading artifacts: %v\n", err)
		return ExitSystemError
	}

	var allErrors []string

	// Schema validation
	config := small.SchemaConfig{BaseDir: artifactsDir}
	schemaErrors := small.ValidateAllArtifactsWithConfig(artifacts, config)
	for _, err := range schemaErrors {
		allErrors = append(allErrors, fmt.Sprintf("Schema: %v", err))
	}

	// Invariant validation
	violations := small.CheckInvariants(artifacts, strict)
	for _, v := range violations {
		allErrors = append(allErrors, fmt.Sprintf("Invariant [%s]: %s", filepath.Base(v.File), v.Message))
	}

	// ReplayId format validation (if present in handoff)
	if handoff, ok := artifacts["handoff"]; ok {
		replayIdErrors := validateReplayId(handoff)
		allErrors = append(allErrors, replayIdErrors...)
	}

	// Report results
	if len(allErrors) > 0 {
		if !ci {
			fmt.Fprintf(os.Stderr, "Verification failed with %d error(s):\n", len(allErrors))
		}
		for _, errMsg := range allErrors {
			fmt.Fprintf(os.Stderr, "  %s\n", errMsg)
		}
		return ExitInvalid
	}

	if !ci {
		fmt.Println("Verification passed")
	}
	return ExitValid
}

// validateReplayId checks if replayId field (if present) has valid format
func validateReplayId(handoff *small.Artifact) []string {
	var errors []string

	root := handoff.Data
	if root == nil {
		return errors
	}

	replayId, ok := root["replayId"].(map[string]interface{})
	if !ok {
		// replayId is optional, no error if missing
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
		validSources := map[string]bool{"cli": true, "agent": true, "ci": true, "manual": true}
		if !validSources[source] {
			errors = append(errors, fmt.Sprintf("replayId.source must be one of [cli, agent, ci, manual], got: %s", source))
		}
	}

	return errors
}
