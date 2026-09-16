package commands

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/justyn-clark/small-protocol/internal/sessionv2"
)

func TestV2OperationsSeparateExecutionFromAcceptanceAndBoundResume(t *testing.T) {
	base := t.TempDir()
	profile := sessionv2.Profile{SmallVersion: sessionv2.ProfileVersion, ProjectID: "project_ops", LineageID: "lineage_ops", Mode: "collaborative", PolicyRevision: "policy_1"}
	if err := sessionv2.Initialize(base, profile, nil, nil); err != nil {
		t.Fatal(err)
	}
	session, _, err := sessionv2.StartSession(base, sessionv2.SessionStartOptions{ToolVersion: "test"})
	if err != nil {
		t.Fatal(err)
	}
	if err := runV2Plan(base, session.SessionID, false, "Verify task", "", "", "", ""); err != nil {
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
	if err := runV2Progress(base, session.SessionID, task.ID, "in_progress", "started", "", "", "", true); err != nil {
		t.Fatal(err)
	}
	if err := runV2Apply(base, session.SessionID, task.ID, "printf operation-ok", false, false, "", false, true); err != nil {
		t.Fatal(err)
	}
	store, err = sessionv2.Load(base)
	if err != nil {
		t.Fatal(err)
	}
	state, err = sessionv2.Strict(store)
	if err != nil {
		t.Fatal(err)
	}
	if state.Tasks[task.ID].Status != "in_progress" {
		t.Fatalf("apply accepted task: %#v", state.Tasks[task.ID])
	}
	if err := runV2Checkpoint(base, session.SessionID, task.ID, "completed", "reviewed command output", "", true); err != nil {
		t.Fatal(err)
	}
	store, err = sessionv2.Load(base)
	if err != nil {
		t.Fatal(err)
	}
	state, err = sessionv2.Strict(store)
	if err != nil {
		t.Fatal(err)
	}
	if state.Tasks[task.ID].Status != "completed" {
		t.Fatalf("checkpoint status = %q", state.Tasks[task.ID].Status)
	}
	if len(store.Receipts) != 2 {
		t.Fatalf("receipts = %d", len(store.Receipts))
	}
	resume, err := buildV2ReconstructOutput(base, session.SessionID, task.ID, true, 2, 4096)
	if err != nil {
		t.Fatal(err)
	}
	if resume.ReturnedEvents > 2 || !resume.Truncated || resume.MatchedEvents <= resume.ReturnedEvents {
		t.Fatalf("resume bounds = %#v", resume)
	}
}

func TestPortableEvidenceVerificationDetectsTamper(t *testing.T) {
	base := t.TempDir()
	profile := sessionv2.Profile{SmallVersion: sessionv2.ProfileVersion, ProjectID: "project_evidence", LineageID: "lineage_evidence", Mode: "solo", PolicyRevision: "policy_1"}
	if err := sessionv2.Initialize(base, profile, nil, nil); err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(base, "proof.txt")
	if err := os.WriteFile(source, []byte("proof\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	receipt, err := sessionv2.SavePortableEvidence(base, source, sessionv2.Receipt{Validator: "test", Outcome: "passed"})
	if err != nil {
		t.Fatal(err)
	}
	results, err := sessionv2.VerifyEvidence(base)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].Status != "verified" {
		t.Fatalf("results = %#v", results)
	}
	if err := os.WriteFile(filepath.Join(base, filepath.FromSlash(receipt.ArtifactPath)), []byte("tampered\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	results, err = sessionv2.VerifyEvidence(base)
	if err != nil {
		t.Fatal(err)
	}
	if results[0].Status != "failed" {
		t.Fatalf("tamper results = %#v", results)
	}
}
