package sessionv2

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/justyn-clark/small-protocol/internal/small"
)

func TestSoloLifecycleRequiresExplicitTakeover(t *testing.T) {
	base := t.TempDir()
	profile := Profile{SmallVersion: ProfileVersion, ProjectID: "project_lifecycle", LineageID: "lineage_lifecycle", PolicyRevision: "policy_1"}
	if err := Initialize(base, profile, []byte("intent\n"), []byte("constraints\n")); err != nil {
		t.Fatal(err)
	}
	store, err := Load(base)
	if err != nil {
		t.Fatal(err)
	}
	if store.Profile.Mode != "solo" {
		t.Fatalf("default mode = %q", store.Profile.Mode)
	}
	first, _, err := StartSession(base, SessionStartOptions{Label: "first", ToolVersion: "test"})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := StartSession(base, SessionStartOptions{Label: "second", ToolVersion: "test"}); err == nil || !strings.Contains(err.Error(), "already has active session") {
		t.Fatalf("second start error = %v", err)
	}
	second, events, err := StartSession(base, SessionStartOptions{Label: "second", ToolVersion: "test", TakeoverSessionID: first.SessionID, TakeoverReason: "operator recovery"})
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 2 {
		t.Fatalf("takeover events = %d", len(events))
	}
	store, err = Load(base)
	if err != nil {
		t.Fatal(err)
	}
	state, err := Strict(store)
	if err != nil {
		t.Fatal(err)
	}
	if !state.SupersededSessions[first.SessionID] {
		t.Fatalf("first session not superseded: %#v", state.SupersededSessions)
	}
	active := ActiveSessions(store, state)
	if len(active) != 1 || active[0] != second.SessionID {
		t.Fatalf("active = %#v", active)
	}
	if _, err := AppendEvent(base, first.SessionID, "command_recorded", map[string]any{}, AppendOptions{}); err == nil {
		t.Fatal("superseded session accepted a write")
	}
}

func TestClosePreservesNarrativeAndRequiresExplicitResume(t *testing.T) {
	base := t.TempDir()
	profile := Profile{SmallVersion: ProfileVersion, ProjectID: "project_close", LineageID: "lineage_close", PolicyRevision: "policy_1"}
	if err := Initialize(base, profile, nil, nil); err != nil {
		t.Fatal(err)
	}
	session, _, err := StartSession(base, SessionStartOptions{ToolVersion: "test"})
	if err != nil {
		t.Fatal(err)
	}
	events, err := CloseSession(base, session.SessionID, "Implemented parser.\nNext: verify migration.")
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 2 || events[0].Kind != "handoff_recorded" || events[1].Kind != "session_closed" {
		t.Fatalf("events = %#v", events)
	}
	if got := stringPayload(events[0].Payload, "summary"); got != "Implemented parser.\nNext: verify migration." {
		t.Fatalf("summary = %q", got)
	}
	if _, err := ResolveSession(base, ""); !errors.Is(err, ErrAmbiguous) {
		t.Fatalf("resolve error = %v", err)
	}
	resumed, _, err := StartSession(base, SessionStartOptions{ToolVersion: "test", FromHandoff: events[0].EventID, ParentSessionID: session.SessionID})
	if err != nil {
		t.Fatal(err)
	}
	if resumed.FromHandoff != events[0].EventID {
		t.Fatalf("from handoff = %q", resumed.FromHandoff)
	}
}

func TestModeChangeUsesFrontierCAS(t *testing.T) {
	base := t.TempDir()
	profile := Profile{SmallVersion: ProfileVersion, ProjectID: "project_mode", LineageID: "lineage_mode", PolicyRevision: "policy_1"}
	if err := Initialize(base, profile, nil, nil); err != nil {
		t.Fatal(err)
	}
	session, _, err := StartSession(base, SessionStartOptions{ToolVersion: "test"})
	if err != nil {
		t.Fatal(err)
	}
	store, err := Load(base)
	if err != nil {
		t.Fatal(err)
	}
	frontier := Frontier(store.Events)
	if _, err := AppendEvent(base, session.SessionID, "mode_changed", map[string]any{"new_mode": "collaborative"}, AppendOptions{ExpectedFrontier: "stale"}); !errors.Is(err, ErrStaleFrontier) {
		t.Fatalf("stale error = %v", err)
	}
	if _, err := AppendEvent(base, session.SessionID, "mode_changed", map[string]any{"new_mode": "collaborative"}, AppendOptions{ExpectedFrontier: frontier}); err != nil {
		t.Fatal(err)
	}
	store, err = Load(base)
	if err != nil {
		t.Fatal(err)
	}
	state, err := Strict(store)
	if err != nil {
		t.Fatal(err)
	}
	if state.Profile.Mode != "collaborative" {
		t.Fatalf("mode = %q", state.Profile.Mode)
	}
}

