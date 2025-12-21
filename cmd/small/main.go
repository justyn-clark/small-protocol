package main

import (
	"fmt"
	"os"

	"github.com/justyn-clark/small-protocol/internal/commands"
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
