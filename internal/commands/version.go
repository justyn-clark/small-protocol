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
			p := currentPrinter()

			if outputQuiet {
				return
			}

			p.PrintInfo(fmt.Sprintf("small %s", version.GetVersion()))
			p.PrintInfo(fmt.Sprintf("Supported spec versions: [\"%s\"]", small.ProtocolVersion))
			maybePrintUpdateNotice(p, false)
		},
	}
}
