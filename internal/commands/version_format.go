package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/justyn-clark/small-protocol/internal/small"
)

var smallVersionLinePattern = regexp.MustCompile(`^(\s*small_version:\s*)(.*)$`)

type versionFormatResult struct {
	File      string
	Canonical bool
	Found     bool
}

func findVersionFormatWarnings(artifactsDir string) ([]string, error) {
	files := versionFormatFiles()
	var warnings []string
	for _, filename := range files {
		path := filepath.Join(artifactsDir, small.SmallDir, filename)
		data, err := os.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("failed to read %s: %w", path, err)
		}
		result := analyzeSmallVersionFormatting(string(data))
		if !result.Found {
			continue
		}
		if !result.Canonical {
			warnings = append(warnings, filepath.Join(small.SmallDir, filename))
		}
	}
	return warnings, nil
}

func analyzeSmallVersionFormatting(content string) versionFormatResult {
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		matches := smallVersionLinePattern.FindStringSubmatch(line)
		if len(matches) == 0 {
			continue
		}
		value, _, _ := splitValueAndComment(matches[2])
		value = strings.TrimSpace(value)
		if value == "" {
			return versionFormatResult{Found: true, Canonical: false}
		}
		if isDoubleQuoted(value) {
			return versionFormatResult{Found: true, Canonical: true}
		}
		return versionFormatResult{Found: true, Canonical: false}
	}
	return versionFormatResult{Found: false, Canonical: true}
}

func versionFormatFiles() []string {
	files := make([]string, 0, len(small.CanonicalFiles)+1)
	files = append(files, small.CanonicalFiles...)
	files = append(files, "workspace.small.yml")
	return files
}

func normalizeSmallVersionLine(line string) (string, bool) {
	matches := smallVersionLinePattern.FindStringSubmatch(line)
	if len(matches) == 0 {
		return line, false
	}
	prefix := matches[1]
	valuePart, comment, hasComment := splitValueAndComment(matches[2])
	normalizedValue, canonical := normalizeSmallVersionValue(valuePart)
	if canonical && !hasComment {
		normalizedLine := prefix + normalizedValue
		return normalizedLine, normalizedLine != line
	}
	normalizedLine := prefix + normalizedValue
	if hasComment {
		normalizedLine += comment
	}
	return normalizedLine, normalizedLine != line
}

func normalizeSmallVersionValue(value string) (string, bool) {
	raw := strings.TrimSpace(value)
	if raw == "" {
		return `""`, false
	}
	if isDoubleQuoted(raw) {
		return raw, true
	}
	if isSingleQuoted(raw) {
		raw = strings.Trim(raw, `'`)
	} else {
		raw = strings.TrimSpace(raw)
	}
	return fmt.Sprintf("%q", raw), false
}

func splitValueAndComment(value string) (string, string, bool) {
	if idx := strings.Index(value, "#"); idx >= 0 {
		return value[:idx], value[idx:], true
	}
	return value, "", false
}

func isDoubleQuoted(value string) bool {
	return len(value) >= 2 && strings.HasPrefix(value, `"`) && strings.HasSuffix(value, `"`)
}

func isSingleQuoted(value string) bool {
	return len(value) >= 2 && strings.HasPrefix(value, `'`) && strings.HasSuffix(value, `'`)
}
