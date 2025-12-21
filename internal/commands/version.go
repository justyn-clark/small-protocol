package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

const (
	CLIVersion     = "0.1.0"
	SupportedSpecs = "[\"0.1\"]"
)

func versionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print CLI version and supported spec versions",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("small v%s\n", CLIVersion)
			fmt.Printf("Supported spec versions: %s\n", SupportedSpecs)
		},
	}
}
