package version

import (
	"runtime/debug"
	"strings"
)

// Build-time variables injected via ldflags.
// These are set by GoReleaser during tagged builds.
var (
	// Version is the semantic version (e.g., "1.0.0" or "v1.0.0").
	Version = "dev"
	// Commit is the git commit SHA.
	Commit = "none"
	// Date is the build date in RFC3339 format.
	Date = "unknown"
)

func resolvedVersion() string {
	v := strings.TrimSpace(Version)
	if v != "" && v != "dev" {
		return normalizeVersion(v)
	}

	info, ok := debug.ReadBuildInfo()
	if !ok || info == nil {
		return "dev"
	}

	moduleVersion := strings.TrimSpace(info.Main.Version)
	switch moduleVersion {
	case "", "(devel)", "devel", "dev":
		return "dev"
	default:
		return normalizeVersion(moduleVersion)
	}
}

func normalizeVersion(v string) string {
	if v == "" {
		return "dev"
	}
	if v[0] == 'v' || v[0] == 'V' {
		return "v" + v[1:]
	}
	return "v" + v
}

// GetVersion returns the version string with "v" prefix for display.
func GetVersion() string {
	return resolvedVersion()
}
