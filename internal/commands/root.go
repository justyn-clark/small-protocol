package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var (
	baseDir string
)

func init() {
	wd, err := os.Getwd()
	if err != nil {
		wd = "."
	}
	baseDir = wd
}

func Execute() error {
	rootCmd := &cobra.Command{
		Use:   "small",
		Short: "SMALL protocol CLI tool",
		Long:  "SMALL is a protocol for durable, agent-legible project state.",
	}

	rootCmd.AddCommand(versionCmd())
	rootCmd.AddCommand(initCmd())
	rootCmd.AddCommand(validateCmd())
	rootCmd.AddCommand(lintCmd())
	rootCmd.AddCommand(handoffCmd())
	rootCmd.AddCommand(planCmd())
	rootCmd.AddCommand(statusCmd())
	rootCmd.AddCommand(applyCmd())

	return rootCmd.Execute()
}

// resolveArtifactsDir resolves the directory that contains .small/
// If dir ends with .small, returns the parent directory
// If dir/.small exists, returns dir
// Otherwise, returns dir (for init/create scenarios)
func resolveArtifactsDir(dir string) string {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		absDir = dir
	}

	// If dir ends with .small, use parent
	if filepath.Base(absDir) == ".small" {
		return filepath.Dir(absDir)
	}

	// If dir/.small exists, use dir
	smallPath := filepath.Join(absDir, ".small")
	if _, err := os.Stat(smallPath); err == nil {
		return absDir
	}

	// Otherwise, return dir as-is (for init/create)
	return absDir
}

func findRepoRoot(startDir string) (string, error) {
	dir := startDir
	for {
		// Try v1.0.0 first, then fall back to v0.1 for backwards compatibility
		specPath := filepath.Join(dir, "spec", "small", "v1.0.0", "schemas")
		if _, err := os.Stat(specPath); err == nil {
			return dir, nil
		}
		// Fallback to v0.1 schemas during transition
		specPathOld := filepath.Join(dir, "spec", "small", "v0.1", "schemas")
		if _, err := os.Stat(specPathOld); err == nil {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("not in SMALL repo: spec/small schemas not found")
		}
		dir = parent
	}
}
