package commands

import (
	"os"
	"strings"

	"golang.org/x/term"
)

const bannerA = `  ██████  ███    ███  █████  ██      ██
 ██       ████  ████ ██   ██ ██      ██
  █████   ██ ████ ██ ███████ ██      ██
       ██ ██  ██  ██ ██   ██ ██      ██
 ██████  ██      ██ ██   ██ ███████ ███████

 Deterministic execution with auditable lineage.
`

const bannerB = `SMALL
Schema - Manifest - Artifact - Lineage - Lifecycle

Durable, agent-legible project state.
`

func shouldPrintBanner(args []string) bool {
	// Only when invoked as `small` with no args.
	if len(args) != 1 {
		return false
	}

	// Opt-out knobs.
	if os.Getenv("SMALL_NO_BANNER") == "1" {
		return false
	}
	if strings.EqualFold(os.Getenv("CI"), "true") || os.Getenv("CI") == "1" {
		return false
	}

	// Only if stdout is a TTY.
	return term.IsTerminal(int(os.Stdout.Fd()))
}

func printBannerIfEligible(args []string) {
	if !shouldPrintBanner(args) {
		return
	}

	// Pick which banner to show (default A).
	style := strings.ToLower(strings.TrimSpace(os.Getenv("SMALL_BANNER")))
	switch style {
	case "b", "minimal":
		_, _ = os.Stdout.WriteString(bannerB + "\n")
	default:
		_, _ = os.Stdout.WriteString(bannerA + "\n")
	}
}
