package sessionv2

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDecodeStrictRejectsAmbiguousJSON(t *testing.T) {
	var profile Profile
	if err := DecodeStrict([]byte(`{"small_version":"2.0.0","small_version":"2.0.0"}`), &profile); err == nil || !strings.Contains(err.Error(), "duplicate") {
		t.Fatalf("duplicate key error = %v", err)
	}
	var event Event
	if err := DecodeStrict([]byte(`{"sequence":1.5}`), &event); err == nil || !strings.Contains(err.Error(), "non-integer") {
		t.Fatalf("fraction error = %v", err)
	}
}

func TestImmutableIdentityAndReducerConflict(t *testing.T) {
	base := t.TempDir()
	profile := Profile{SmallVersion: ProfileVersion, ProjectID: "project_test", LineageID: "lineage_test", Mode: "collaborative", PolicyRevision: "policy_1"}
	data, _ := CanonicalJSON(profile)
	if err := os.MkdirAll(filepath.Join(base, ".small"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(base, ".small", "profile.json"), append(data, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"session_a", "session_b"} {
		session := Session{SmallVersion: ProfileVersion, ProjectID: profile.ProjectID, LineageID: profile.LineageID, SessionID: id, ObservedFrontier: Frontier(nil), ToolVersion: "test", CreatedAt: "2026-01-01T00:00:00Z"}
		if _, err := PublishSession(base, session); err != nil {
			t.Fatal(err)
		}
	}
	created := Event{SmallVersion: ProfileVersion, ProjectID: profile.ProjectID, LineageID: profile.LineageID, SessionID: "session_a", EventID: "event_create", Kind: "task_created", Sequence: 1, OccurredAt: "2026-01-01T00:00:00Z", TaskID: "task_native", TaskRevision: 1, PolicyRevision: "policy_1", Payload: map[string]any{"alias": "task-1", "title": "shared", "acceptance": []string{"verify"}}}
	if _, err := PublishEvent(base, created); err != nil {
		t.Fatal(err)
	}
	left := Event{SmallVersion: ProfileVersion, ProjectID: profile.ProjectID, LineageID: profile.LineageID, SessionID: "session_a", EventID: "event_left", Kind: "task_transitioned", Sequence: 2, PreviousEvent: "event_create", OccurredAt: "2026-01-01T00:00:02Z", TaskID: "task_native", TaskRevision: 1, PolicyRevision: "policy_1", SourceDigest: "candidate_a", Payload: map[string]any{"status": "completed"}}
	right := Event{SmallVersion: ProfileVersion, ProjectID: profile.ProjectID, LineageID: profile.LineageID, SessionID: "session_b", EventID: "event_right", Kind: "task_transitioned", Sequence: 1, Parents: []string{"event_create"}, OccurredAt: "2025-01-01T00:00:00Z", TaskID: "task_native", TaskRevision: 1, PolicyRevision: "policy_1", SourceDigest: "candidate_b", Payload: map[string]any{"status": "blocked"}}
	if _, err := PublishEvent(base, left); err != nil {
		t.Fatal(err)
	}
	if _, err := PublishEvent(base, right); err != nil {
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
	if len(state.Conflicts) != 1 || state.Conflicts[0].Kind != "task_outcome" {
		t.Fatalf("conflicts = %#v", state.Conflicts)
	}
	if _, err := Strict(store); !IsConflict(err) {
		t.Fatalf("strict error = %v", err)
	}

	left.EventID = "event_same_id"
	left.Sequence = 3
	left.PreviousEvent = "event_left"
	if _, err := PublishEvent(base, left); err != nil {
		t.Fatal(err)
	}
	left.Payload["status"] = "blocked"
	if _, err := PublishEvent(base, left); err == nil || !strings.Contains(err.Error(), "corruption") {
		t.Fatalf("altered identity error = %v", err)
	}
}

func TestValidationRejectsMissingParentAndSessionFork(t *testing.T) {
	profile := Profile{SmallVersion: ProfileVersion, ProjectID: "p", LineageID: "l", Mode: "solo", PolicyRevision: "r"}
	session := Session{SmallVersion: ProfileVersion, ProjectID: "p", LineageID: "l", SessionID: "s", ObservedFrontier: Frontier(nil), ToolVersion: "t", CreatedAt: "now"}
	session.Digest, _ = DigestRecord(session)
	event := Event{SmallVersion: ProfileVersion, ProjectID: "p", LineageID: "l", SessionID: "s", EventID: "e", Kind: "session_started", Sequence: 1, Parents: []string{"missing"}, OccurredAt: "now", Payload: map[string]any{}}
	event.Digest, _ = DigestRecord(event)
	store := &Store{Profile: profile, Sessions: map[string]Session{"s": session}, Events: map[string]Event{"e": event}, Receipts: map[string]Receipt{}}
	if err := Validate(store); err == nil || !strings.Contains(err.Error(), "missing parent") {
		t.Fatalf("missing parent error = %v", err)
	}
	event.Parents = nil
	event.Digest, _ = DigestRecord(event)
	other := event
	other.EventID = "other"
	other.Digest, _ = DigestRecord(other)
	store.Events = map[string]Event{"e": event, "other": other}
	if err := Validate(store); err == nil || !strings.Contains(err.Error(), "fork") {
		t.Fatalf("fork error = %v", err)
	}
}

func TestExplicitResolutionSelectsOutcomeAndRejectsStaleResolution(t *testing.T) {
	base := t.TempDir()
	profile := Profile{SmallVersion: ProfileVersion, ProjectID: "resolve_project", LineageID: "resolve_lineage", Mode: "collaborative", PolicyRevision: "policy_1"}
	if err := Initialize(base, profile, nil, nil); err != nil {
		t.Fatal(err)
	}
	a, _, err := StartSession(base, SessionStartOptions{ToolVersion: "test"})
	if err != nil {
		t.Fatal(err)
	}
	created, err := AppendEvent(base, a.SessionID, "task_created", map[string]any{"alias": "task-1", "title": "shared"}, AppendOptions{TaskID: "task_shared", TaskRevision: 1})
	if err != nil {
		t.Fatal(err)
	}
	b, bEvents, err := StartSession(base, SessionStartOptions{ToolVersion: "test"})
	if err != nil {
		t.Fatal(err)
	}
	left := Event{SmallVersion: ProfileVersion, ProjectID: profile.ProjectID, LineageID: profile.LineageID, SessionID: a.SessionID, EventID: "event_left_resolution", Kind: "task_transitioned", Sequence: 3, PreviousEvent: created.EventID, OccurredAt: "2026-01-01T00:00:01Z", TaskID: "task_shared", TaskRevision: 1, PolicyRevision: profile.PolicyRevision, SourceDigest: "left", Payload: map[string]any{"status": "completed"}}
	if _, err := PublishEvent(base, left); err != nil {
		t.Fatal(err)
	}
	right := Event{SmallVersion: ProfileVersion, ProjectID: profile.ProjectID, LineageID: profile.LineageID, SessionID: b.SessionID, EventID: "event_right_resolution", Kind: "task_transitioned", Sequence: 2, PreviousEvent: bEvents[0].EventID, OccurredAt: "2025-01-01T00:00:01Z", TaskID: "task_shared", TaskRevision: 1, PolicyRevision: profile.PolicyRevision, SourceDigest: "right", Payload: map[string]any{"status": "blocked"}}
	if _, err := PublishEvent(base, right); err != nil {
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
	if len(state.Conflicts) != 1 {
		t.Fatalf("conflicts = %#v", state.Conflicts)
	}
	conflict := state.Conflicts[0]
	frontier := state.Frontier
	resolution, err := AppendEvent(base, a.SessionID, "conflict_resolved", map[string]any{"conflict_id": conflict.ID, "heads": conflict.Heads, "selected_event": right.EventID, "expected_frontier": frontier, "reason": "reviewed"}, AppendOptions{ExpectedFrontier: frontier, Parents: []string{left.EventID, right.EventID}})
	if err != nil {
		t.Fatal(err)
	}
	store, err = Load(base)
	if err != nil {
		t.Fatal(err)
	}
	state, err = Strict(store)
	if err != nil {
		t.Fatal(err)
	}
	if state.Tasks["task_shared"].Status != "blocked" {
		t.Fatalf("selected status = %q", state.Tasks["task_shared"].Status)
	}
	if resolution.Kind != "conflict_resolved" {
		t.Fatalf("resolution = %#v", resolution)
	}
	if _, err := AppendEvent(base, a.SessionID, "conflict_resolved", map[string]any{"conflict_id": conflict.ID}, AppendOptions{ExpectedFrontier: frontier}); !errors.Is(err, ErrStaleFrontier) {
		t.Fatalf("stale resolution error = %v", err)
	}
}
