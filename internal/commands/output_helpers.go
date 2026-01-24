package commands

import (
	"fmt"
	"strings"
)

const defaultListCap = 5

func formatListWithCap(items []string, cap int) string {
	if len(items) == 0 {
		return "none"
	}
	if cap <= 0 || len(items) <= cap {
		return strings.Join(items, ", ")
	}
	return fmt.Sprintf("%s, and %d more", strings.Join(items[:cap], ", "), len(items)-cap)
}

func formatCountedList(label string, items []string, cap int) string {
	return fmt.Sprintf("%s (%d): %s", label, len(items), formatListWithCap(items, cap))
}

func formatBulletList(items []string) []string {
	lines := make([]string, 0, len(items))
	for _, item := range items {
		lines = append(lines, fmt.Sprintf("- %s", item))
	}
	return lines
}
