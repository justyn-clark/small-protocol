package commands

import (
	"os"
	"strings"
)

func currentCommandHint() string {
	if len(os.Args) == 0 {
		return ""
	}
	value := strings.TrimSpace(strings.Join(os.Args, " "))
	if value == "" {
		return ""
	}
	return value
}
