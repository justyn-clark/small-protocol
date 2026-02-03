package commands

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var (
	baseDir       string
	outputNoColor bool
	outputQuiet   bool
	errFlagError  = errors.New("flag error")
)

func init() {
	wd, err := os.Getwd()
	if err != nil {
		wd = "."
	}
	baseDir = wd
}

func Execute() error {
	printBannerIfEligible(os.Args)

	rootCmd := &cobra.Command{
		Use:   "small",
		Short: "SMALL protocol CLI tool",
		Long:  "SMALL is a protocol for durable, agent-legible project state.",
	}

	rootCmd.SilenceUsage = true
	rootCmd.SilenceErrors = true

	configureRootOutput(rootCmd)

	rootCmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		configurePrinter(outputNoColor, outputQuiet)
	}

	rootCmd.PersistentFlags().BoolVar(&outputNoColor, "no-color", false, "Disable ANSI color output")
	rootCmd.PersistentFlags().BoolVar(&outputQuiet, "quiet", false, "Suppress non-error output")

	rootCmd.AddCommand(versionCmd())
	rootCmd.AddCommand(initCmd())
	rootCmd.AddCommand(validateCmd())
	rootCmd.AddCommand(lintCmd())
	rootCmd.AddCommand(handoffCmd())
	rootCmd.AddCommand(startCmd())
	rootCmd.AddCommand(draftCmd())
	rootCmd.AddCommand(acceptCmd())
	rootCmd.AddCommand(fixCmd())
	rootCmd.AddCommand(planCmd())
	rootCmd.AddCommand(statusCmd())
	rootCmd.AddCommand(applyCmd())
	rootCmd.AddCommand(resetCmd())
	rootCmd.AddCommand(progressCmd())
	rootCmd.AddCommand(checkpointCmd())
	rootCmd.AddCommand(checkCmd())
	rootCmd.AddCommand(emitCmd())
	rootCmd.AddCommand(verifyCmd())
	rootCmd.AddCommand(doctorCmd())
	rootCmd.AddCommand(selftestCmd())
	rootCmd.AddCommand(archiveCmd())
	rootCmd.AddCommand(runCmd())
	rootCmd.AddCommand(agentsCmd())

	err := rootCmd.Execute()
	if errors.Is(err, errFlagError) {
		return nil
	}
	return err
}

func configureRootOutput(rootCmd *cobra.Command) {
	rootCmd.SetOut(os.Stdout)
	rootCmd.SetErr(os.Stderr)

	defaultHelp := rootCmd.HelpFunc()
	defaultUsage := rootCmd.UsageFunc()

	rootCmd.SetFlagErrorFunc(func(cmd *cobra.Command, err error) error {
		p := currentPrinter()
		usage, _ := captureCommandOutput(cmd, defaultUsage)
		lines := []string{err.Error()}
		if usage != "" {
			lines = append(lines, "Usage:")
			lines = append(lines, strings.Split(usage, "\n")...)
		}
		if p != nil {
			p.PrintError(p.FormatErrorBlock("Error", lines))
		}
		return errFlagError
	})

	rootCmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		p := currentPrinter()
		if p == nil || outputQuiet {
			return
		}
		output := captureHelpOutput(cmd, defaultHelp, args)
		if output != "" {
			p.PrintInfo(output)
		}
	})

	rootCmd.SetUsageFunc(func(cmd *cobra.Command) error {
		p := currentPrinter()
		if p == nil || outputQuiet {
			return nil
		}
		output, err := captureCommandOutput(cmd, defaultUsage)
		if output != "" {
			p.PrintInfo(output)
		}
		return err
	})
}

func captureHelpOutput(cmd *cobra.Command, help func(*cobra.Command, []string), args []string) string {
	var buf bytes.Buffer
	oldOut := cmd.OutOrStdout()
	oldErr := cmd.ErrOrStderr()
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	help(cmd, args)
	cmd.SetOut(oldOut)
	cmd.SetErr(oldErr)
	return strings.TrimRight(buf.String(), "\n")
}

func captureCommandOutput(cmd *cobra.Command, fn func(*cobra.Command) error) (string, error) {
	var buf bytes.Buffer
	oldOut := cmd.OutOrStdout()
	oldErr := cmd.ErrOrStderr()
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	err := fn(cmd)
	cmd.SetOut(oldOut)
	cmd.SetErr(oldErr)
	return strings.TrimRight(buf.String(), "\n"), err
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
