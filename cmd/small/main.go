package main

import (
	"fmt"
	"os"

	"github.com/agentlegible/small-cli/internal/commands"
)

const (
	Version = "0.1.0"
)

func main() {
	if err := commands.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
