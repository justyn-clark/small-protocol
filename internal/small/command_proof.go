package small

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const CommandProofDir = ".small-command-proofs"
const MaxCommandProofBytes = 1 << 20

var commandLogRefPattern = regexp.MustCompile(`^\.small-cache/logs/([a-f0-9]{64})/commands/([0-9]{8}T[0-9]{6}\.[0-9]{9}Z\.txt)$`)
var contentCommandRefPattern = regexp.MustCompile(`^\.small-cache/commands/([a-f0-9]{64})\.txt$`)
var commandDigestPattern = regexp.MustCompile(`^[a-f0-9]{64}$`)

// ReadCommandProof accepts only CLI local refs and a mirrored portable proof.
// A corrupt/unreadable cache never falls back to a different proof. os.Root
// contains symlink traversal; regular-file checks and bounded reads limit input.
func ReadCommandProof(baseDir, ref, digest string) (string, error) {
	if !commandDigestPattern.MatchString(digest) {
		return "", fmt.Errorf("command_sha256 must be 64 lowercase hex digits")
	}
	legacy := commandLogRefPattern.MatchString(ref)
	content := contentCommandRefPattern.FindStringSubmatch(ref)
	if !legacy && content == nil {
		return "", fmt.Errorf("unsupported command_ref: use a relative .small-cache/logs/<replayId>/commands/<timestamp>.txt or .small-cache/commands/<sha256>.txt ref")
	}
	if content != nil && content[1] != digest {
		return "", fmt.Errorf("command_ref digest does not match command_sha256")
	}
	root, err := os.OpenRoot(baseDir)
	if err != nil {
		return "", fmt.Errorf("open command proof workspace: %w", err)
	}
	defer root.Close()
	portable := CommandProofDir + strings.TrimPrefix(ref, CacheDirName)
	data, err := readCommandProofFile(root, ref)
	if os.IsNotExist(err) {
		data, err = readCommandProofFile(root, portable)
	}
	if err != nil {
		return "", fmt.Errorf("command proof unavailable for %s: %w; supply the exact non-secret command bytes at %s (see docs/command-proof.md); do not edit receipts", ref, err, portable)
	}
	sum := sha256.Sum256(data)
	if hex.EncodeToString(sum[:]) != digest {
		return "", fmt.Errorf("command proof SHA256 mismatch for %s; restore the exact original bytes, do not edit receipts", ref)
	}
	return string(data), nil
}

func readCommandProofFile(root *os.Root, path string) ([]byte, error) {
	info, err := root.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("command proof must be a regular non-symlink file: %s", path)
	}
	if info.Size() > MaxCommandProofBytes {
		return nil, fmt.Errorf("command proof exceeds %d-byte limit: %s", MaxCommandProofBytes, path)
	}
	file, err := root.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	info, err = file.Stat()
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("command proof must be a regular file: %s", path)
	}
	data, err := io.ReadAll(io.LimitReader(file, MaxCommandProofBytes+1))
	if err != nil {
		return nil, err
	}
	if len(data) > MaxCommandProofBytes {
		return nil, fmt.Errorf("command proof exceeds %d-byte limit: %s", MaxCommandProofBytes, path)
	}
	return data, nil
}

func summaryVersionTwo(value any) bool {
	switch v := value.(type) {
	case int:
		return v == 2
	case int64:
		return v == 2
	case float64:
		return v == 2
	case json.Number:
		return v.String() == "2"
	}
	return false
}

func insecureCommandSummary(summary string) bool {
	for _, url := range extractHTTPURLs(summary) {
		if !isAllowedLocalhostHTTP(url) {
			return true
		}
	}
	return false
}

