package small

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"unicode/utf8"
)

func proofFixture(t *testing.T, command string, modern bool) (string, map[string]any) {
	t.Helper()
	base := t.TempDir()
	replay := strings.Repeat("a", 64)
	timestamp := "2026-10-08T04:32:03.067052000Z"
	ref, digest, err := WriteCommandLog(base, replay, timestamp, command)
	if err != nil {
		t.Fatal(err)
	}
	summary := legacyCommandSummary(command)
	record := map[string]any{"timestamp": timestamp, "replayId": replay, "command_ref": ref, "command_sha256": digest, "status": "in_progress", "evidence": "Apply started"}
	if modern {
		summary = SummarizeCommand(command, DefaultCommandSummaryCap)
		record["command_summary_version"] = 2
	}
	record["command"] = summary
	record["command_summary"] = summary
	return base, record
}

func boundaryCommand() string {
	return "printf '%s\\n' '" + strings.Repeat("0", 169) + " http://127.0.0.1:4345/'"
}

func TestCommandProofLegacyPreservesRecordAndFiles(t *testing.T) {
	command := boundaryCommand()
	base, record := proofFixture(t, command, false)
	before := make(map[string]any)
	for k, v := range record {
		before[k] = v
	}
	ref := record["command_ref"].(string)
	if !strings.Contains(record["command_summary"].(string), "http://127.0...") {
		t.Fatal("fixture does not reproduce legacy cut")
	}
	if violations := CheckCommandRecord(base, "progress", record); len(violations) != 0 {
		t.Fatalf("violations: %v", violations)
	}
	if !reflect.DeepEqual(record, before) {
		t.Fatal("lint mutated receipt")
	}
	data, err := os.ReadFile(filepath.Join(base, ref))
	if err != nil || string(data) != command {
		t.Fatalf("command bytes changed: %v", err)
	}
	artifact := &Artifact{Type: "progress", Path: filepath.Join(base, ".small", "progress.small.yml"), Data: map[string]any{"entries": []any{record}}}
	copy, violations := commandSecurityArtifact(artifact)
	if len(violations) != 0 || len(checkInsecureLinks(copy)) != 0 {
		t.Fatalf("progress policy: %v", violations)
	}
	if !reflect.DeepEqual(record, before) {
		t.Fatal("progress policy mutated receipt")
	}
}

