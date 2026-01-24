package commands

import (
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/term"
)

type Printer struct {
	out   io.Writer
	err   io.Writer
	color bool
	quiet bool
}

type ansiCode string

const (
	ansiReset  ansiCode = "\u001b[0m"
	ansiBold   ansiCode = "\u001b[1m"
	ansiRed    ansiCode = "\u001b[31m"
	ansiYellow ansiCode = "\u001b[33m"
	ansiGreen  ansiCode = "[32m"
	ansiCyan   ansiCode = "[36m"
)

var (
	globalPrinter *Printer
)

func NewPrinter(out, err io.Writer, color bool, quiet bool) *Printer {
	return &Printer{out: out, err: err, color: color, quiet: quiet}
}

func configurePrinter(noColor, quiet bool) {
	globalPrinter = NewPrinter(os.Stdout, os.Stderr, shouldUseColor(noColor), quiet)
}

func currentPrinter() *Printer {
	if globalPrinter == nil {
		configurePrinter(outputNoColor, outputQuiet)
		return globalPrinter
	}

	if outFile, ok := globalPrinter.out.(*os.File); ok && outFile != os.Stdout {
		configurePrinter(outputNoColor, outputQuiet)
		return globalPrinter
	}
	if errFile, ok := globalPrinter.err.(*os.File); ok && errFile != os.Stderr {
		configurePrinter(outputNoColor, outputQuiet)
		return globalPrinter
	}

	return globalPrinter
}

func shouldUseColor(noColor bool) bool {
	if noColor {
		return false
	}
	if !term.IsTerminal(int(os.Stdout.Fd())) {
		return false
	}
	if isCIEnv() {
		return false
	}
	return true
}

func isCIEnv() bool {
	return os.Getenv("CI") != ""
}

func isInteractiveOutput() bool {
	return term.IsTerminal(int(os.Stdout.Fd())) || term.IsTerminal(int(os.Stderr.Fd()))
}

func (p *Printer) PrintInfo(message string) {
	if p == nil || p.quiet {
		return
	}
	fmt.Fprintln(p.out, message)
}

func (p *Printer) PrintWarn(message string) {
	if p == nil || p.quiet {
		return
	}
	fmt.Fprintln(p.err, p.colorize(message, ansiYellow))
}

func (p *Printer) PrintError(message string) {
	if p == nil {
		return
	}
	fmt.Fprintln(p.err, p.colorize(message, ansiRed))
}

func (p *Printer) PrintSuccess(message string) {
	if p == nil || p.quiet {
		return
	}
	fmt.Fprintln(p.out, p.colorize(message, ansiGreen))
}

func (p *Printer) PrintLabel(label, message string) {
	if p == nil || p.quiet {
		return
	}
	renderedLabel := p.Label(label)
	if message == "" {
		fmt.Fprintln(p.out, renderedLabel)
		return
	}
	fmt.Fprintf(p.out, "%s %s\n", renderedLabel, message)
}

func (p *Printer) Label(s string) string {
	if p == nil || !p.color || s == "" {
		return s
	}
	return string(ansiCyan) + s + string(ansiReset)
}

func (p *Printer) FormatBlock(title string, lines []string) string {
	header := title
	if p != nil && p.color {
		header = string(ansiBold) + title + string(ansiReset)
	}

	var builder strings.Builder
	builder.WriteString(header)
	for _, line := range lines {
		builder.WriteString("\n")
		if line == "" {
			continue
		}
		builder.WriteString("  ")
		builder.WriteString(line)
	}
	return builder.String()
}

func (p *Printer) FormatErrorBlock(title string, lines []string) string {
	if title == "" {
		title = "Error"
	}
	return p.FormatBlock(title, lines)
}

func (p *Printer) colorize(message string, code ansiCode) string {
	if p == nil || !p.color || code == "" {
		return message
	}
	return string(code) + message + string(ansiReset)
}
