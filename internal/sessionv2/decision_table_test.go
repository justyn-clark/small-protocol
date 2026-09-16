package sessionv2

import "testing"

func TestDecisionTableEquivalentOutcomesRetainBothAttestations(t *testing.T) {
	store, create := decisionStore(t)
	left := decisionEvent(store, "a", "left", "task_transitioned", 1, "", []string{create.EventID}, "task", map[string]any{"status": "completed"})
	left.SourceDigest = "same"
	left.Digest, _ = DigestRecord(left)
	right := decisionEvent(store, "b", "right", "task_transitioned", 1, "", []string{create.EventID}, "task", map[string]any{"status": "completed"})
	right.SourceDigest = "same"
	right.OccurredAt = "2020-01-01T00:00:00Z"
	right.Digest, _ = DigestRecord(right)
	store.Events[left.EventID] = left
	store.Events[right.EventID] = right
	state, err := Strict(store)
	if err != nil {
		t.Fatal(err)
	}
	if len(state.Tasks["task"].Attestations) != 2 {
		t.Fatalf("attestations = %#v", state.Tasks["task"].Attestations)
	}
}

func TestDecisionTableStaleAndUnavailableEvidenceConflict(t *testing.T) {
	store, create := decisionStore(t)
	update := decisionEvent(store, "base", "update", "task_updated", 2, create.EventID, nil, "task", map[string]any{"alias": "task-1", "title": "revised"})
	store.Events[update.EventID] = update
	stale := decisionEvent(store, "a", "stale", "task_transitioned", 1, "", []string{update.EventID}, "task", map[string]any{"status": "completed"})
	store.Events[stale.EventID] = stale
	missing := decisionEvent(store, "b", "missing", "task_transitioned", 2, "", []string{update.EventID}, "task", map[string]any{"status": "completed", "evidence_digests": []string{"absent"}})
	store.Events[missing.EventID] = missing
	state, err := Reduce(store)
	if err != nil {
		t.Fatal(err)
	}
	kinds := map[string]bool{}
	for _, conflict := range state.Conflicts {
		kinds[conflict.Kind] = true
	}
	if !kinds["stale_evidence"] || !kinds["evidence_unavailable"] {
		t.Fatalf("conflicts = %#v", state.Conflicts)
	}
}

func TestDecisionTableDivergentModesOwnershipAndTakeoversConflict(t *testing.T) {
	store, create := decisionStore(t)
	modeA := decisionEvent(store, "a", "mode-a", "mode_changed", 0, "", []string{create.EventID}, "", map[string]any{"new_mode": "collaborative"})
	modeB := decisionEvent(store, "b", "mode-b", "mode_changed", 0, "", []string{create.EventID}, "", map[string]any{"new_mode": "solo"})
	store.Events[modeA.EventID] = modeA
	store.Events[modeB.EventID] = modeB
	claimA := decisionEvent(store, "a", "claim-a", "ownership_claimed", 1, modeA.EventID, nil, "task", map[string]any{"owner_session": "a"})
	claimA.Sequence = 2
	claimA.Digest, _ = DigestRecord(claimA)
	claimB := decisionEvent(store, "b", "claim-b", "ownership_claimed", 1, modeB.EventID, nil, "task", map[string]any{"owner_session": "b"})
	claimB.Sequence = 2
	claimB.Digest, _ = DigestRecord(claimB)
	store.Events[claimA.EventID] = claimA
	store.Events[claimB.EventID] = claimB
	state, err := Reduce(store)
	if err != nil {
		t.Fatal(err)
	}
	kinds := map[string]bool{}
	for _, conflict := range state.Conflicts {
		kinds[conflict.Kind] = true
	}
	if !kinds["mode_change"] || !kinds["ownership"] {
		t.Fatalf("conflicts = %#v", state.Conflicts)
	}
}

