package commands

import (
	"os"
	"testing"
)

func TestShouldPrintBanner_RootOnly(t *testing.T) {
	t.Setenv("SMALL_NO_BANNER", "")
	t.Setenv("CI", "")

	// Cannot reliably assert TTY in tests, so we only assert the root-args gate.
	if shouldPrintBanner([]string{"small", "version"}) {
		t.Fatal("expected banner disabled for non-root invocation")
	}
}

func TestShouldPrintBanner_RespectsEnvOptOut(t *testing.T) {
	t.Setenv("SMALL_NO_BANNER", "1")
	if shouldPrintBanner([]string{"small"}) {
		t.Fatal("expected banner disabled when SMALL_NO_BANNER=1")
	}
}

func TestShouldPrintBanner_RespectsCI(t *testing.T) {
	t.Setenv("SMALL_NO_BANNER", "")
	t.Setenv("CI", "true")
	if shouldPrintBanner([]string{"small"}) {
		t.Fatal("expected banner disabled in CI")
	}
}

func TestPrintBannerIfEligible_DoesNotCrash(t *testing.T) {
	// Just ensure no panic when called.
	t.Setenv("SMALL_NO_BANNER", "1")
	printBannerIfEligible([]string{"small"})
	_ = os.Stdout
}
