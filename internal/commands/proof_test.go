package commands

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/justyn-clark/small-protocol/internal/sessionv2"
	"github.com/justyn-clark/small-protocol/internal/small"
)

func proofInventoryFiles(t *testing.T, base string) map[string][32]byte {
	t.Helper()
	files := map[string][32]byte{}
	err := filepath.WalkDir(base, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() {
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			files[path] = sha256.Sum256(data)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return files
}

func TestProofStatusV1JSONReadOnly(t *testing.T) {
	base := t.TempDir()
	writeArtifacts(t, base, defaultArtifacts())
	replay := strings.Repeat("a", 64)
	timestamp := "2026-10-29T20:00:00.000000000Z"
	command := "printf fixture-value"
	ref, digest, err := small.WriteCommandLog(base, replay, timestamp, command)
	if err != nil {
		t.Fatal(err)
	}
	artifact, err := small.LoadArtifact(base, "progress.small.yml")
	if err != nil {
		t.Fatal(err)
	}
	artifact.Data["entries"] = []any{map[string]any{"timestamp": timestamp, "replayId": replay, "command_summary": command, "command_ref": ref, "command_sha256": digest}}
	if err := small.SaveArtifact(base, "progress.small.yml", artifact.Data); err != nil {
		t.Fatal(err)
	}
	before := proofInventoryFiles(t, base)
	var output bytes.Buffer
	cmd := newRootCmd()
	cmd.SetOut(&output)
	cmd.SetArgs([]string{"proof", "status", "--json", "--dir", filepath.Join(base, ".small")})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	var report small.CommandProofInventory
	if err := json.Unmarshal(output.Bytes(), &report); err != nil {
		t.Fatal(err)
	}
	if report.Coverage != "complete" || report.Available != 1 || report.DisplayDigestMatches != 1 {
		t.Fatalf("wrong inventory: %#v", report)
	}
	if strings.Contains(output.String(), "fixture-value") {
		t.Fatal("command disclosed")
	}
	if !reflect.DeepEqual(before, proofInventoryFiles(t, base)) {
		t.Fatal("proof status wrote files")
	}
}

func TestProofStatusV2ReadOnly(t *testing.T) {
	base := t.TempDir()
	profile := sessionv2.Profile{SmallVersion: sessionv2.ProfileVersion, ProjectID: "proof_project", LineageID: "proof_lineage", Mode: "collaborative", PolicyRevision: "policy_1"}
	if err := sessionv2.Initialize(base, profile, nil, nil); err != nil {
		t.Fatal(err)
	}
	session, _, err := sessionv2.StartSession(base, sessionv2.SessionStartOptions{ToolVersion: "test"})
	if err != nil {
		t.Fatal(err)
	}
	if err := runV2Plan(base, session.SessionID, false, "Proof coverage", "", "", "", ""); err != nil {
		t.Fatal(err)
	}
	store, err := sessionv2.Load(base)
	if err != nil {
		t.Fatal(err)
	}
	state, err := sessionv2.Reduce(store)
	if err != nil {
		t.Fatal(err)
	}
	task, err := resolveV2Task(state, "task-1")
	if err != nil {
		t.Fatal(err)
	}
	if err := runV2Apply(base, session.SessionID, task.ID, "printf proof-status", false, false, "", false, true); err != nil {
		t.Fatal(err)
	}
	before := proofInventoryFiles(t, base)
	report, err := buildCommandProofInventory(base)
	if err != nil {
		t.Fatal(err)
	}
	if report.Available != 1 || report.Total != 1 || report.Coverage != "complete" {
		t.Fatalf("wrong v2 inventory: %#v", report)
	}
	if !reflect.DeepEqual(before, proofInventoryFiles(t, base)) {
		t.Fatal("v2 inventory wrote files")
	}
}
