package version

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

// GetVersion returns the version string with "v" prefix for display.
func GetVersion() string {
	if Version == "dev" {
		return "dev"
	}
	// GoReleaser provides version without "v" prefix, add it for display
	if len(Version) > 0 && Version[0] != 'v' {
		return "v" + Version
	}
	return Version
}
