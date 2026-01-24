package commands

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/justyn-clark/small-protocol/internal/updatecheck"
	"github.com/justyn-clark/small-protocol/internal/version"
)

func maybePrintUpdateNotice(p *Printer, jsonOutput bool) {
	if p == nil || jsonOutput || outputQuiet {
		return
	}
	if isCIEnv() {
		return
	}
	if !isInteractiveOutput() {
		return
	}
	if strings.TrimSpace(os.Getenv("SMALL_NO_UPDATE_CHECK")) == "1" {
		return
	}

	current := version.GetVersion()
	notice, err := updatecheck.Check(context.Background(), updatecheck.Options{
		CurrentVersion: current,
		UpdateURL:      os.Getenv("SMALL_UPDATE_URL"),
	})
	if err != nil || notice == nil {
		return
	}
	p.PrintWarn(fmt.Sprintf("Update available: %s (current %s).", notice.Latest, notice.Current))
}
