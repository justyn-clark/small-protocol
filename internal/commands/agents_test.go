package commands

import (
	"strings"
	"testing"

	"github.com/justyn-clark/small-protocol/internal/small"
)

func TestParseAgentsMode(t *testing.T) {
	tests := []struct {
		input   string
		want    AgentsMode
		wantErr bool
	}{
		{"", AgentsModeNone, false},
		{"append", AgentsModeAppend, false},
		{"Append", AgentsModeAppend, false},
		{"APPEND", AgentsModeAppend, false},
		{"prepend", AgentsModePrepend, false},
		{"Prepend", AgentsModePrepend, false},
		{"invalid", AgentsModeNone, true},
		{"overwrite", AgentsModeNone, true}, // overwrite is a separate flag
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := ParseAgentsMode(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseAgentsMode(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ParseAgentsMode(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestFindAgentsBlock(t *testing.T) {
	version := small.ProtocolVersion

	tests := []struct {
		name      string
		content   string
		wantFound bool
		wantErr   bool
		errMsg    string
	}{
		{
			name:      "no block",
			content:   "# My AGENTS.md\n\nSome content here.",
			wantFound: false,
			wantErr:   false,
		},
		{
			name: "valid block",
			content: `# Existing content
<!-- BEGIN SMALL HARNESS v` + version + ` -->
# SMALL Execution Harness
Some harness content
<!-- END SMALL HARNESS v` + version + ` -->
More content`,
			wantFound: true,
			wantErr:   false,
		},
		{
			name: "multiple BEGIN markers",
			content: `<!-- BEGIN SMALL HARNESS v` + version + ` -->
content
<!-- BEGIN SMALL HARNESS v` + version + ` -->
more
<!-- END SMALL HARNESS v` + version + ` -->`,
			wantFound: false,
			wantErr:   true,
			errMsg:    "multiple SMALL harness BEGIN markers",
		},
		{
			name: "multiple END markers",
			content: `<!-- BEGIN SMALL HARNESS v` + version + ` -->
content
<!-- END SMALL HARNESS v` + version + ` -->
<!-- END SMALL HARNESS v` + version + ` -->`,
			wantFound: false,
			wantErr:   true,
			errMsg:    "multiple SMALL harness END markers",
		},
		{
			name:      "BEGIN without END",
			content:   `<!-- BEGIN SMALL HARNESS v` + version + ` -->`,
			wantFound: false,
			wantErr:   true,
			errMsg:    "BEGIN marker found without END marker",
		},
		{
			name:      "END without BEGIN",
			content:   `<!-- END SMALL HARNESS v` + version + ` -->`,
			wantFound: false,
			wantErr:   true,
			errMsg:    "END marker found without BEGIN marker",
		},
		{
			name: "END before BEGIN",
			content: `<!-- END SMALL HARNESS v` + version + ` -->
<!-- BEGIN SMALL HARNESS v` + version + ` -->`,
			wantFound: false,
			wantErr:   true,
			errMsg:    "END marker appears before BEGIN marker",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info, err := FindAgentsBlock(tt.content)
			if (err != nil) != tt.wantErr {
				t.Errorf("FindAgentsBlock() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("FindAgentsBlock() error = %v, want error containing %q", err, tt.errMsg)
			}
			if info.Found != tt.wantFound {
				t.Errorf("FindAgentsBlock().Found = %v, want %v", info.Found, tt.wantFound)
			}
		})
	}
}

func TestGenerateAgentsBlock(t *testing.T) {
	block := GenerateAgentsBlock()

	// Check for BEGIN/END markers
	if !strings.Contains(block, agentsBlockBegin) {
		t.Error("GenerateAgentsBlock() missing BEGIN marker")
	}
	if !strings.Contains(block, agentsBlockEnd) {
		t.Error("GenerateAgentsBlock() missing END marker")
	}

	// Check that BEGIN comes before END
	beginIdx := strings.Index(block, agentsBlockBegin)
	endIdx := strings.Index(block, agentsBlockEnd)
	if beginIdx >= endIdx {
		t.Error("GenerateAgentsBlock() BEGIN marker should come before END marker")
	}

	// Check for required content
	requiredContent := []string{
		"SMALL Execution Harness",
		"Ownership Rules",
		"Artifact Rules",
		"Strict Mode Rules",
		"Localhost HTTP Allowlist",
		"Ralph Loop",
	}
	for _, content := range requiredContent {
		if !strings.Contains(block, content) {
			t.Errorf("GenerateAgentsBlock() missing required content: %q", content)
		}
	}
}

func TestComposeAgentsFile(t *testing.T) {
	version := small.ProtocolVersion

	tests := []struct {
		name            string
		existingContent string
		mode            AgentsMode
		wantErr         bool
		checkFunc       func(t *testing.T, result string)
	}{
		{
			name:            "empty content",
			existingContent: "",
			mode:            AgentsModeAppend,
			wantErr:         false,
			checkFunc: func(t *testing.T, result string) {
				if !strings.Contains(result, agentsBlockBegin) {
					t.Error("expected block in result")
				}
			},
		},
		{
			name:            "append to existing content",
			existingContent: "# My Custom AGENTS.md\n\nThis is my content.",
			mode:            AgentsModeAppend,
			wantErr:         false,
			checkFunc: func(t *testing.T, result string) {
				// Existing content should come first
				customIdx := strings.Index(result, "My Custom AGENTS.md")
				blockIdx := strings.Index(result, agentsBlockBegin)
				if customIdx >= blockIdx {
					t.Error("expected existing content before block in append mode")
				}
			},
		},
		{
			name:            "prepend to existing content",
			existingContent: "# My Custom AGENTS.md\n\nThis is my content.",
			mode:            AgentsModePrepend,
			wantErr:         false,
			checkFunc: func(t *testing.T, result string) {
				// Block should come first
				customIdx := strings.Index(result, "My Custom AGENTS.md")
				blockIdx := strings.Index(result, agentsBlockBegin)
				if blockIdx >= customIdx {
					t.Error("expected block before existing content in prepend mode")
				}
			},
		},
		{
			name: "replace existing block in append mode",
			existingContent: `# Header

<!-- BEGIN SMALL HARNESS v` + version + ` -->
Old content
<!-- END SMALL HARNESS v` + version + ` -->

# Footer`,
			mode:    AgentsModeAppend,
			wantErr: false,
			checkFunc: func(t *testing.T, result string) {
				// Should have exactly one block
				count := strings.Count(result, agentsBlockBegin)
				if count != 1 {
					t.Errorf("expected 1 block, got %d", count)
				}
				// Should preserve header and footer
				if !strings.Contains(result, "# Header") {
					t.Error("expected header to be preserved")
				}
				if !strings.Contains(result, "# Footer") {
					t.Error("expected footer to be preserved")
				}
				// Should not contain old content
				if strings.Contains(result, "Old content") {
					t.Error("expected old content to be replaced")
				}
			},
		},
		{
			name: "replace existing block in prepend mode",
			existingContent: `# Header

<!-- BEGIN SMALL HARNESS v` + version + ` -->
Old content
<!-- END SMALL HARNESS v` + version + ` -->

# Footer`,
			mode:    AgentsModePrepend,
			wantErr: false,
			checkFunc: func(t *testing.T, result string) {
				// Should have exactly one block
				count := strings.Count(result, agentsBlockBegin)
				if count != 1 {
					t.Errorf("expected 1 block, got %d", count)
				}
				// Should preserve surrounding content
				if !strings.Contains(result, "# Header") {
					t.Error("expected header to be preserved")
				}
				if !strings.Contains(result, "# Footer") {
					t.Error("expected footer to be preserved")
				}
			},
		},
		{
			name: "malformed block returns error",
			existingContent: `<!-- BEGIN SMALL HARNESS v` + version + ` -->
Content without end`,
			mode:    AgentsModeAppend,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ComposeAgentsFile(tt.existingContent, tt.mode)
			if (err != nil) != tt.wantErr {
				t.Errorf("ComposeAgentsFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && tt.checkFunc != nil {
				tt.checkFunc(t, result)
			}
		})
	}
}

func TestValidateAgentsModeFlags(t *testing.T) {
	tests := []struct {
		name            string
		agentsMode      AgentsMode
		noAgents        bool
		overwriteAgents bool
		wantErr         bool
	}{
		{"no flags", AgentsModeNone, false, false, false},
		{"only agents-mode", AgentsModeAppend, false, false, false},
		{"only no-agents", AgentsModeNone, true, false, false},
		{"only overwrite-agents", AgentsModeNone, false, true, false},
		{"agents-mode and no-agents", AgentsModeAppend, true, false, true},
		{"agents-mode and overwrite-agents", AgentsModeAppend, false, true, true},
		{"no-agents and overwrite-agents", AgentsModeNone, true, true, true},
		{"all three", AgentsModeAppend, true, true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateAgentsModeFlags(tt.agentsMode, tt.noAgents, tt.overwriteAgents)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateAgentsModeFlags() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAgentsFileExistsMessage(t *testing.T) {
	msg := AgentsFileExistsMessage()

	requiredContent := []string{
		"AGENTS.md already exists",
		"--overwrite-agents",
		"--agents-mode=append",
		"--agents-mode=prepend",
	}

	for _, content := range requiredContent {
		if !strings.Contains(msg, content) {
			t.Errorf("AgentsFileExistsMessage() missing: %q", content)
		}
	}
}
