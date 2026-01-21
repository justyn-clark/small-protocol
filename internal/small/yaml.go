package small

import (
	"fmt"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

var smallVersionLinePattern = regexp.MustCompile(`^(\s*small_version:\s*)(.*)$`)

// MarshalYAMLWithQuotedVersion encodes YAML while forcing small_version to be a quoted string.
func MarshalYAMLWithQuotedVersion(value interface{}) ([]byte, error) {
	data, err := yaml.Marshal(value)
	if err != nil {
		return nil, err
	}
	return normalizeSmallVersionYAML(data), nil
}

func normalizeSmallVersionYAML(data []byte) []byte {
	content := string(data)
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		matches := smallVersionLinePattern.FindStringSubmatch(line)
		if len(matches) == 0 {
			continue
		}
		prefix := matches[1]
		value := strings.TrimSpace(matches[2])
		if value == "" {
			break
		}
		value = strings.TrimSpace(strings.Trim(value, `"'`))
		lines[i] = prefix + fmt.Sprintf("%q", value)
		result := strings.Join(lines, "\n")
		if strings.HasSuffix(content, "\n") && !strings.HasSuffix(result, "\n") {
			result += "\n"
		}
		return []byte(result)
	}
	return data
}