// commandRecordForLint clones only the proven CLI display fields. Original
// artifact data, arbitrary notes/evidence and all other strings stay untouched.
func commandRecordForLint(baseDir string, record map[string]any) (map[string]any, bool, []string) {
	summary, _ := record["command_summary"].(string)
	version, marked := record["command_summary_version"]
	legacyCandidate := len(summary) == DefaultCommandSummaryCap && strings.HasSuffix(summary, "...") && insecureCommandSummary(summary)
	literalCandidate := commandHTTPLiteralsForLint(summary) != summary
	if !marked && !legacyCandidate && !literalCandidate {
		return record, false, nil
	}
	if marked && !summaryVersionTwo(version) {
		return record, false, []string{"unsupported command_summary_version; expected 2"}
	}
	ref, _ := record["command_ref"].(string)
	digest, _ := record["command_sha256"].(string)
	command, err := ReadCommandProof(baseDir, ref, digest)
	if err != nil {
		return record, false, []string{err.Error()}
	}
	if parts := commandLogRefPattern.FindStringSubmatch(ref); parts != nil {
		if replay, ok := record["replayId"].(string); ok && replay != parts[1] {
			return record, false, []string{"command_ref replayId does not match receipt replayId"}
		}
		if timestamp, ok := record["timestamp"].(string); ok {
			filename, err := SanitizeTimestampForFilename(timestamp)
			if err != nil || filename+".txt" != parts[2] {
				return record, false, []string{"command_ref timestamp does not match receipt timestamp"}
			}
		}
	}
	expected := legacyCommandSummary(command)
	if marked {
		expected = SummarizeCommand(command, DefaultCommandSummaryCap)
	}
	if summary != expected {
		return record, false, []string{"command_summary does not match the verified command summarizer output"}
	}
	if value, exists := record["command"]; exists && value != expected {
		return record, false, []string{"command does not match the verified command summary"}
	}
	copy := make(map[string]any, len(record))
	for key, value := range record {
		copy[key] = value
	}
	copy["command_summary"] = command
	if _, exists := record["command"]; exists {
		copy["command"] = command
	}
	if record["evidence"] == "Dry-run: no command executed" && record["status"] == "pending" && record["notes"] == fmt.Sprintf("apply --dry-run (cmd: %q)", expected) {
		copy["notes"] = "apply --dry-run (verified command display)"
	}
	return copy, true, nil
}

var commandSecretPattern = regexp.MustCompile(`(?i)(?:\b(?:api[_-]?key|access[_-]?token|password|passwd|secret|token)\s*=\s*["']?[^\s"'$;<>][^\s"';]*|\bbearer\s+[a-z0-9._~+/-]{8,}|\b(?:gh[pousr]_[a-z0-9]{20,}|github_pat_[a-z0-9_]{20,}|sk-[a-z0-9_-]{16,}|AKIA[A-Z0-9]{16})\b|-----BEGIN (?:RSA |EC |OPENSSH )?PRIVATE KEY-----)`)

func commandSecretViolations(path string, record map[string]any) []InvariantViolation {
	command, _ := record["command_summary"].(string)
	if command == "" {
		command, _ = record["command"].(string)
	}
	if commandSecretPattern.MatchString(command) {
		return []InvariantViolation{{File: path, Message: "potential secret detected in captured command; remove credentials from commands, never rewrite historical receipts"}}
	}
	return nil
}

// CheckCommandRecord provides the same strict command policy to v2 receipts.
func CheckCommandRecord(baseDir, path string, record map[string]any) []InvariantViolation {
	copy, verified, errors := commandRecordForLint(baseDir, record)
	var violations []InvariantViolation
	for _, err := range errors {
		violations = append(violations, InvariantViolation{File: path, Message: err})
	}
	artifact := &Artifact{Type: "progress", Path: path, Data: copy}
	if verified {
		artifact.verifiedCommandPaths = map[string]bool{"/command": true, "/command_summary": true}
	}
	violations = append(violations, checkSecrets(artifact)...)
	violations = append(violations, checkInsecureLinks(artifact)...)
	return append(violations, commandSecretViolations(path, copy)...)
}

func commandSecurityArtifact(artifact *Artifact) (*Artifact, []InvariantViolation) {
	if artifact.Type != "progress" || artifact.Data == nil {
		return artifact, nil
	}
	root := make(map[string]any, len(artifact.Data))
	for key, value := range artifact.Data {
		root[key] = value
	}
	entries, _ := root["entries"].([]any)
	copies := make([]any, len(entries))
	verifiedPaths := make(map[string]bool)
	var violations []InvariantViolation
	baseDir := filepath.Dir(filepath.Dir(artifact.Path))
	for i, entry := range entries {
		record, ok := entry.(map[string]any)
		if !ok {
			copies[i] = entry
			continue
		}
		copy, verified, errors := commandRecordForLint(baseDir, record)
		copies[i] = copy
		if verified {
			verifiedPaths[fmt.Sprintf("/entries/%d/command", i)] = true
			verifiedPaths[fmt.Sprintf("/entries/%d/command_summary", i)] = true
		}
		for _, err := range errors {
			violations = append(violations, InvariantViolation{File: artifact.Path, Message: fmt.Sprintf("entries[%d]: %s", i, err)})
		}
		violations = append(violations, commandSecretViolations(artifact.Path, copy)...)
	}
	root["entries"] = copies
	return &Artifact{Type: artifact.Type, Path: artifact.Path, Data: root, verifiedCommandPaths: verifiedPaths}, violations
}
