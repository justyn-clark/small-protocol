package commands

import (
	"fmt"

	"github.com/justyn-clark/small-protocol/internal/small"
	"github.com/justyn-clark/small-protocol/internal/version"
	"github.com/spf13/cobra"
)

func versionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print CLI version and supported spec versions",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("small %s\n", version.GetVersion())
			fmt.Printf("Supported spec versions: [\"%s\"]\n", small.ProtocolVersion)
		},
	}
}
