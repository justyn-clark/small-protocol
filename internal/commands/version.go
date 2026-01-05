package commands

import (
	"fmt"

	"github.com/justyn-clark/small-protocol/internal/small"
	"github.com/spf13/cobra"
)

const (
	CLIVersion = "1.0.0"
)

func versionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print CLI version and supported spec versions",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("small v%s\n", CLIVersion)
			fmt.Printf("Supported spec versions: [\"%s\"]\n", small.ProtocolVersion)
		},
	}
}
