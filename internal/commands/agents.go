package commands

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/justyn-clark/small-protocol/internal/small"
)

// AgentsMode specifies how to handle AGENTS.md when it already exists.
type AgentsMode string

const (
	AgentsModeNone      AgentsMode = ""
	AgentsModeAppend    AgentsMode = "append"
	AgentsModePrepend   AgentsMode = "prepend"
	AgentsModeOverwrite AgentsMode = "overwrite"
)

// agentsBlockBeginPattern matches the BEGIN marker with any version.
var agentsBlockBeginPattern = regexp.MustCompile(`<!-- BEGIN SMALL HARNESS v[\d.]+ -->`)

// agentsBlockEndPattern matches the END marker with any version.
var agentsBlockEndPattern = regexp.MustCompile(`<!-- END SMALL HARNESS v[\d.]+ -->`)

// ParseAgentsMode parses the --agents-mode flag value.
func ParseAgentsMode(value string) (AgentsMode, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "":
		return AgentsModeNone, nil
	case "append":
		return AgentsModeAppend, nil
	case "prepend":
		return AgentsModePrepend, nil
	default:
		return AgentsModeNone, fmt.Errorf("invalid agents-mode: %q (valid: append, prepend)", value)
	}
}

// AgentsBlockInfo contains information about a SMALL harness block in a file.
type AgentsBlockInfo struct {
	Found      bool
	StartIndex int
	EndIndex   int
	Version    string
}

// FindAgentsBlock locates the SMALL harness block in content.
// Returns error if multiple blocks found or malformed (BEGIN without END or vice versa).
func FindAgentsBlock(content string) (AgentsBlockInfo, error) {
	beginMatches := agentsBlockBeginPattern.FindAllStringIndex(content, -1)
	endMatches := agentsBlockEndPattern.FindAllStringIndex(content, -1)

	if len(beginMatches) == 0 && len(endMatches) == 0 {
		return AgentsBlockInfo{Found: false}, nil
	}

	if len(beginMatches) > 1 {
		return AgentsBlockInfo{}, fmt.Errorf("multiple SMALL harness BEGIN markers found")
	}

	if len(endMatches) > 1 {
		return AgentsBlockInfo{}, fmt.Errorf("multiple SMALL harness END markers found")
	}

	if len(beginMatches) == 1 && len(endMatches) == 0 {
		return AgentsBlockInfo{}, fmt.Errorf("SMALL harness BEGIN marker found without END marker")
	}

	if len(beginMatches) == 0 && len(endMatches) == 1 {
		return AgentsBlockInfo{}, fmt.Errorf("SMALL harness END marker found without BEGIN marker")
	}

	beginIdx := beginMatches[0][0]
	endIdx := endMatches[0][1] // End of the END marker

	if beginIdx >= endMatches[0][0] {
		return AgentsBlockInfo{}, fmt.Errorf("SMALL harness END marker appears before BEGIN marker")
	}

	// Extract version from BEGIN marker
	beginMarker := content[beginMatches[0][0]:beginMatches[0][1]]
	version := extractVersionFromMarker(beginMarker)

	return AgentsBlockInfo{
		Found:      true,
		StartIndex: beginIdx,
		EndIndex:   endIdx,
		Version:    version,
	}, nil
}

// extractVersionFromMarker extracts the version string from a BEGIN marker.
func extractVersionFromMarker(marker string) string {
	// Match "v1.0.0" pattern
	versionPattern := regexp.MustCompile(`v([\d.]+)`)
	match := versionPattern.FindStringSubmatch(marker)
	if len(match) >= 2 {
		return match[1]
	}
	return ""
}

// ComposeAgentsFile composes the final AGENTS.md content based on mode.
func ComposeAgentsFile(existingContent string, mode AgentsMode) (string, error) {
	block := GenerateAgentsBlock()

	// If no existing content, just return the block
	if existingContent == "" {
		return block, nil
	}

	// Find existing block
	info, err := FindAgentsBlock(existingContent)
	if err != nil {
		return "", err
	}

	if info.Found {
		// Replace existing block in-place
		before := existingContent[:info.StartIndex]
		after := existingContent[info.EndIndex:]

		// Ensure proper newline handling
		before = strings.TrimRight(before, "\n")
		after = strings.TrimLeft(after, "\n")

		if before == "" && after == "" {
			return block, nil
		}

		if before == "" {
			return block + "\n" + after, nil
		}

		if after == "" {
			return before + "\n\n" + block, nil
		}

		return before + "\n\n" + block + "\n" + after, nil
	}

	// No existing block - append or prepend
	existingContent = strings.TrimSpace(existingContent)

	switch mode {
	case AgentsModeAppend:
		return existingContent + "\n\n" + block, nil
	case AgentsModePrepend:
		return block + "\n" + existingContent + "\n", nil
	default:
		return "", fmt.Errorf("unexpected agents mode: %s", mode)
	}
}

// ValidateAgentsModeFlags checks that agents-mode flags are not conflicting.
func ValidateAgentsModeFlags(agentsMode AgentsMode, noAgents, overwriteAgents bool) error {
	count := 0
	if agentsMode != AgentsModeNone {
		count++
	}
	if noAgents {
		count++
	}
	if overwriteAgents {
		count++
	}

	if count > 1 {
		return fmt.Errorf("--agents-mode, --no-agents, and --overwrite-agents are mutually exclusive")
	}

	return nil
}

// AgentsFileExistsMessage returns the message to show when AGENTS.md exists with no flags.
func AgentsFileExistsMessage() string {
	return `AGENTS.md already exists.
Use one of:
  --overwrite-agents       Replace entire file with SMALL harness
  --agents-mode=append     Add SMALL harness after existing content
  --agents-mode=prepend    Add SMALL harness before existing content`
}

// CurrentHarnessVersion returns the current SMALL harness version.
func CurrentHarnessVersion() string {
	return small.ProtocolVersion
}
