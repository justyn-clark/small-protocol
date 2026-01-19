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

	cmd := &cobra.Command{
		Use:   "check",
		Short: "Run validate, lint, and verify",
		Run: func(cmd *cobra.Command, args []string) {
			if dir == "" {
				dir = baseDir
			}
			scope, err := workspace.ParseScope(workspaceFlag)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Invalid workspace scope: %v\n", err)
				os.Exit(ExitSystemError)
			}

			code, output, err := runCheck(dir, strict, ci, jsonOutput, scope)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(ExitSystemError)
			}

			if jsonOutput {
				if err := outputCheckJSON(output); err != nil {
					fmt.Fprintf(os.Stderr, "Error: %v\n", err)
					os.Exit(ExitSystemError)
				}
				os.Exit(code)
			}

			if code == ExitValid && !ci {
				fmt.Println("Check passed")
			}
			os.Exit(code)
		},
	}

	cmd.Flags().BoolVar(&strict, "strict", false, "Enable strict mode (strict invariants, secrets, insecure links)")
	cmd.Flags().BoolVar(&ci, "ci", false, "CI mode (minimal output)")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output in JSON format")
	cmd.Flags().StringVar(&dir, "dir", ".", "Directory containing .small/ artifacts")
	cmd.Flags().StringVar(&workspaceFlag, "workspace", string(workspace.ScopeRoot), "Workspace scope (root, examples, or any)")

	return cmd
}

func runCheck(dir string, strict, ci, jsonOutput bool, scope workspace.Scope) (int, checkOutput, error) {
	artifactsDir := resolveArtifactsDir(dir)
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
		for _, vErr := range validationErrors {
			result.Validate.Errors = append(result.Validate.Errors, vErr.Error())
		}
		result.ExitCode = ExitInvalid
		if !ci && !jsonOutput {
			fmt.Fprintf(os.Stderr, "Validate failed with %d error(s)\n", len(validationErrors))
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
			fmt.Fprintf(os.Stderr, "Lint failed with %d violation(s)\n", len(lintViolations))
		}
		return ExitInvalid, result, nil
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
