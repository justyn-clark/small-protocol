package small

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

var canonicalSmallRootFiles = map[string]struct{}{
	"intent.small.yml":      {},
	"constraints.small.yml": {},
	"plan.small.yml":        {},
	"progress.small.yml":    {},
	"handoff.small.yml":     {},
	"workspace.small.yml":   {},
}

func StrictSmallLayoutViolations(baseDir, commandHint string) ([]InvariantViolation, error) {
	smallDir := filepath.Join(baseDir, SmallDir)
	entries, err := os.ReadDir(smallDir)
	if err != nil {
		return nil, fmt.Errorf("failed to enumerate %s: %w", smallDir, err)
	}

	var unexpected []string
	for _, entry := range entries {
		name := entry.Name()
		relPath := filepath.ToSlash(filepath.Join(SmallDir, name))
		if entry.IsDir() {
			unexpected = append(unexpected, relPath+"/")
			continue
		}
		if _, ok := canonicalSmallRootFiles[name]; !ok {
			unexpected = append(unexpected, relPath)
		}
	}

	if len(unexpected) == 0 {
		return nil, nil
	}

	sort.Strings(unexpected)
	message := fmt.Sprintf(
		"strict invariant S4 failed: unexpected paths under .small/: %s. Delete or move these paths outside .small/.",
		strings.Join(unexpected, ", "),
	)
	if strings.TrimSpace(commandHint) != "" {
		message = message + fmt.Sprintf(" Command: %s", strings.TrimSpace(commandHint))
	}

	return []InvariantViolation{{
		File:    smallDir,
		Message: message,
	}}, nil
}
