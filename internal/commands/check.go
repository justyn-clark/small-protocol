package commands

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/justyn-clark/small-protocol/internal/small"
	"github.com/justyn-clark/small-protocol/internal/workspace"
	"github.com/spf13/cobra"
)

type checkStageResult struct {
	Status string   `json:"status"`
	Errors []string `json:"errors,omitempty"`
}

type checkOutput struct {
	Validate checkStageResult `json:"validate"`
	Lint     checkStageResult `json:"lint"`
	Verify   checkStageResult `json:"verify"`
	ExitCode int              `json:"exit_code"`
}

func checkCmd() *cobra.Command {
	var strict bool
	var ci bool
	var dir string
	var workspaceFlag string
	var jsonOutput bool
	var formatStrict bool

	cmd := &cobra.Command{
		Use:   "check",
		Short: "Run validate, lint, and verify",
		Run: func(cmd *cobra.Command, args []string) {
			p := currentPrinter()
			if dir == "" {
				dir = baseDir
			}
			scope, err := workspace.ParseScope(workspaceFlag)
			if err != nil {
				p.PrintError(fmt.Sprintf("Invalid workspace scope: %v", err))
				os.Exit(ExitSystemError)
			}

			code, output, err := runCheck(dir, strict, ci, jsonOutput, scope, formatStrict)
			if err != nil {
				p.PrintError(fmt.Sprintf("Error: %v", err))
				os.Exit(ExitSystemError)
			}

			if jsonOutput {
				if err := outputCheckJSON(output); err != nil {
					p.PrintError(fmt.Sprintf("Error: %v", err))
					os.Exit(ExitSystemError)
				}
				os.Exit(code)
			}

			if code == ExitValid && !ci {
				p.PrintSuccess("Check passed")
			}
			os.Exit(code)
		},
	}

	cmd.Flags().BoolVar(&strict, "strict", false, "Enable strict mode (strict invariants, secrets, insecure links)")
	cmd.Flags().BoolVar(&formatStrict, "format-strict", false, "Treat small_version formatting drift as an error")
	cmd.Flags().BoolVar(&ci, "ci", false, "CI mode (minimal output)")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output in JSON format")
	cmd.Flags().StringVar(&dir, "dir", ".", "Directory containing .small/ artifacts")
	cmd.Flags().StringVar(&workspaceFlag, "workspace", string(workspace.ScopeRoot), "Workspace scope (root, examples, or any)")

	return cmd
}

func runCheck(dir string, strict, ci, jsonOutput bool, scope workspace.Scope, formatStrict bool) (int, checkOutput, error) {
	artifactsDir := resolveArtifactsDir(dir)
	p := currentPrinter()
	if scope != workspace.ScopeAny {
		if err := enforceWorkspaceScope(artifactsDir, scope); err != nil {
			return ExitInvalid, checkOutput{}, err
		}
	}

	result := checkOutput{
		Validate: checkStageResult{Status: "ok"},
		Lint:     checkStageResult{Status: "ok"},
		Verify:   checkStageResult{Status: "ok"},
		ExitCode: ExitValid,
	}

	validationErrors, err := runValidateArtifacts(artifactsDir, small.SchemaConfig{BaseDir: artifactsDir})
	if err != nil {
		result.Validate.Status = "error"
		result.Validate.Errors = []string{err.Error()}
		result.ExitCode = ExitSystemError
		return ExitSystemError, result, err
	}
	if len(validationErrors) > 0 {
		result.Validate.Status = "failed"
		var errorLines []string
		for _, vErr := range validationErrors {
			result.Validate.Errors = append(result.Validate.Errors, vErr.Error())
			errorLines = append(errorLines, vErr.Error())
		}
		result.ExitCode = ExitInvalid
		if !ci && !jsonOutput {
			p.PrintError(p.FormatBlock(fmt.Sprintf("Validate failed (%d error(s))", len(validationErrors)), errorLines))
		}
		return ExitInvalid, result, nil
	}

	lintViolations, err := runLintArtifacts(artifactsDir, strict)
	if err != nil {
		result.Lint.Status = "error"
		result.Lint.Errors = []string{err.Error()}
		result.ExitCode = ExitSystemError
		return ExitSystemError, result, err
	}
	if len(lintViolations) > 0 {
		result.Lint.Status = "failed"
		for _, violation := range lintViolations {
			result.Lint.Errors = append(result.Lint.Errors, fmt.Sprintf("%s: %s", violation.File, violation.Message))
		}
		result.ExitCode = ExitInvalid
		if !ci && !jsonOutput {
			if strict {
				report, other := buildStrictS2ReportFromViolations(lintViolations)
				if report != nil {
					p.PrintError(p.FormatBlock("Strict S2 failed (current run only)", strictS2ReportLines(*report)))
					if len(other) > 0 {
						p.PrintError(p.FormatBlock("Other invariant violations", formatBulletList(other)))
					}
					return ExitInvalid, result, nil
				}
			}
			// Always show the actual violations so humans can understand what's wrong
			var violationLines []string
			for _, v := range lintViolations {
				violationLines = append(violationLines, fmt.Sprintf("%s: %s", v.File, v.Message))
			}
			p.PrintError(p.FormatBlock(fmt.Sprintf("Lint failed (%d violation(s))", len(lintViolations)), violationLines))
		}
		return ExitInvalid, result, nil
	}

	versionWarnings, err := findVersionFormatWarnings(artifactsDir)
	if err != nil {
		result.Lint.Status = "error"
		result.Lint.Errors = []string{err.Error()}
		result.ExitCode = ExitSystemError
		return ExitSystemError, result, err
	}
	if len(versionWarnings) > 0 {
		if formatStrict {
			result.Lint.Status = "failed"
			var warningLines []string
			for _, warning := range versionWarnings {
				msg := fmt.Sprintf("%s: small_version should be a quoted string", warning)
				result.Lint.Errors = append(result.Lint.Errors, msg)
				warningLines = append(warningLines, msg)
			}
			warningLines = append(warningLines, "", "Fix: small fix --versions")
			result.ExitCode = ExitInvalid
			if !ci && !jsonOutput {
				p.PrintError(p.FormatBlock(fmt.Sprintf("Lint failed (%d violation(s))", len(versionWarnings)), warningLines))
			}
			return ExitInvalid, result, nil
		}
		if !ci && !jsonOutput {
			for _, warning := range versionWarnings {
				p.PrintWarn(fmt.Sprintf("%s: small_version should be a quoted string. Fix: small fix --versions", warning))
			}
		}
	}

	verifyCi := ci || jsonOutput
	verifyCode := runVerify(artifactsDir, strict, verifyCi, scope)
	if verifyCode != ExitValid {
		result.Verify.Status = "failed"
		result.ExitCode = verifyCode
		return verifyCode, result, nil
	}

	return ExitValid, result, nil
}

func outputCheckJSON(payload interface{}) error {
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}
