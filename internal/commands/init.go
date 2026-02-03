package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/justyn-clark/small-protocol/internal/small"
	"github.com/justyn-clark/small-protocol/internal/workspace"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

func initCmd() *cobra.Command {
	var force bool
	var intentStr string
	var dir string
	var noAgents bool
	var overwriteAgents bool
	var agentsModeStr string

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize a new SMALL v" + small.ProtocolVersion + " project",
		Long: `Creates .small/ directory, all five canonical files from templates, and AGENTS.md.

AGENTS.md handling:
  - If AGENTS.md does not exist: creates it with SMALL harness block
  - If AGENTS.md exists (no flags): exits with guidance message
  - --overwrite-agents: replaces entire file with SMALL harness block
  - --agents-mode=append: adds SMALL harness block after existing content
  - --agents-mode=prepend: adds SMALL harness block before existing content
  - --no-agents: skips AGENTS.md entirely

If a SMALL harness block already exists in AGENTS.md, it will be replaced in-place
regardless of append/prepend mode.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			targetDir := baseDir
			if dir != "" {
				targetDir = dir
			}
			targetDir = resolveArtifactsDir(targetDir)

			// Parse and validate agents mode flags
			agentsMode, err := ParseAgentsMode(agentsModeStr)
			if err != nil {
				return err
			}
			if err := ValidateAgentsModeFlags(agentsMode, noAgents, overwriteAgents); err != nil {
				return err
			}

			smallDir := filepath.Join(targetDir, small.SmallDir)

			if !force {
				if _, err := os.Stat(smallDir); err == nil {
					return fmt.Errorf(".small/ directory already exists. Use --force to overwrite")
				}
			}

			if err := os.MkdirAll(smallDir, 0755); err != nil {
				return fmt.Errorf("failed to create .small directory: %w", err)
			}

			templates := map[string]string{
				"intent.small.yml":      intentTemplate,
				"constraints.small.yml": constraintsTemplate,
				"plan.small.yml":        planTemplate,
				"progress.small.yml":    progressTemplate,
			}

			for filename, template := range templates {
				content := template

				// Seed intent if provided
				if filename == "intent.small.yml" && strings.TrimSpace(intentStr) != "" {
					var data map[string]interface{}
					if err := yaml.Unmarshal([]byte(template), &data); err == nil {
						data["intent"] = intentStr
						updated, err := small.MarshalYAMLWithQuotedVersion(data)
						if err == nil {
							content = string(updated)
						}
					}
				}

				path := filepath.Join(smallDir, filename)
				if err := os.WriteFile(path, []byte(content), 0644); err != nil {
					return fmt.Errorf("failed to write %s: %w", path, err)
				}
			}

			handoff, err := buildHandoff(targetDir, "Project initialized", "", nil, nil, nil, defaultNextStepsLimit)
			if err != nil {
				return err
			}
			if err := writeHandoff(targetDir, handoff); err != nil {
				return err
			}

			if err := workspace.Save(targetDir, workspace.KindRepoRoot); err != nil {
				return err
			}

			// Handle AGENTS.md
			if err := handleAgentsFile(targetDir, noAgents, overwriteAgents, agentsMode); err != nil {
				return err
			}

			entry := map[string]interface{}{
				"task_id":   "init",
				"status":    "completed",
				"timestamp": formatProgressTimestamp(time.Now().UTC()),
				"evidence":  "Initialized .small workspace and seeded canonical artifacts",
				"notes":     fmt.Sprintf("small init in %s", targetDir),
				"command":   "small init",
			}
			if err := appendProgressEntry(targetDir, entry); err != nil {
				return fmt.Errorf("failed to record init progress: %w", err)
			}

			fmt.Printf("Initialized SMALL v%s project in %s\n", small.ProtocolVersion, smallDir)
			return nil
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Overwrite existing .small/ directory")
	cmd.Flags().StringVar(&intentStr, "intent", "", "Intent string to seed in intent.small.yml")
	cmd.Flags().StringVar(&dir, "dir", "", "Target directory for the new workspace (default: current working directory)")
	cmd.Flags().BoolVar(&noAgents, "no-agents", false, "Skip creating AGENTS.md")
	cmd.Flags().BoolVar(&overwriteAgents, "overwrite-agents", false, "Overwrite existing AGENTS.md entirely")
	cmd.Flags().StringVar(&agentsModeStr, "agents-mode", "", "How to handle existing AGENTS.md (append, prepend)")

	return cmd
}

// handleAgentsFile handles AGENTS.md creation/modification based on flags.
func handleAgentsFile(targetDir string, noAgents, overwriteAgents bool, agentsMode AgentsMode) error {
	if noAgents {
		return nil
	}

	agentsPath := filepath.Join(targetDir, "AGENTS.md")
	fileExists := false
	var existingContent string

	if data, err := os.ReadFile(agentsPath); err == nil {
		fileExists = true
		existingContent = string(data)
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("failed to read AGENTS.md: %w", err)
	}

	// Case 1: File does not exist - create with SMALL block only
	if !fileExists {
		if err := os.WriteFile(agentsPath, []byte(GenerateAgentsBlock()), 0644); err != nil {
			return fmt.Errorf("failed to write AGENTS.md: %w", err)
		}
		fmt.Println("Created AGENTS.md")
		return nil
	}

	// Case 2: File exists, no flags - error with guidance
	if agentsMode == AgentsModeNone && !overwriteAgents {
		fmt.Println(AgentsFileExistsMessage())
		return fmt.Errorf("AGENTS.md already exists")
	}

	// Case 3: --overwrite-agents - replace entire file
	if overwriteAgents {
		if err := os.WriteFile(agentsPath, []byte(GenerateAgentsBlock()), 0644); err != nil {
			return fmt.Errorf("failed to write AGENTS.md: %w", err)
		}
		fmt.Println("Overwrote AGENTS.md")
		return nil
	}

	// Case 4: --agents-mode=append or prepend
	newContent, err := ComposeAgentsFile(existingContent, agentsMode)
	if err != nil {
		return fmt.Errorf("failed to compose AGENTS.md: %w", err)
	}

	if err := os.WriteFile(agentsPath, []byte(newContent), 0644); err != nil {
		return fmt.Errorf("failed to write AGENTS.md: %w", err)
	}

	// Check if we replaced an existing block or added new
	info, _ := FindAgentsBlock(existingContent)
	if info.Found {
		fmt.Println("Updated SMALL harness block in AGENTS.md")
	} else {
		switch agentsMode {
		case AgentsModeAppend:
			fmt.Println("Appended SMALL harness block to AGENTS.md")
		case AgentsModePrepend:
			fmt.Println("Prepended SMALL harness block to AGENTS.md")
		}
	}

	return nil
}