func TestDecisionTableConcurrentResolutionsConflict(t *testing.T) {
	store, create := decisionStore(t)
	left := decisionEvent(store, "a", "out-a", "task_transitioned", 1, "", []string{create.EventID}, "task", map[string]any{"status": "completed"})
	right := decisionEvent(store, "b", "out-b", "task_transitioned", 1, "", []string{create.EventID}, "task", map[string]any{"status": "blocked"})
	store.Events[left.EventID] = left
	store.Events[right.EventID] = right
	state, err := Reduce(store)
	if err != nil {
		t.Fatal(err)
	}
	conflict := state.Conflicts[0]
	frontier := state.Frontier
	store.Sessions["c"] = decisionSession(store.Profile, "c")
	store.Sessions["d"] = decisionSession(store.Profile, "d")
	r1 := decisionEvent(store, "c", "resolve-a", "conflict_resolved", 0, "", []string{left.EventID, right.EventID}, "", map[string]any{"conflict_id": conflict.ID, "heads": conflict.Heads, "selected_event": left.EventID, "expected_frontier": frontier})
	r2 := decisionEvent(store, "d", "resolve-b", "conflict_resolved", 0, "", []string{left.EventID, right.EventID}, "", map[string]any{"conflict_id": conflict.ID, "heads": conflict.Heads, "selected_event": right.EventID, "expected_frontier": frontier})
	store.Events[r1.EventID] = r1
	store.Events[r2.EventID] = r2
	state, err = Reduce(store)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, item := range state.Conflicts {
		if item.Kind == "concurrent_resolution" {
			found = true
		}
	}
	if !found {
		t.Fatalf("conflicts = %#v", state.Conflicts)
	}
}

func TestDecisionTableConcurrentPolicyChangesAndTakeoversConflict(t *testing.T) {
	store, create := decisionStore(t)
	policyA := decisionEvent(store, "a", "policy-a", "policy_changed", 0, "", []string{create.EventID}, "", map[string]any{"new_revision": "policy-a", "intent_digest": "intent-a", "constraints_digest": "constraints"})
	policyB := decisionEvent(store, "b", "policy-b", "policy_changed", 0, "", []string{create.EventID}, "", map[string]any{"new_revision": "policy-b", "intent_digest": "intent-b", "constraints_digest": "constraints"})
	store.Events[policyA.EventID] = policyA
	store.Events[policyB.EventID] = policyB
	takeoverA := decisionEvent(store, "a", "takeover-a", "ownership_taken_over", 0, policyA.EventID, nil, "", map[string]any{"scope": "session", "prior_session": "base", "owner_session": "a"})
	takeoverB := decisionEvent(store, "b", "takeover-b", "ownership_taken_over", 0, policyB.EventID, nil, "", map[string]any{"scope": "session", "prior_session": "base", "owner_session": "b"})
	store.Events[takeoverA.EventID] = takeoverA
	store.Events[takeoverB.EventID] = takeoverB
	state, err := Reduce(store)
	if err != nil {
		t.Fatal(err)
	}
	kinds := map[string]bool{}
	for _, conflict := range state.Conflicts {
		kinds[conflict.Kind] = true
	}
	if !kinds["policy_change"] || !kinds["session_takeover"] {
		t.Fatalf("conflicts = %#v", state.Conflicts)
	}
}

func decisionStore(t *testing.T) (*Store, Event) {
	t.Helper()
	profile := Profile{SmallVersion: ProfileVersion, ProjectID: "decision-project", LineageID: "decision-lineage", Mode: "collaborative", PolicyRevision: "policy-1"}
	store := &Store{Profile: profile, Sessions: map[string]Session{}, Events: map[string]Event{}, Receipts: map[string]Receipt{}}
	for _, id := range []string{"base", "a", "b"} {
		store.Sessions[id] = decisionSession(profile, id)
	}
	create := decisionEvent(store, "base", "create", "task_created", 1, "", nil, "task", map[string]any{"alias": "task-1", "title": "shared"})
	store.Events[create.EventID] = create
	return store, create
}
func decisionSession(profile Profile, id string) Session {
	session := Session{SmallVersion: ProfileVersion, ProjectID: profile.ProjectID, LineageID: profile.LineageID, SessionID: id, ObservedFrontier: Frontier(nil), ToolVersion: "test", CreatedAt: "2026-01-01T00:00:00Z"}
	session.Digest, _ = DigestRecord(session)
	return session
}
func decisionEvent(store *Store, session, id, kind string, revision int64, previous string, parents []string, task string, payload map[string]any) Event {
	sequence := int64(1)
	if previous != "" {
		sequence = store.Events[previous].Sequence + 1
	}
	event := Event{SmallVersion: ProfileVersion, ProjectID: store.Profile.ProjectID, LineageID: store.Profile.LineageID, SessionID: session, EventID: id, Kind: kind, Sequence: sequence, PreviousEvent: previous, Parents: parents, OccurredAt: "2026-01-01T00:00:00Z", TaskID: task, TaskRevision: revision, PolicyRevision: store.Profile.PolicyRevision, Payload: payload}
	event.Digest, _ = DigestRecord(event)
	return event
}
