package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/justyn-clark/small-protocol/internal/sessionv2"
	"github.com/justyn-clark/small-protocol/internal/small"
	"github.com/justyn-clark/small-protocol/internal/workspace"
)

func TestApplyCommandProofAuditAndDryRun(t *testing.T) {
	t.Setenv(progressModeEnvVar, string(progressModeAudit))
	command := "printf '%s\\n' '" + strings.Repeat("0", 169) + " http://127.0.0.1:4345/'"
	for _, dry := range []bool{false, true} {
		base := t.TempDir()
		writeArtifacts(t, base, defaultArtifacts())
		mustSaveWorkspace(t, base, workspace.KindRepoRoot)
		args := []string{"--dir", base, "--workspace", "any", "--task", "task-1", "--cmd", command, "--json"}
		if dry {
			args = append(args, "--dry-run")
		}
		cmd := applyCmd()
		cmd.SetArgs(args)
		if err := cmd.Execute(); err != nil {
			t.Fatal(err)
		}
		progress, err := loadProgressData(filepath.Join(base, ".small", "progress.small.yml"))
		if err != nil {
			t.Fatal(err)
		}
		want := 2
		if dry {
			want = 1
		}
		if len(progress.Entries) != want {
			t.Fatalf("receipts = %d, want %d", len(progress.Entries), want)
		}
		for _, entry := range progress.Entries {
			if entry["command_summary_version"] != 2 {
				t.Fatalf("missing bounded marker: %#v", entry)
			}
			if v := small.CheckCommandRecord(base, "progress", entry); len(v) != 0 {
				t.Fatalf("generated receipt failed: %v", v)
			}
		}
		violations, err := runLintArtifacts(base, true)
		if err != nil || len(violations) != 0 {
			t.Fatalf("strict lint: %v %v", violations, err)
		}
		if err := os.RemoveAll(filepath.Join(base, small.CacheDirName)); err != nil {
			t.Fatal(err)
		}
		violations, err = runLintArtifacts(base, true)
		if err != nil || len(violations) == 0 {
			t.Fatalf("missing bounded proof accepted: %v %v", violations, err)
		}
	}
}

func TestV2CommandProofStrictAndPortable(t *testing.T) {
	for _, suffix := range []string{"", " # http://external.example", " # API_KEY=sk-abcdefghijklmnopqrstuvwxyz0123456789"} {
		t.Run(suffix, func(t *testing.T) {
			base := t.TempDir()
			profile := sessionv2.Profile{SmallVersion: sessionv2.ProfileVersion, ProjectID: "project_command_proof", LineageID: "lineage_command_proof", Mode: "solo", PolicyRevision: "policy_1"}
			if err := sessionv2.Initialize(base, profile, nil, nil); err != nil {
				t.Fatal(err)
			}
			session, _, err := sessionv2.StartSession(base, sessionv2.SessionStartOptions{ToolVersion: "test"})
			if err != nil {
				t.Fatal(err)
			}
			if err := runV2Plan(base, session.SessionID, false, "Boundary command", "", "", "", ""); err != nil {
				t.Fatal(err)
			}
			command := "printf '%s\\n' '" + strings.Repeat("0", 169) + " http://[::1]:4345/'" + suffix
			if err := runV2Apply(base, session.SessionID, "task-1", command, false, false, "", false, true); err != nil {
				t.Fatal(err)
			}
			store, err := sessionv2.Load(base)
			if err != nil {
				t.Fatal(err)
			}
			if store.Profile.SmallVersion != sessionv2.ProfileVersion {
				t.Fatal("profile changed")
			}
			_, strictErr := sessionv2.Strict(store)
			code, _, err := runCheck(base, true, true, false, workspace.ScopeAny, false)
			if err != nil {
				t.Fatal(err)
			}
			if suffix != "" {
				if strictErr == nil || code == ExitValid {
					t.Fatal("unsafe full v2 command passed")
				}
				return
			}
			if strictErr != nil || code != ExitValid {
				t.Fatalf("v2 strict: %v %d", strictErr, code)
			}
			var ref string
			for _, event := range store.Events {
				if event.Kind == "command_recorded" {
					ref = event.Payload["command_ref"].(string)
				}
			}
			original, err := os.ReadFile(filepath.Join(base, ref))
			if err != nil {
				t.Fatal(err)
			}
			if err := os.RemoveAll(filepath.Join(base, small.CacheDirName)); err != nil {
				t.Fatal(err)
			}
			store, err = sessionv2.Load(base)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := sessionv2.Strict(store); err == nil {
				t.Fatal("cacheless v2 without proof accepted")
			}
			portable := filepath.Join(base, small.CommandProofDir+strings.TrimPrefix(ref, small.CacheDirName))
			if err := os.MkdirAll(filepath.Dir(portable), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(portable, original, 0o600); err != nil {
				t.Fatal(err)
			}
			if _, err := sessionv2.Strict(store); err != nil {
				t.Fatalf("portable v2 proof: %v", err)
			}
			code, _, err = runCheck(base, true, true, false, workspace.ScopeAny, false)
			if err != nil || code != ExitValid {
				t.Fatalf("portable v2 strict: %v %d", err, code)
			}
			if err := os.WriteFile(portable, append(original, '\n'), 0o600); err != nil {
				t.Fatal(err)
			}
			if _, err := sessionv2.Strict(store); err == nil {
				t.Fatal("tampered portable v2 proof accepted")
			}
		})
	}
}

func TestV2CommandProofPersistenceFailureDoesNotRetry(t *testing.T) {
	base := t.TempDir()
	profile := sessionv2.Profile{SmallVersion: sessionv2.ProfileVersion, ProjectID: "project_command_failure", LineageID: "lineage_command_failure", Mode: "solo", PolicyRevision: "policy_1"}
	if err := sessionv2.Initialize(base, profile, nil, nil); err != nil {
		t.Fatal(err)
	}
	session, _, err := sessionv2.StartSession(base, sessionv2.SessionStartOptions{ToolVersion: "test"})
	if err != nil {
		t.Fatal(err)
	}
	if err := runV2Plan(base, session.SessionID, false, "Record once", "", "", "", ""); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(base, small.CacheDirName, "commands"), []byte("not a directory"), 0o600); err != nil {
		t.Fatal(err)
	}
	err = runV2Apply(base, session.SessionID, "task-1", "printf once >> executed.txt", false, false, "", false, true)
	if err == nil || !strings.Contains(err.Error(), "not retried") {
		t.Fatalf("missing persistence failure: %v", err)
	}
	if _, ok := err.(reportedJSONError); !ok {
		t.Fatalf("JSON failure not reported: %T", err)
	}
	data, err := os.ReadFile(filepath.Join(base, "executed.txt"))
	if err != nil || string(data) != "once" {
		t.Fatalf("child retried or not executed: %q %v", data, err)
	}
	store, err := sessionv2.Load(base)
	if err != nil {
		t.Fatal(err)
	}
	for _, event := range store.Events {
		if event.Kind == "command_recorded" {
			t.Fatal("unpersisted command marked recorded")
		}
	}
}
