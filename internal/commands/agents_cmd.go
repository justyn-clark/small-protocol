package commands

import (
	"github.com/justyn-clark/small-protocol/internal/small"
	"github.com/spf13/cobra"
)

func agentsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "agents",
		Short: "Manage AGENTS.md harness block",
		Long: `Standalone commands for managing the SMALL harness block in AGENTS.md.

These commands NEVER touch the .small/ directory. They only operate on
AGENTS.md (or another file specified with --file).

The SMALL harness block is a bounded section delimited by:
  <!-- BEGIN SMALL HARNESS v` + small.ProtocolVersion + ` -->
  ... protocol rules for AI agents ...
  <!-- END SMALL HARNESS v` + small.ProtocolVersion + ` -->

Commands:
  apply   Write or update the SMALL harness block in AGENTS.md
  check   Validate AGENTS.md harness block (read-only)
  print   Print the canonical SMALL harness block to stdout

Examples:
  small agents apply                        # Create AGENTS.md if missing
  small agents apply --agents-mode=append   # Add block to existing file
  small agents apply --dry-run              # Preview changes without writing
  small agents check --strict               # Fail if file exists without block
  small agents print                        # Print canonical block`,
	}

	cmd.AddCommand(agentsApplyCmd())
	cmd.AddCommand(agentsCheckCmd())
	cmd.AddCommand(agentsPrintCmd())

	return cmd
}
