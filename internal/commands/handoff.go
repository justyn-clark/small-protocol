package commands

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/justyn-clark/small-protocol/internal/small"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// handoffOut represents the v1.0.0 handoff structure
type handoffOut struct {
	SmallVersion string       `yaml:"small_version"`
	Owner        string       `yaml:"owner"`
	Summary      string       `yaml:"summary"`
	Resume       resumeOut    `yaml:"resume"`
	Links        []linkOut    `yaml:"links"`
	ReplayId     *replayIdOut `yaml:"replayId,omitempty"`
}

// replayIdOut represents optional deterministic metadata for replay identification
type replayIdOut struct {
	Value  string `yaml:"value"`
	Source string `yaml:"source"`
}

type resumeOut struct {
	CurrentTaskID string   `yaml:"current_task_id"`
	NextSteps     []string `yaml:"next_steps"`
}

type linkOut struct {
	URL         string `yaml:"url,omitempty"`
	Description string `yaml:"description,omitempty"`
}

func handoffCmd() *cobra.Command {
	var (
		summary      string
		dir          string
		withReplayId bool
	)

	cmd := &cobra.Command{
		Use:   "handoff",
		Short: "Generate or update handoff.small.yml",
		Long: `Generates handoff.small.yml from current plan with resume information.

Use --replay-id to include deterministic metadata for replay identification.
Note: replayId is optional metadata; git history remains the canonical audit trail.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if dir == "" {
				dir = baseDir
			}

			artifactsDir := resolveArtifactsDir(dir)

			planArtifact, err := small.LoadArtifact(artifactsDir, "plan.small.yml")
			if err != nil {
				return fmt.Errorf("failed to load plan.small.yml: %w", err)
			}

			// Build next_steps from pending tasks and find current task
			var nextSteps []string
			var currentTaskID string
			if rawTasks, ok := planArtifact.Data["tasks"].([]interface{}); ok {
				for _, t := range rawTasks {
					m, ok := t.(map[string]interface{})
					if !ok {
						continue
					}

					taskID := stringVal(m["id"])
					title := stringVal(m["title"])
					status := stringVal(m["status"])

					// Find the first in_progress task as current
					if status == "in_progress" && currentTaskID == "" {
						currentTaskID = taskID
					}

					// Add pending and in_progress tasks to next_steps
					if status == "pending" || status == "in_progress" || status == "" {
						step := title
						if step == "" {
							step = taskID
						}
						if step != "" {
							nextSteps = append(nextSteps, step)
						}
					}
				}
			}

			// Use provided summary or generate a default one
			handoffSummary := summary
			if handoffSummary == "" {
				handoffSummary = "Handoff generated from current plan state"
			}

			h := handoffOut{
				SmallVersion: small.ProtocolVersion,
				Owner:        "agent",
				Summary:      handoffSummary,
				Resume: resumeOut{
					CurrentTaskID: currentTaskID,
					NextSteps:     nextSteps,
				},
				Links: []linkOut{},
			}

			// Add replayId if requested
			if withReplayId {
				h.ReplayId = generateReplayId(handoffSummary, nextSteps)
			}

			yml, err := yaml.Marshal(h)
			if err != nil {
				return fmt.Errorf("failed to marshal handoff: %w", err)
			}

			// Write to .small/handoff.small.yml
			smallDir := filepath.Join(artifactsDir, small.SmallDir)
			if err := os.MkdirAll(smallDir, 0755); err != nil {
				return fmt.Errorf("failed to create .small directory: %w", err)
			}
			outPath := filepath.Join(smallDir, "handoff.small.yml")
			if err := os.WriteFile(outPath, yml, 0644); err != nil {
				return fmt.Errorf("failed to write %s: %w", outPath, err)
			}

			fmt.Printf("Generated handoff.small.yml with %d next steps\n", len(nextSteps))
			if withReplayId {
				fmt.Printf("Included replayId: %s\n", h.ReplayId.Value[:16]+"...")
			}
			fmt.Println(string(yml))
			return nil
		},
	}

	cmd.Flags().StringVar(&summary, "summary", "", "Summary description for the handoff")
	cmd.Flags().StringVar(&dir, "dir", ".", "Directory containing .small/ artifacts")
	cmd.Flags().BoolVar(&withReplayId, "replay-id", false, "Include deterministic replayId metadata")

	return cmd
}

// generateReplayId creates a deterministic SHA256 hash from handoff content
func generateReplayId(summary string, nextSteps []string) *replayIdOut {
	// Create deterministic content for hashing
	content := fmt.Sprintf("summary:%s;steps:%s;ts:%d",
		summary,
		strings.Join(nextSteps, ","),
		time.Now().UnixNano(),
	)

	hash := sha256.Sum256([]byte(content))
	return &replayIdOut{
		Value:  hex.EncodeToString(hash[:]),
		Source: "cli",
	}
}

func stringVal(v interface{}) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", v)
}