func TestPolicyRevisionIsAtomicAndMaterialBound(t *testing.T) {
	base := t.TempDir()
	profile := Profile{SmallVersion: ProfileVersion, ProjectID: "project_policy", LineageID: "lineage_policy", PolicyRevision: "policy_initial"}
	if err := Initialize(base, profile, []byte("intent: initial\n"), []byte("constraints: initial\n")); err != nil {
		t.Fatal(err)
	}
	session, _, err := StartSession(base, SessionStartOptions{ToolVersion: "test"})
	if err != nil {
		t.Fatal(err)
	}
	store, err := Load(base)
	if err != nil {
		t.Fatal(err)
	}
	frontier := Frontier(store.Events)
	material := PolicyMaterial{Intent: []byte("intent: revised\n"), Constraints: []byte("constraints: revised\n"), HasIntent: true, HasConstraints: true}
	event, err := RevisePolicy(base, session.SessionID, frontier, "reviewed requirements", "test", material)
	if err != nil {
		t.Fatal(err)
	}
	store, err = Load(base)
	if err != nil {
		t.Fatal(err)
	}
	state, err := Strict(store)
	if err != nil {
		t.Fatal(err)
	}
	if state.Profile.PolicyRevision != stringPayload(event.Payload, "new_revision") {
		t.Fatalf("policy revision = %q", state.Profile.PolicyRevision)
	}
	if err := os.WriteFile(filepath.Join(base, ".small", "policy", "intent.small.yml"), []byte("intent: unrecorded\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	store, err = Load(base)
	if err != nil {
		t.Fatal(err)
	}
	state, err = Reduce(store)
	if err != nil {
		t.Fatal(err)
	}
	if len(state.Conflicts) != 1 || state.Conflicts[0].Kind != "policy_material" {
		t.Fatalf("policy material conflicts = %#v", state.Conflicts)
	}
}

func TestEventAndReceiptRecoverAsOneTransaction(t *testing.T) {
	base := t.TempDir()
	profile := Profile{SmallVersion: ProfileVersion, ProjectID: "project_txn", LineageID: "lineage_txn", PolicyRevision: "policy_1"}
	if err := Initialize(base, profile, nil, nil); err != nil {
		t.Fatal(err)
	}
	session, _, err := StartSession(base, SessionStartOptions{ToolVersion: "test"})
	if err != nil {
		t.Fatal(err)
	}
	store, err := Load(base)
	if err != nil {
		t.Fatal(err)
	}
	state, err := Reduce(store)
	if err != nil {
		t.Fatal(err)
	}
	receipt := Receipt{Strength: "cli_captured", Availability: "verified", Outcome: "passed"}
	t.Setenv("SMALL_TEST_FAIL_TRANSACTION_AFTER", "publish-1")
	if _, _, err := AppendEventWithReceipts(base, session.SessionID, "command_recorded", map[string]any{"outcome": "passed"}, AppendOptions{ExpectedFrontier: state.Frontier}, []Receipt{receipt}); err == nil {
		t.Fatal("fault did not interrupt")
	}
	t.Setenv("SMALL_TEST_FAIL_TRANSACTION_AFTER", "")
	if err := small.RecoverStateTransactions(base); err != nil {
		t.Fatal(err)
	}
	store, err = Load(base)
	if err != nil {
		t.Fatal(err)
	}
	if len(store.Events) != 2 || len(store.Receipts) != 1 {
		t.Fatalf("events=%d receipts=%d", len(store.Events), len(store.Receipts))
	}
}
