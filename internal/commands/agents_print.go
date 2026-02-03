package commands

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func agentsPrintCmd() *cobra.Command {
	var withMarkers bool
	var format string

	cmd := &cobra.Command{
		Use:   "print",
		Short: "Print canonical SMALL harness block",
		Long: `Print the canonical SMALL harness block to stdout.

This command is read-only and performs no file operations.
Useful for manual paste into other files or contexts.

Examples:
  small agents print                    # Full block with markers
  small agents print --with-markers=false   # Content only, no markers`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if withMarkers {
				fmt.Print(GenerateAgentsBlock())
			} else {
				// Print content without markers
				content := agentsBlockContent
				if format == "plain" {
					// Strip markdown formatting if requested
					content = stripMarkdown(content)
				}
				fmt.Print(content)
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&withMarkers, "with-markers", true, "Include BEGIN/END markers")
	cmd.Flags().StringVar(&format, "format", "md", "Output format (md, plain)")

	return cmd
}

// stripMarkdown removes basic markdown formatting for plain output.
func stripMarkdown(content string) string {
	lines := strings.Split(content, "\n")
	var result []string

	for _, line := range lines {
		// Remove heading markers
		line = strings.TrimLeft(line, "# ")
		// Remove bold markers
		line = strings.ReplaceAll(line, "**", "")
		// Remove code backticks
		line = strings.ReplaceAll(line, "`", "")
		// Keep table formatting as-is for readability
		result = append(result, line)
	}

	return strings.Join(result, "\n")
}
