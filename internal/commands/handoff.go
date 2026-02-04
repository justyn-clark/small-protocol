package commands

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/justyn-clark/small-protocol/internal/small"
	"github.com/justyn-clark/small-protocol/internal/workspace"
	"github.com/spf13/cobra"
)

// handoffOut represents the v1.0.0 handoff structure
type handoffOut struct {
	SmallVersion string      `yaml:"small_version"`
	Owner        string      `yaml:"owner"`
	Summary      string      `yaml:"summary"`
	Resume       resumeOut   `yaml:"resume"`
	Links        []linkOut   `yaml:"links"`
	ReplayId     replayIdOut `yaml:"replayId"`
	Run          *runOut     `yaml:"run,omitempty"`
}

// replayIdOut represents the required deterministic identifier for replay and session tracking
type replayIdOut struct {
	Value  string `yaml:"value"`
	Source string `yaml:"source"`
}

type runOut struct {
	CreatedAt        string `yaml:"created_at,omitempty"`
	TransitionReason string `yaml:"transition_reason,omitempty"`
	PreviousReplayID string `yaml:"previous_replay_id,omitempty"`
	PreviousRunRef   string `yaml:"previous_run_ref,omitempty"`
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
		summary       string
		dir           string
		replayId      string
		workspaceFlag string
	)

	cmd := &cobra.Command{
		Use:   "handoff",
		Short: "Generate or update handoff.small.yml",
		Long: `Generates handoff.small.yml from current plan with resume information.

	ReplayId is required in handoff.small.yml. It is generated automatically by
	hashing the run-defining artifacts (intent + plan + optional constraints).
	Use --replay-id to override with a manual value if needed.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if dir == "" {
				dir = baseDir
			}

			artifactsDir := resolveArtifactsDir(dir)
			p := currentPrinter()

			scope, err := workspace.ParseScope(workspaceFlag)
			if err != nil {
				return err
			}
			if scope == workspace.ScopeExamples {
				return fmt.Errorf("--workspace examples is not supported for handoff (use --workspace any to bypass)")
			}
			if scope != workspace.ScopeAny {
				if err := enforceWorkspaceScope(artifactsDir, workspace.ScopeRoot); err != nil {
					return err
				}
			}

			planArtifact, err := small.LoadArtifact(artifactsDir, "plan.small.yml")
			if err != nil {
				return fmt.Errorf("failed to load plan.small.yml: %w", err)
			}

			// Load progress artifact for dangling tasks check
			progressArtifact, err := small.LoadArtifact(artifactsDir, "progress.small.yml")
			if err != nil {
				return fmt.Errorf("failed to load progress.small.yml: %w", err)
			}

			// Check for dangling tasks (tasks with progress but not completed/blocked)
			danglingTasks := small.CheckDanglingTasks(planArtifact, progressArtifact)
			if len(danglingTasks) > 0 {
				sort.Slice(danglingTasks, func(i, j int) bool {
					return danglingTasks[i].ID < danglingTasks[j].ID
				})
				p := currentPrinter()
				lines := []string{
					fmt.Sprintf("Why: %d task(s) have progress but are not completed or blocked.", len(danglingTasks)),
					"Tasks:",
				}
				limit := defaultListCap
				for i, task := range danglingTasks {
					if i >= limit {
						lines = append(lines, fmt.Sprintf("and %d more", len(danglingTasks)-limit))
						break
					}
					status := task.Status
					if status == "" {
						status = "pending"
					}
					lines = append(lines, fmt.Sprintf("- %s: %s (status: %s)", task.ID, task.Title, status))
				}
				lines = append(lines, "", "Fix:")
				lines = append(lines, "small checkpoint --task <id> --status completed --evidence \"...\"")
				lines = append(lines, "small checkpoint --task <id> --status blocked --evidence \"...\"")
				p.PrintError(p.FormatBlock("Handoff blocked by dangling tasks", lines))
				return fmt.Errorf("dangling tasks detected: %d task(s) have progress but are not completed or blocked", len(danglingTasks))
			}

			h, err := buildHandoff(artifactsDir, summary, replayId, nil, nil, nil, defaultNextStepsLimit)
			if err != nil {
				return err
			}

			if err := writeHandoff(artifactsDir, h); err != nil {
				return err
			}

			p.PrintInfo(fmt.Sprintf("Generated handoff.small.yml with %d next steps", len(h.Resume.NextSteps)))
			p.PrintInfo(fmt.Sprintf("replayId: %s (source: %s)", h.ReplayId.Value[:16]+"...", h.ReplayId.Source))
			yml, _ := small.MarshalYAMLWithQuotedVersion(h)
			p.PrintInfo(string(yml))
			return nil
		},
	}

	cmd.Flags().StringVar(&summary, "summary", "", "Summary description for the handoff")
	cmd.Flags().StringVar(&dir, "dir", ".", "Directory containing .small/ artifacts")
	cmd.Flags().StringVar(&replayId, "replay-id", "", "Manual replayId override (64 hex chars, normalized to lowercase)")
	cmd.Flags().StringVar(&workspaceFlag, "workspace", string(workspace.ScopeRoot), "Workspace scope (root or any)")

	return cmd
}

// replayIdPattern validates that a manual replayId is 64 hex chars (uppercase or lowercase)
var replayIdPattern = regexp.MustCompile(`^[a-fA-F0-9]{64}$`)

// generateReplayId creates a deterministic SHA256 hash from run-defining artifacts
// or validates and uses a manual override if provided.
//
// Auto mode: sha256(intent.small.yml + "\n" + plan.small.yml [+ "\n" + constraints.small.yml if present])
// Manual mode: validates --replay-id matches ^[a-fA-F0-9]{64}$ (accepts both cases) and normalizes to lowercase
func generateReplayId(smallDir string, manualReplayId string) (*replayIdOut, error) {
	// Manual mode: validate and use provided replayId (normalized to lowercase)
	if manualReplayId != "" {
		if !replayIdPattern.MatchString(manualReplayId) {
			return nil, fmt.Errorf("invalid replayId format: must be 64 hex chars, got: %s", manualReplayId)
		}
		return &replayIdOut{
			Value:  strings.ToLower(manualReplayId),
			Source: "manual",
		}, nil
	}

	// Auto mode: generate deterministic hash from artifacts
	intentPath := filepath.Join(smallDir, "intent.small.yml")
	planPath := filepath.Join(smallDir, "plan.small.yml")
	constraintsPath := filepath.Join(smallDir, "constraints.small.yml")

	// Read intent (required)
	intentBytes, err := os.ReadFile(intentPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read intent.small.yml: %w", err)
	}

	// Read plan (required)
	planBytes, err := os.ReadFile(planPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read plan.small.yml: %w", err)
	}

	// Normalize line endings to LF
	intentContent := normalizeLineEndings(intentBytes)
	planContent := normalizeLineEndings(planBytes)

	// Build content to hash: intent + "\n" + plan [+ "\n" + constraints]
	var hashContent []byte
	hashContent = append(hashContent, intentContent...)
	hashContent = append(hashContent, '\n')
	hashContent = append(hashContent, planContent...)

	// Optionally include constraints if present
	if constraintsBytes, err := os.ReadFile(constraintsPath); err == nil {
		constraintsContent := normalizeLineEndings(constraintsBytes)
		hashContent = append(hashContent, '\n')
		hashContent = append(hashContent, constraintsContent...)
	}

	hash := sha256.Sum256(hashContent)
	return &replayIdOut{
		Value:  hex.EncodeToString(hash[:]),
		Source: "auto",
	}, nil
}

// normalizeLineEndings converts CRLF and CR to LF for consistent hashing
func normalizeLineEndings(content []byte) []byte {
	// Replace CRLF with LF, then CR with LF
	s := strings.ReplaceAll(string(content), "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	return []byte(s)
}

func stringVal(v any) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", v)
}