func TestCommandProofPortableFreshWorkspace(t *testing.T) {
	base, record := proofFixture(t, boundaryCommand(), false)
	fresh := t.TempDir()
	ref := record["command_ref"].(string)
	if violations := CheckCommandRecord(fresh, "progress", record); !proofMessagesContain(violations, "proof unavailable") {
		t.Fatalf("missing proof did not fail closed: %v", violations)
	}
	portable := filepath.Join(fresh, CommandProofDir+strings.TrimPrefix(ref, CacheDirName))
	if err := os.MkdirAll(filepath.Dir(portable), 0o755); err != nil {
		t.Fatal(err)
	}
	original, err := os.ReadFile(filepath.Join(base, ref))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(portable, original, 0o600); err != nil {
		t.Fatal(err)
	}
	if violations := CheckCommandRecord(fresh, "progress", record); len(violations) != 0 {
		t.Fatalf("portable proof failed: %v", violations)
	}
	if err := os.WriteFile(portable, append(original, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
	if violations := CheckCommandRecord(fresh, "progress", record); !proofMessagesContain(violations, "SHA256 mismatch") {
		t.Fatalf("tampered bytes passed: %v", violations)
	}
}

func TestCommandProofFullCommandEnforcement(t *testing.T) {
	for _, modern := range []bool{false, true} {
		for _, test := range []struct{ name, suffix, message string }{
			{"external after cap", " http://external.example/path", "insecure link"},
			{"secret after cap", " API_KEY=sk-abcdefghijklmnopqrstuvwxyz0123456789", "potential secret"},
			{"private key after cap", " -----BEGIN PRIVATE KEY-----", "potential secret"},
			{"bearer after cap", " Authorization: Bearer abcdefghijklmnopqrstuvwxyz", "potential secret"},
		} {
			t.Run(fmt.Sprintf("%t/%s", modern, test.name), func(t *testing.T) {
				base, record := proofFixture(t, boundaryCommand()+test.suffix, modern)
				violations := CheckCommandRecord(base, "progress", record)
				if !proofMessagesContain(violations, test.message) {
					t.Fatalf("hidden violation passed: %v", violations)
				}
				for _, v := range violations {
					if strings.Contains(v.Message, "sk-abcdefghijklmnopqrstuvwxyz") {
						t.Fatal("diagnostic exposed credential")
					}
				}
			})
		}
	}
	// The marker requires proof even when the summary contains no URL at all.
	base, record := proofFixture(t, "printf '"+strings.Repeat("x", 230)+"' # http://external.example", true)
	if v := CheckCommandRecord(base, "progress", record); !proofMessagesContain(v, "insecure link") {
		t.Fatalf("omitted external URL passed: %v", v)
	}
}

func TestCommandProofDoesNotExemptAuthoredStrings(t *testing.T) {
	for _, field := range []string{"notes", "evidence", "link", "test", "verification"} {
		t.Run(field, func(t *testing.T) {
			base, record := proofFixture(t, boundaryCommand(), false)
			record[field] = "http://12..."
			if v := CheckCommandRecord(base, "progress", record); !proofMessagesContain(v, "insecure link") {
				t.Fatalf("authored %s exempted: %v", field, v)
			}
		})
	}
	base, record := proofFixture(t, boundaryCommand(), false)
	record["status"] = "pending"
	record["evidence"] = "Dry-run: no command executed"
	record["notes"] = fmt.Sprintf("apply --dry-run (cmd: %q)", record["command_summary"])
	if v := CheckCommandRecord(base, "progress", record); len(v) != 0 {
		t.Fatalf("exact generated wrapper failed: %v", v)
	}
	record["notes"] = record["notes"].(string) + " http://external.example"
	if v := CheckCommandRecord(base, "progress", record); !proofMessagesContain(v, "insecure link") {
		t.Fatalf("modified wrapper exempted: %v", v)
	}
}

func TestCommandProofForgeryAndMetadata(t *testing.T) {
	for _, field := range []string{"command", "command_summary", "command_sha256", "command_ref", "timestamp", "replayId", "command_summary_version"} {
		t.Run(field, func(t *testing.T) {
			base, record := proofFixture(t, boundaryCommand(), true)
			switch field {
			case "command", "command_summary":
				record[field] = "forged display"
			case "command_sha256", "replayId":
				record[field] = strings.Repeat("b", 64)
			case "command_ref":
				record[field] = "/tmp/substituted.txt"
			case "timestamp":
				record[field] = "2026-10-08T04:32:04.067052000Z"
			case "command_summary_version":
				record[field] = 3
			}
			if v := CheckCommandRecord(base, "progress", record); len(v) == 0 {
				t.Fatalf("forged %s passed", field)
			}
		})
	}
	// Legacy compatibility also requires the exact historical summarizer.
	base, record := proofFixture(t, boundaryCommand(), false)
	record["command_summary"] = "X" + record["command_summary"].(string)[1:]
	record["command"] = record["command_summary"]
	if v := CheckCommandRecord(base, "progress", record); !proofMessagesContain(v, "does not match") {
		t.Fatalf("forged legacy summary passed: %v", v)
	}
	// A valid old receipt does not acquire a universal cache requirement.
	record = map[string]any{"command_summary": "echo " + strings.Repeat("x", 192) + "...", "command_ref": "missing", "command_sha256": strings.Repeat("a", 64)}
	if v := CheckCommandRecord(t.TempDir(), "progress", record); len(v) != 0 {
		t.Fatalf("valid old cacheless receipt rejected: %v", v)
	}
}

func TestReadCommandProofContainmentAndBounds(t *testing.T) {
	base, record := proofFixture(t, boundaryCommand(), false)
	ref := record["command_ref"].(string)
	digest := record["command_sha256"].(string)
	for _, bad := range []string{"../" + ref, "/" + ref, ".small-cache/../" + ref, "https://example.com/command", strings.ReplaceAll(ref, "/", `\`), strings.Replace(ref, "/commands/", "/commands/../commands/", 1)} {
		if _, err := ReadCommandProof(base, bad, digest); err == nil {
			t.Fatalf("unsafe ref accepted: %s", bad)
		}
	}
	path := filepath.Join(base, ref)
	if err := os.WriteFile(path, []byte(strings.Repeat("x", MaxCommandProofBytes+1)), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadCommandProof(base, ref, digest); err == nil || !strings.Contains(err.Error(), "byte limit") {
		t.Fatalf("unbounded proof: %v", err)
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(t.TempDir(), "command.txt")
	if err := os.WriteFile(outside, []byte(boundaryCommand()), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, path); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadCommandProof(base, ref, digest); err == nil {
		t.Fatal("file symlink accepted")
	}
	fresh := t.TempDir()
	if err := os.Symlink(filepath.Join(base, CacheDirName), filepath.Join(fresh, CacheDirName)); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadCommandProof(fresh, ref, digest); err == nil {
		t.Fatal("directory symlink escape accepted")
	}
}

func TestReadCommandProofCorruptCacheDoesNotUsePortableFallback(t *testing.T) {
	base, record := proofFixture(t, boundaryCommand(), true)
	ref := record["command_ref"].(string)
	portable := filepath.Join(base, CommandProofDir+strings.TrimPrefix(ref, CacheDirName))
	if err := os.MkdirAll(filepath.Dir(portable), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(portable, []byte(boundaryCommand()), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(base, ref), []byte("tampered"), 0o600); err != nil {
		t.Fatal(err)
	}
	if v := CheckCommandRecord(base, "progress", record); !proofMessagesContain(v, "SHA256 mismatch") {
		t.Fatalf("corrupt cache bypassed: %v", v)
	}
}

func TestSummarizeCommandBoundaries(t *testing.T) {
	for _, command := range []string{
		"echo http://127.0.0.1:4345/path tail", "echo 'http://[::1]:4345/path' tail",
		`echo "http://localhost:3000/path" tail`, "echo https://example.com:443/path tail",
		"echo http://127.0.0.1/a http://[::1]:3000/b tail", "echo prehttp://localhost:3000/path tail",
		"echo\t\nhttp://0.0.0.0:4345/path\t tail", "é日🙂 " + strings.Repeat("界", 20), strings.Repeat("é", 50),
	} {
		normalized := normalizeCommand(command)
		for cap := 1; cap < len(normalized); cap++ {
			got := SummarizeCommand(command, cap)
			if len(got) > cap || !utf8.ValidString(got) {
				t.Fatalf("cap %d: invalid display %q", cap, got)
			}
			// Every included HTTP URL must be one of the complete original URLs.
			for _, url := range extractHTTPURLs(got) {
				found := false
				for _, original := range extractHTTPURLs(normalized) {
					if url == original {
						found = true
					}
				}
				if !found {
					t.Fatalf("cap %d: partial URL %q from %q", cap, url, command)
				}
			}
		}
		if got := SummarizeCommand(command, len(normalized)); got != normalized {
			t.Fatalf("unbounded normalization: %q", got)
		}
	}
	for _, cap := range []int{0, -1} {
		if got := SummarizeCommand(boundaryCommand(), cap); got != SummarizeCommand(boundaryCommand(), DefaultCommandSummaryCap) {
			t.Fatal("default cap drift")
		}
	}
}

func TestExactLocalhostHostnames(t *testing.T) {
	for _, url := range []string{"http://localhost.evil", "http://127.0.0.1.evil", "http://0.0.0.0.evil", "http://localhost@evil.example", "http://evil@localhost", "http://[::1].evil", "http://127.0.0.12", "http://127.0.0.1:bad", "http://localhost%2eevil", "http://[::1%25zone]"} {
		if isAllowedLocalhostHTTP(url) {
			t.Fatalf("loopback lookalike allowed: %s", url)
		}
	}
}

func proofMessagesContain(violations []InvariantViolation, text string) bool {
	for _, v := range violations {
		if strings.Contains(v.Message, text) {
			return true
		}
	}
	return false
}
