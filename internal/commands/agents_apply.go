package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

func agentsApplyCmd() *cobra.Command {
	var dir string
	var file string
	var agentsModeStr string
	var overwriteAgents bool
	var force bool
	var dryRun bool
	var printOutput bool

	cmd := &cobra.Command{
		Use:   "apply",
		Short: "Write or update SMALL harness block in AGENTS.md",
		Long: `Write or update the SMALL harness block in AGENTS.md.

This command NEVER touches the .small/ directory.

Behavior:
  - If file missing: creates it with SMALL harness block
  - If file exists with SMALL block: replaces block in-place (no flags needed)
  - If file exists without SMALL block:
    - No flags: error with guidance
    - --force or --overwrite-agents: replaces entire file
    - --agents-mode=append: adds block after existing content
    - --agents-mode=prepend: adds block before existing content
  - If malformed markers: hard error with actionable message

Flags --print and --dry-run both prevent writes.
Flag --force is shorthand for --overwrite-agents.

Examples:
  small agents apply
  small agents apply --force
  small agents apply --agents-mode=append
  small agents apply --overwrite-agents
  small agents apply --dry-run
  small agents apply --file GOVERNANCE.md --agents-mode=prepend`,
		RunE: func(cmd *cobra.Command, args []string) error {
			p := currentPrinter()

			// Parse and validate flags
			agentsMode, err := ParseAgentsMode(agentsModeStr)
			if err != nil {
				return err
			}

			// --print implies --dry-run
			if printOutput {
				dryRun = true
			}

			// --force is an alias for --overwrite-agents
			if force {
				overwriteAgents = true
			}

			// Validate mutually exclusive flags
			if agentsMode != AgentsModeNone && overwriteAgents {
				return fmt.Errorf("--agents-mode and --force/--overwrite-agents are mutually exclusive")
			}

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

			// Read existing file if present
			var existingContent string
			fileExists := false
			if data, err := os.ReadFile(targetPath); err == nil {
				fileExists = true
				existingContent = string(data)
			} else if !os.IsNotExist(err) {
				return fmt.Errorf("failed to read %s: %w", fileName, err)
			}

			// Determine new content
			var newContent string
			var action string

			if !fileExists {
				// Case 1: File does not exist - create with SMALL block only
				newContent = GenerateAgentsBlock()
				action = fmt.Sprintf("Created %s", fileName)
			} else {
				// Check for existing block
				info, err := FindAgentsBlock(existingContent)
				if err != nil {
					return fmt.Errorf("malformed AGENTS.md: %w", err)
				}

				if info.Found {
					// Case 2: File exists with SMALL block - replace in-place
					newContent, err = ComposeAgentsFile(existingContent, AgentsModeAppend) // mode doesn't matter for replacement
					if err != nil {
						return fmt.Errorf("failed to compose %s: %w", fileName, err)
					}
					action = fmt.Sprintf("Updated SMALL harness block in %s", fileName)
				} else {
					// Case 3: File exists without SMALL block
					if agentsMode == AgentsModeNone && !overwriteAgents {
						return fmt.Errorf("%s already exists without SMALL harness block.\n%s", fileName, agentsFileExistsMessageForApply())
					}

					if overwriteAgents {
						newContent = GenerateAgentsBlock()
						action = fmt.Sprintf("Overwrote %s", fileName)
					} else {
						newContent, err = ComposeAgentsFile(existingContent, agentsMode)
						if err != nil {
							return fmt.Errorf("failed to compose %s: %w", fileName, err)
						}
						switch agentsMode {
						case AgentsModeAppend:
							action = fmt.Sprintf("Appended SMALL harness block to %s", fileName)
						case AgentsModePrepend:
							action = fmt.Sprintf("Prepended SMALL harness block to %s", fileName)
						}
					}
				}
			}

			// Handle --dry-run and --print
			if dryRun {
				if printOutput {
					fmt.Print(newContent)
				} else {
					// Show diff or preview
					diff := generateDiff(existingContent, newContent, targetPath)
					if diff == "" {
						p.PrintInfo("No changes needed")
					} else {
						p.PrintInfo(fmt.Sprintf("Dry run - changes to %s:\n%s", fileName, diff))
					}
				}
				return nil
			}

			// Write file
			if err := os.WriteFile(targetPath, []byte(newContent), 0644); err != nil {
				return fmt.Errorf("failed to write %s: %w", fileName, err)
			}

			p.PrintSuccess(action)
			return nil
		},
	}

	cmd.Flags().StringVar(&dir, "dir", "", "Target directory (default: current working directory)")
	cmd.Flags().StringVar(&file, "file", "", "Target file name (default: AGENTS.md)")
	cmd.Flags().StringVar(&agentsModeStr, "agents-mode", "", "How to handle existing file (append, prepend)")
	cmd.Flags().BoolVar(&overwriteAgents, "overwrite-agents", false, "Replace entire file with SMALL harness")
	cmd.Flags().BoolVarP(&force, "force", "f", false, "Replace entire file (shorthand for --overwrite-agents)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview changes without writing")
	cmd.Flags().BoolVar(&printOutput, "print", false, "Print composed result to stdout (implies --dry-run)")

	return cmd
}

// agentsFileExistsMessageForApply returns guidance when file exists without flags.
func agentsFileExistsMessageForApply() string {
	return `Use one of:
  --force (-f)             Replace entire file with SMALL harness
  --overwrite-agents       Replace entire file (same as --force)
  --agents-mode=append     Add SMALL harness after existing content
  --agents-mode=prepend    Add SMALL harness before existing content`
}

// generateDiff creates a simple diff representation between old and new content.
func generateDiff(oldContent, newContent, path string) string {
	if oldContent == newContent {
		return ""
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("--- %s (current)\n", path))
	sb.WriteString(fmt.Sprintf("+++ %s (proposed)\n", path))

	if oldContent == "" {
		// New file
		sb.WriteString("@@ -0,0 +1 @@\n")
		for _, line := range strings.Split(strings.TrimSuffix(newContent, "\n"), "\n") {
			sb.WriteString("+ ")
			sb.WriteString(line)
			sb.WriteString("\n")
		}
	} else {
		// Show a simplified diff - find the SMALL block region
		oldInfo, _ := FindAgentsBlock(oldContent)
		newInfo, _ := FindAgentsBlock(newContent)

		if oldInfo.Found && newInfo.Found {
			// Block replaced
			sb.WriteString("@@ SMALL harness block updated @@\n")
			oldBlock := oldContent[oldInfo.StartIndex:oldInfo.EndIndex]
			newBlock := newContent[newInfo.StartIndex:newInfo.EndIndex]
			if oldBlock != newBlock {
				sb.WriteString("- [old block content]\n")
				sb.WriteString("+ [new block content]\n")
			}
		} else if !oldInfo.Found && newInfo.Found {
			// Block added
			sb.WriteString("@@ SMALL harness block added @@\n")
			newBlock := newContent[newInfo.StartIndex:newInfo.EndIndex]
			for _, line := range strings.Split(strings.TrimSuffix(newBlock, "\n"), "\n") {
				sb.WriteString("+ ")
				sb.WriteString(line)
				sb.WriteString("\n")
			}
		} else {
			// Full file diff
			sb.WriteString("@@ file contents changed @@\n")
			for _, line := range strings.Split(strings.TrimSuffix(newContent, "\n"), "\n") {
				sb.WriteString("+ ")
				sb.WriteString(line)
				sb.WriteString("\n")
			}
		}
	}

	return sb.String()
}
