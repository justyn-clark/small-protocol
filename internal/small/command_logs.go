package small

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

const DefaultCommandSummaryCap = 200

// SummarizeCommand is a byte-bounded display, never a substitute for command proof.
// Keep a URL-bearing whitespace token intact or omit it, including its quotes.
func SummarizeCommand(command string, cap int) string {
	normalized := normalizeCommand(command)
	if cap <= 0 {
		cap = DefaultCommandSummaryCap
	}
	if len(normalized) <= cap {
		return normalized
	}
	if cap <= 3 {
		return strings.Repeat(".", cap)
	}
	cut := cap - 3
	offset := 0
	for _, token := range strings.Fields(normalized) {
		end := offset + len(token)
		lower := strings.ToLower(token)
		if offset <= cut && end >= cut && (strings.Contains(lower, "http://") || strings.Contains(lower, "https://")) {
			cut = offset
			break
		}
		offset = end + 1
	}
	for cut > 0 && !utf8.RuneStart(normalized[cut]) {
		cut--
	}
	return normalized[:cut] + "..."
}

func normalizeCommand(command string) string {
	return strings.Join(strings.Fields(strings.ToValidUTF8(command, "\uFFFD")), " ")
}

// CommandNeedsProof marks new bounded captures; old valid receipts remain portable.
func CommandNeedsProof(command string) bool {
	return len(normalizeCommand(command)) > DefaultCommandSummaryCap
}

// legacyCommandSummary reproduces the pre-correction bytes only for verification.
func legacyCommandSummary(command string) string {
	normalized := strings.Join(strings.Fields(strings.TrimSpace(command)), " ")
	if len(normalized) <= DefaultCommandSummaryCap {
		return normalized
	}
	return normalized[:DefaultCommandSummaryCap-3] + "..."
}

// WriteContentCommandLog is the v2 content-addressed local command store.
func WriteContentCommandLog(baseDir, command string) (string, string, error) {
	sha := sha256.Sum256([]byte(command))
	digest := hex.EncodeToString(sha[:])
	ref := filepath.ToSlash(filepath.Join(CacheDirName, "commands", digest+".txt"))
	path := filepath.Join(baseDir, filepath.FromSlash(ref))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", "", err
	}
	if err := os.WriteFile(path, []byte(command), 0o600); err != nil {
		return "", "", err
	}
	return ref, digest, nil
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
