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

	return rootCmd.Execute()
}

func findRepoRoot() (string, error) {
	dir := baseDir
	for {
		specPath := filepath.Join(dir, "spec", "small", "v0.1", "schemas")
		if _, err := os.Stat(specPath); err == nil {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("not in SMALL repo: spec/small/v0.1/schemas not found")
		}
		dir = parent
	}
}
