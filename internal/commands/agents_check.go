package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

// AgentsCheckResult contains the result of agents check command.
type AgentsCheckResult struct {
	HasFile        bool    `json:"hasFile"`
	HasBlock       bool    `json:"hasBlock"`
	BlockValid     bool    `json:"blockValid"`
	BlockVersion   string  `json:"blockVersion,omitempty"`
	BlockSpan      [2]int  `json:"blockSpan,omitempty"` // [start, end] indices
	HasDrift       bool    `json:"hasDrift"`
	Error          string  `json:"error,omitempty"`
	Recommendation string  `json:"recommendation,omitempty"`
	FilePath       string  `json:"filePath"`
}

func agentsCheckCmd() *cobra.Command {
	var dir string
	var file string
	var allowMissing bool
	var strict bool
	var format string

	cmd := &cobra.Command{
		Use:   "check",
		Short: "Validate AGENTS.md harness block (read-only)",
		Long: `Check AGENTS.md for a valid SMALL harness block.

This command is read-only and NEVER modifies any files.

Exit codes:
  0  OK (file exists and block is valid, or --allow-missing and file missing)
  1  Missing / invalid / drift detected
  2  Usage error

Checks performed:
  - File existence
  - BEGIN/END marker detection and pairing
  - Marker uniqueness (no duplicates)
  - Block structure validation
  - Drift detection (compares block to canonical template)

Examples:
  small agents check
  small agents check --strict
  small agents check --allow-missing
  small agents check --format json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			p := currentPrinter()

			// Resolve target path
			targetDir := baseDir
			if dir != "" {
				targetDir = dir
			}
			absDir, err := filepath.Abs(targetDir)
			if err != nil {
				return fmt.Errorf("failed to resolve directory: %w", err)
			}

			fileName := "AGENTS.md"
			if file != "" {
				fileName = file
			}
			targetPath := filepath.Join(absDir, fileName)

			result := AgentsCheckResult{
				FilePath: targetPath,
			}

			// Read file
			data, err := os.ReadFile(targetPath)
			if err != nil {
				if os.IsNotExist(err) {
					result.HasFile = false
					if allowMissing {
						result.Recommendation = "File missing (allowed by --allow-missing)"
						return outputCheckResult(p, result, format, false)
					}
					result.Error = fmt.Sprintf("%s not found", fileName)
					result.Recommendation = "Run: small agents apply"
					return outputCheckResult(p, result, format, true)
				}
				return fmt.Errorf("failed to read %s: %w", fileName, err)
			}

			result.HasFile = true
			content := string(data)

			// Find block
			info, err := FindAgentsBlock(content)
			if err != nil {
				result.Error = err.Error()
				result.Recommendation = "Fix malformed markers in " + fileName
				return outputCheckResult(p, result, format, true)
			}

			if !info.Found {
				result.HasBlock = false
				if strict {
					result.Error = fmt.Sprintf("%s exists but has no SMALL harness block", fileName)
					result.Recommendation = "Run: small agents apply --agents-mode=append"
					return outputCheckResult(p, result, format, true)
				}
				result.Recommendation = "No SMALL harness block found (use --strict to enforce)"
				return outputCheckResult(p, result, format, false)
			}

			result.HasBlock = true
			result.BlockVersion = info.Version
			result.BlockSpan = [2]int{info.StartIndex, info.EndIndex}

			// Validate block structure
			existingBlock := content[info.StartIndex:info.EndIndex]
			blockValid, validationErrors := validateBlockStructure(existingBlock)
			result.BlockValid = blockValid

			if !blockValid {
				result.Error = fmt.Sprintf("block structure invalid: %s", strings.Join(validationErrors, "; "))
				result.Recommendation = "Run: small agents apply to regenerate block"
				return outputCheckResult(p, result, format, true)
			}

			// Check for drift
			canonicalBlock := GenerateAgentsBlock()
			hasDrift := strings.TrimSpace(existingBlock) != strings.TrimSpace(canonicalBlock)
			result.HasDrift = hasDrift

			if hasDrift {
				result.Recommendation = "Block differs from canonical template. Run: small agents apply"
				// Drift is informational, not an error unless strict
				if strict {
					result.Error = "block content differs from canonical template"
					return outputCheckResult(p, result, format, true)
				}
			}

			return outputCheckResult(p, result, format, false)
		},
	}

	cmd.Flags().StringVar(&dir, "dir", "", "Target directory (default: current working directory)")
	cmd.Flags().StringVar(&file, "file", "", "Target file name (default: AGENTS.md)")
	cmd.Flags().BoolVar(&allowMissing, "allow-missing", false, "Do not fail if file is missing")
	cmd.Flags().BoolVar(&strict, "strict", false, "Fail if file exists without SMALL block or if drift detected")
	cmd.Flags().StringVar(&format, "format", "text", "Output format (text, json)")

	return cmd
}

// validateBlockStructure checks that the block contains expected content.
func validateBlockStructure(block string) (bool, []string) {
	var errors []string

	requiredHeadings := []string{
		"Ownership Rules",
		"Artifact Rules",
		"Strict Mode Rules",
	}

	for _, heading := range requiredHeadings {
		if !strings.Contains(block, heading) {
			errors = append(errors, fmt.Sprintf("missing section: %s", heading))
		}
	}

	// Check for BEGIN/END markers
	if !strings.Contains(block, "<!-- BEGIN SMALL HARNESS") {
		errors = append(errors, "missing BEGIN marker")
	}
	if !strings.Contains(block, "<!-- END SMALL HARNESS") {
		errors = append(errors, "missing END marker")
	}

	return len(errors) == 0, errors
}

// outputCheckResult outputs the check result in the specified format.
func outputCheckResult(p *Printer, result AgentsCheckResult, format string, isError bool) error {
	if format == "json" {
		data, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal result: %w", err)
		}
		fmt.Println(string(data))
		if isError {
			return fmt.Errorf("check failed")
		}
		return nil
	}

	// Text format
	var lines []string

	if result.HasFile {
		lines = append(lines, fmt.Sprintf("File: %s [exists]", result.FilePath))
	} else {
		lines = append(lines, fmt.Sprintf("File: %s [missing]", result.FilePath))
	}

	if result.HasBlock {
		lines = append(lines, fmt.Sprintf("Block: found (v%s)", result.BlockVersion))
		if result.BlockValid {
			lines = append(lines, "Structure: valid")
		} else {
			lines = append(lines, "Structure: invalid")
		}
		if result.HasDrift {
			lines = append(lines, "Drift: detected (block differs from template)")
		} else {
			lines = append(lines, "Drift: none")
		}
	} else if result.HasFile {
		lines = append(lines, "Block: not found")
	}

	if result.Error != "" {
		lines = append(lines, fmt.Sprintf("Error: %s", result.Error))
	}

	if result.Recommendation != "" {
		lines = append(lines, fmt.Sprintf("Recommendation: %s", result.Recommendation))
	}

	output := strings.Join(lines, "\n")
	if isError {
		p.PrintError(output)
		return fmt.Errorf("check failed")
	}
	p.PrintSuccess(output)
	return nil
}
