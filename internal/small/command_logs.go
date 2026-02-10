package small

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const DefaultCommandSummaryCap = 200

func SummarizeCommand(command string, cap int) string {
	trimmed := strings.TrimSpace(command)
	if trimmed == "" {
		return ""
	}
	normalized := strings.Join(strings.Fields(trimmed), " ")
	if cap <= 0 {
		cap = DefaultCommandSummaryCap
	}
	if len(normalized) <= cap {
		return normalized
	}
	if cap <= 3 {
		return normalized[:cap]
	}
	return normalized[:cap-3] + "..."
}

func SanitizeTimestampForFilename(timestamp string) (string, error) {
	parsed, err := ParseProgressTimestamp(timestamp)
	if err != nil {
		return "", err
	}
	return parsed.UTC().Format("20060102T150405.000000000Z"), nil
}

func EnsureCommandLogDir(baseDir, replayId string) (string, error) {
	if strings.TrimSpace(replayId) == "" {
		return "", fmt.Errorf("replayId is required")
	}
	dir := CacheCommandLogsDir(baseDir, replayId)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return dir, nil
}

func WriteCommandLog(baseDir, replayId, timestamp, command string) (string, string, error) {
	if strings.TrimSpace(command) == "" {
		return "", "", fmt.Errorf("command is required")
	}
	if strings.TrimSpace(timestamp) == "" {
		return "", "", fmt.Errorf("timestamp is required")
	}
	sanitized, err := SanitizeTimestampForFilename(timestamp)
	if err != nil {
		return "", "", err
	}
	dir, err := EnsureCommandLogDir(baseDir, replayId)
	if err != nil {
		return "", "", err
	}
	filename := sanitized + ".txt"
	path := filepath.Join(dir, filename)
	if err := os.WriteFile(path, []byte(command), 0o644); err != nil {
		return "", "", err
	}
	sha := sha256.Sum256([]byte(command))
	refPath := filepath.ToSlash(filepath.Join(CacheDirName, "logs", replayId, "commands", filename))
	return refPath, hex.EncodeToString(sha[:]), nil
}
