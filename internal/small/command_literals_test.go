package small

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestCapturedLoopbackLiterals(t *testing.T) {
	for _, literal := range []string{
		"`http://127.0.0.1:${server.address().port}`",
		"`http://localhost:${port}/health`",
		"`http://0.0.0.0:${PORT}?health=1`",
		"`http://[::1]:${address.port}/health`",
		"`http://127.0.0.1:3000`",
	} {
		for _, marked := range []bool{false, true} {
			t.Run(literal+map[bool]string{false: "/short", true: "/bounded"}[marked], func(t *testing.T) {
				command := "node -e 'const origin = " + literal + ";'"
				if marked {
					command = "apply_patch <<'PATCH'\n*** Begin Patch\n*** Add File: qa.mjs\n+// " + strings.Repeat("x", 220) + "\n+const origin = " + literal + ";\n*** End Patch\nPATCH"
				}
				base, record := proofFixture(t, command, marked)
				before := make(map[string]any)
				for k, v := range record {
					before[k] = v
				}
				if got := CheckCommandRecord(base, "command", record); len(got) != 0 {
					t.Fatalf("verified v2 command policy: %v", got)
				}
				artifact := &Artifact{Type: "progress", Path: filepath.Join(base, ".small", "progress.small.yml"), Data: map[string]any{"entries": []any{record}}}
				copy, errors := commandSecurityArtifact(artifact)
				if len(errors) != 0 || len(checkInsecureLinks(copy)) != 0 {
					t.Fatalf("verified v1 command policy: %v / %v", errors, checkInsecureLinks(copy))
				}
				if !reflect.DeepEqual(record, before) {
					t.Fatal("receipt mutated")
				}
				bytes, err := os.ReadFile(filepath.Join(base, record["command_ref"].(string)))
				if err != nil || string(bytes) != command {
					t.Fatal("original proof mutated")
				}
			})
		}
	}
}

func TestCapturedLoopbackLiteralsFailClosed(t *testing.T) {
	cases := []string{
		"`http://external.example:${port}`",
		"`http://localhost.evil:${port}`",
		"`http://127.0.0.1.evil:${port}`",
		"`http://127.0.0.12:${port}`",
		"`http://[::1].evil:${port}`",
		"`http://evil@localhost:${port}`",
		"`http://localhost:${port}@evil.example`",
		"`http://localhost:${port}.evil`",
		"`http://localhost:bad`",
		"`http://localhost:${port || 3000}`",
		"`http://localhost:${getPort()}`",
		"`http://${host}:${port}`",
		"http://localhost:${port}",
		"`http://localhost:${port}?next=http://external.example`",
		"`http://localhost:${port}` http://external.example",
	}
	for _, literal := range cases {
		t.Run(literal, func(t *testing.T) {
			base, record := proofFixture(t, "node -e 'const origin = "+literal+";'", true)
			if got := CheckCommandRecord(base, "command", record); !proofMessagesContain(got, "insecure link") {
				t.Fatalf("unsafe/unsupported literal passed: %v", got)
			}
		})
	}
}

func TestCapturedLoopbackLiteralDoesNotExemptAuthoredFields(t *testing.T) {
	literal := "`http://localhost:${port}`"
	for _, field := range []string{"notes", "evidence", "link", "test", "verification", "entries/0/command_summary", "command/summary"} {
		t.Run(field, func(t *testing.T) {
			base, record := proofFixture(t, "node -e 'const origin = "+literal+";'", true)
			record[field] = literal
			if got := CheckCommandRecord(base, "command", record); !proofMessagesContain(got, "insecure link") {
				t.Fatalf("authored template exempted: %v", got)
			}
			artifact := &Artifact{Type: "progress", Path: filepath.Join(base, ".small", "progress.small.yml"), Data: map[string]any{"entries": []any{record}}}
			copy, _ := commandSecurityArtifact(artifact)
			if len(checkInsecureLinks(copy)) == 0 {
				t.Fatal("authored v1 template exempted")
			}
		})
	}
}

func TestCapturedLoopbackLiteralProofRequired(t *testing.T) {
	for _, marked := range []bool{false, true} {
		base, record := proofFixture(t, "node -e 'const origin = `http://localhost:${port}`;'", marked)
		if err := os.Remove(filepath.Join(base, record["command_ref"].(string))); err != nil {
			t.Fatal(err)
		}
		if got := CheckCommandRecord(base, "command", record); !proofMessagesContain(got, "proof unavailable") {
			t.Fatalf("missing exact proof passed: %v", got)
		}
	}
}

func TestCapturedLoopbackLiteralDoesNotMaskSecrets(t *testing.T) {
	variable := "gh" + "p_" + strings.Repeat("fixture", 4)
	commands := []string{
		"node -e 'const origin = `http://localhost:${" + variable + "}`;'",
		"node -e 'const origin = `http://localhost:${port}`;' " + strings.Repeat("x", 250) + " API_KEY=" + "sk-" + strings.Repeat("fixture", 4),
	}
	for _, command := range commands {
		base, record := proofFixture(t, command, true)
		if got := CheckCommandRecord(base, "command", record); !proofMessagesContain(got, "potential secret") {
			t.Fatalf("credential hidden by literal normalization: %v", got)
		}
	}
}

func TestCapturedLoopbackLiteralTamperingFailsClosed(t *testing.T) {
	for _, marked := range []bool{false, true} {
		base, record := proofFixture(t, "node -e 'const origin = `http://localhost:${port}`;'", marked)
		p := filepath.Join(base, record["command_ref"].(string))
		bytes, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, append(bytes, ' '), 0o600); err != nil {
			t.Fatal(err)
		}
		if got := CheckCommandRecord(base, "command", record); !proofMessagesContain(got, "SHA256 mismatch") {
			t.Fatalf("tampered literal proof passed: %v", got)
		}
	}
}

func TestCapturedLoopbackLiteralPathMetadataCannotBeAuthored(t *testing.T) {
	literal := "`http://localhost:${port}`"
	base, record := proofFixture(t, "node -e 'const origin = "+literal+";'", true)
	artifact := &Artifact{Type: "progress", Path: filepath.Join(base, ".small", "progress.small.yml"), Data: map[string]any{"entries": []any{record}, "entries/0/command_summary": literal, "verifiedCommandPaths": literal}}
	copy, _ := commandSecurityArtifact(artifact)
	if len(checkInsecureLinks(copy)) != 2 {
		t.Fatalf("authored metadata/path bypassed URL policy: %v", checkInsecureLinks(copy))
	}
}

func TestCapturedLoopbackLiteralLegacyHiddenHTTPStillFails(t *testing.T) {
	command := "node -e 'const origin = `http://localhost:${port}`;' " + strings.Repeat("x", 250) + " http://external.example"
	base, record := proofFixture(t, command, false)
	if got := CheckCommandRecord(base, "command", record); !proofMessagesContain(got, "insecure link") {
		t.Fatalf("legacy proof hid external HTTP after cap: %v", got)
	}
}
