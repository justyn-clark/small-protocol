package sessionv2

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
)

func Reduce(store *Store) (State, error) {
	if err := Validate(store); err != nil {
		return State{}, err
	}
	state := State{
		Profile: store.Profile, Frontier: Frontier(store.Events), EventCount: len(store.Events),
		SessionCount: len(store.Sessions), Tasks: map[string]TaskState{}, ClosedSessions: map[string]bool{}, SupersededSessions: map[string]bool{},
	}
	ordered, err := topologicalEvents(store.Events)
	if err != nil {
		return State{}, err
	}
	ancestry := newAncestryIndex(store.Events)
	definitions := map[string][]Event{}
	updates := map[string][]Event{}
	transitions := map[string][]Event{}
	modes := []Event{}
	policies := []Event{}
	claims := map[string][]Event{}
	sessionTakeovers := map[string][]Event{}
	resolutions := map[string][]Event{}
	conflicts := map[string]Conflict{}

	for _, event := range ordered {
		switch event.Kind {
		case "session_closed":
			state.ClosedSessions[event.SessionID] = true
		case "handoff_recorded":
			state.Handoffs = append(state.Handoffs, event)
		case "task_created":
			definitions[event.TaskID] = append(definitions[event.TaskID], event)
		case "task_updated":
			updates[event.TaskID] = append(updates[event.TaskID], event)
		case "task_transitioned":
			transitions[event.TaskID] = append(transitions[event.TaskID], event)
		case "mode_changed":
			modes = append(modes, event)
		case "policy_changed":
			policies = append(policies, event)
		case "ownership_claimed", "ownership_taken_over":
			if event.TaskID == "" && event.Kind == "ownership_taken_over" && stringPayload(event.Payload, "scope") == "session" {
				prior := stringPayload(event.Payload, "prior_session")
				state.SupersededSessions[prior] = true
				sessionTakeovers[prior] = append(sessionTakeovers[prior], event)
			} else {
				claims[event.TaskID] = append(claims[event.TaskID], event)
			}
		case "conflict_resolved":
			conflictID := stringPayload(event.Payload, "conflict_id")
			resolutions[conflictID] = append(resolutions[conflictID], event)
		}
	}

	for taskID, defs := range definitions {
		if taskID == "" {
			addConflict(conflicts, "task_identity", "", eventIDs(defs), "task_created event is missing task identity")
			continue
		}
		selected := defs[0]
		for _, candidate := range defs[1:] {
			if semanticDefinition(candidate) != semanticDefinition(selected) && concurrent(selected, candidate, ancestry) {
				addConflict(conflicts, "task_definition", taskID, eventIDs(defs), "concurrent task definitions differ")
			}
			if ancestry.isAncestor(selected.EventID, candidate.EventID) {
				selected = candidate
			}
		}
		state.Tasks[taskID] = taskFromDefinition(selected)
	}

	for taskID, taskUpdates := range updates {
		task, exists := state.Tasks[taskID]
		if !exists {
			addConflict(conflicts, "missing_task", taskID, eventIDs(taskUpdates), "task update has no task_created event")
			continue
		}
		for _, update := range taskUpdates {
			if update.TaskRevision <= task.Revision {
				addConflict(conflicts, "stale_task_revision", taskID, []string{update.EventID}, "task update does not advance the observed revision")
				continue
			}
			for _, other := range taskUpdates {
				if other.EventID != update.EventID && other.TaskRevision == update.TaskRevision && concurrent(update, other, ancestry) && semanticDefinition(update) != semanticDefinition(other) {
					addConflict(conflicts, "task_update", taskID, []string{update.EventID, other.EventID}, "concurrent task updates differ")
				}
			}
			if update.TaskRevision >= task.Revision {
				applyTaskDefinition(&task, update)
			}
		}
		state.Tasks[taskID] = task
	}

	for taskID, claimsForTask := range transitions {
		task, exists := state.Tasks[taskID]
		if !exists {
			addConflict(conflicts, "missing_task", taskID, eventIDs(claimsForTask), "task transition has no task_created event")
			continue
		}
		terminal := []Event{}
		for _, claim := range claimsForTask {
			if claim.TaskRevision != task.Revision || (claim.PolicyRevision != "" && claim.PolicyRevision != task.PolicyRevision) {
				addConflict(conflicts, "stale_evidence", taskID, []string{claim.EventID}, "transition evidence targets a stale task or policy revision")
				continue
			}
			for _, digest := range stringSlicePayload(claim.Payload, "evidence_digests") {
				receipt, ok := store.Receipts[digest]
				if !ok || receipt.Availability != "verified" {
					addConflict(conflicts, "evidence_unavailable", taskID, []string{claim.EventID}, "referenced portable evidence is missing or not verified")
				}
			}
			terminal = append(terminal, claim)
		}
		if len(terminal) == 0 {
			state.Tasks[taskID] = task
			continue
		}
		selected := terminal[0]
		for _, candidate := range terminal[1:] {
			if ancestry.isAncestor(selected.EventID, candidate.EventID) {
				selected = candidate
				continue
			}
			if ancestry.isAncestor(candidate.EventID, selected.EventID) {
				continue
			}
			if semanticTransition(candidate) != semanticTransition(selected) {
				addConflict(conflicts, "task_outcome", taskID, []string{selected.EventID, candidate.EventID}, "concurrent task outcomes are not semantically equivalent")
			}
		}
		task.Status = stringPayload(selected.Payload, "status")
		task.SourceDigest = firstNonEmpty(selected.SourceDigest, stringPayload(selected.Payload, "candidate_digest"))
		task.EvidenceDigests = stringSlicePayload(selected.Payload, "evidence_digests")
		task.Attestations = eventIDs(terminal)
		state.Tasks[taskID] = task
	}

	mode := store.Profile.Mode
	if len(modes) > 0 {
		selected := modes[0]
		for _, candidate := range modes[1:] {
			if concurrent(selected, candidate, ancestry) && stringPayload(selected.Payload, "new_mode") != stringPayload(candidate.Payload, "new_mode") {
				addConflict(conflicts, "mode_change", "", []string{selected.EventID, candidate.EventID}, "concurrent mode changes differ")
			}
			if ancestry.isAncestor(selected.EventID, candidate.EventID) {
				selected = candidate
			}
		}
		mode = stringPayload(selected.Payload, "new_mode")
	}
	state.Profile.Mode = mode

	var selectedPolicy *Event
	if len(policies) > 0 {
		selected := policies[0]
		for _, candidate := range policies[1:] {
			if concurrent(selected, candidate, ancestry) && semanticPolicy(candidate) != semanticPolicy(selected) {
				addConflict(conflicts, "policy_change", "", []string{selected.EventID, candidate.EventID}, "concurrent project policy revisions differ")
			}
			if ancestry.isAncestor(selected.EventID, candidate.EventID) {
				selected = candidate
			}
		}
		selectedPolicy = &selected
		state.Profile.PolicyRevision = stringPayload(selected.Payload, "new_revision")
	}

	for taskID, taskClaims := range claims {
		active := []Event{}
		for _, claim := range taskClaims {
			if stringPayload(claim.Payload, "action") != "release" {
				active = append(active, claim)
			}
		}
		for i := range active {
			for j := i + 1; j < len(active); j++ {
				if concurrent(active[i], active[j], ancestry) && stringPayload(active[i].Payload, "owner_session") != stringPayload(active[j].Payload, "owner_session") {
					addConflict(conflicts, "ownership", taskID, []string{active[i].EventID, active[j].EventID}, "competing advisory ownership claims")
				}
			}
		}
	}
	for prior, takeovers := range sessionTakeovers {
		for i := range takeovers {
			for j := i + 1; j < len(takeovers); j++ {
				if concurrent(takeovers[i], takeovers[j], ancestry) && stringPayload(takeovers[i].Payload, "owner_session") != stringPayload(takeovers[j].Payload, "owner_session") {
					addConflict(conflicts, "session_takeover", prior, []string{takeovers[i].EventID, takeovers[j].EventID}, "competing session takeovers require explicit resolution")
				}
			}
		}
	}

	selectedResolutions := applyResolutions(conflicts, resolutions, store.Events, ancestry)
	for conflictID, resolution := range selectedResolutions {
		conflict := conflicts[conflictID]
		selected := store.Events[stringPayload(resolution.Payload, "selected_event")]
		switch conflict.Kind {
		case "task_definition", "task_update":
			task := state.Tasks[conflict.TaskID]
			applyTaskDefinition(&task, selected)
			state.Tasks[conflict.TaskID] = task
		case "task_outcome":
			task := state.Tasks[conflict.TaskID]
			task.Status = stringPayload(selected.Payload, "status")
			task.SourceDigest = firstNonEmpty(selected.SourceDigest, stringPayload(selected.Payload, "candidate_digest"))
			task.EvidenceDigests = stringSlicePayload(selected.Payload, "evidence_digests")
			state.Tasks[conflict.TaskID] = task
		case "mode_change":
			state.Profile.Mode = stringPayload(selected.Payload, "new_mode")
		case "policy_change":
			state.Profile.PolicyRevision = stringPayload(selected.Payload, "new_revision")
			copy := selected
			selectedPolicy = &copy
		}
	}
	expectedIntent := store.Profile.IntentDigest
	expectedConstraints := store.Profile.ConstraintsDigest
	policyHead := "profile:" + store.Profile.PolicyRevision
	if selectedPolicy != nil {
		expectedIntent = stringPayload(selectedPolicy.Payload, "intent_digest")
		expectedConstraints = stringPayload(selectedPolicy.Payload, "constraints_digest")
		policyHead = selectedPolicy.EventID
	}
	if expectedIntent != "" && store.PolicyIntentDigest != "" && expectedIntent != store.PolicyIntentDigest {
		addConflict(conflicts, "policy_material", "", []string{policyHead}, "tracked intent does not match the selected policy revision")
	}
	if expectedConstraints != "" && store.PolicyConstraintsDigest != "" && expectedConstraints != store.PolicyConstraintsDigest {
		addConflict(conflicts, "policy_material", "", []string{policyHead}, "tracked constraints do not match the selected policy revision")
	}
	state.Conflicts = make([]Conflict, 0, len(conflicts))
	for _, conflict := range conflicts {
		if conflict.ResolvedBy == "" {
			state.Conflicts = append(state.Conflicts, conflict)
		}
	}
	sort.Slice(state.Conflicts, func(i, j int) bool { return state.Conflicts[i].ID < state.Conflicts[j].ID })
	sort.Slice(state.Handoffs, func(i, j int) bool { return state.Handoffs[i].EventID < state.Handoffs[j].EventID })
	return state, nil
}

func Strict(store *Store) (State, error) {
	state, err := Reduce(store)
	if err != nil {
		return State{}, err
	}
	if len(state.Conflicts) > 0 {
		return state, fmt.Errorf("%w: %d conflict(s)", ErrConflict, len(state.Conflicts))
	}
	return state, nil
}

func applyResolutions(conflicts map[string]Conflict, resolutions map[string][]Event, events map[string]Event, ancestry *ancestryIndex) map[string]Event {
	selected := map[string]Event{}
	for conflictID, options := range resolutions {
		conflict, exists := conflicts[conflictID]
		if !exists || len(options) == 0 {
			continue
		}
		valid := []Event{}
		for _, resolution := range options {
			if !sameStringSet(conflict.Heads, stringSlicePayload(resolution.Payload, "heads")) {
				continue
			}
			expected := stringPayload(resolution.Payload, "expected_frontier")
			without := map[string]Event{}
			for id, event := range events {
				if event.Kind == "conflict_resolved" && stringPayload(event.Payload, "conflict_id") == conflictID && concurrent(event, resolution, ancestry) {
					continue
				}
				if id != resolution.EventID && !ancestry.isAncestor(resolution.EventID, id) {
					without[id] = event
				}
			}
			if expected == "" || expected != Frontier(without) {
				continue
			}
			valid = append(valid, resolution)
		}
		if len(valid) == 1 {
			conflict.ResolvedBy = valid[0].EventID
			conflicts[conflictID] = conflict
			selected[conflictID] = valid[0]
			continue
		}
		if len(valid) > 1 {
			unrelated := false
			for i := range valid {
				for j := i + 1; j < len(valid); j++ {
					if concurrent(valid[i], valid[j], ancestry) {
						unrelated = true
					}
				}
			}
			if unrelated {
				addConflict(conflicts, "concurrent_resolution", "", eventIDs(valid), "concurrent resolutions require another explicit resolution")
			}
		}
	}
	return selected
}

func topologicalEvents(events map[string]Event) ([]Event, error) {
	indegree := map[string]int{}
	children := map[string][]string{}
	for id := range events {
		indegree[id] = 0
	}
	for id, event := range events {
		parents := uniqueNonEmpty(append(append([]string{}, event.Parents...), event.PreviousEvent))
		indegree[id] = len(parents)
		for _, parent := range parents {
			children[parent] = append(children[parent], id)
		}
	}
	ready := []string{}
	for id, degree := range indegree {
		if degree == 0 {
			ready = append(ready, id)
		}
	}
	sort.Strings(ready)
	ordered := make([]Event, 0, len(events))
	for len(ready) > 0 {
		id := ready[0]
		ready = ready[1:]
		ordered = append(ordered, events[id])
		for _, child := range children[id] {
			indegree[child]--
			if indegree[child] == 0 {
				ready = append(ready, child)
				sort.Strings(ready)
			}
		}
	}
	if len(ordered) != len(events) {
		return nil, fmt.Errorf("event graph contains a cycle")
	}
	return ordered, nil
}

type ancestryIndex struct {
	events map[string]Event
	memo   map[string]bool
}

func newAncestryIndex(events map[string]Event) *ancestryIndex {
	return &ancestryIndex{events: events, memo: map[string]bool{}}
}
func (index *ancestryIndex) isAncestor(ancestor, descendant string) bool {
	if ancestor == "" || descendant == "" || ancestor == descendant {
		return false
	}
	key := ancestor + "\x00" + descendant
	if value, ok := index.memo[key]; ok {
		return value
	}
	stack := []string{descendant}
	seen := map[string]bool{}
	for len(stack) > 0 {
		id := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if seen[id] {
			continue
		}
		seen[id] = true
		event, ok := index.events[id]
		if !ok {
			continue
		}
		for _, parent := range uniqueNonEmpty(append(append([]string{}, event.Parents...), event.PreviousEvent)) {
			if parent == ancestor {
				index.memo[key] = true
				return true
			}
			if !seen[parent] {
				stack = append(stack, parent)
			}
		}
	}
	index.memo[key] = false
	return false
}
func concurrent(a, b Event, ancestry *ancestryIndex) bool {
	return a.EventID != b.EventID && !ancestry.isAncestor(a.EventID, b.EventID) && !ancestry.isAncestor(b.EventID, a.EventID)
}

func taskFromDefinition(event Event) TaskState {
	task := TaskState{ID: event.TaskID, Revision: event.TaskRevision, Status: "pending"}
	applyTaskDefinition(&task, event)
	return task
}

func applyTaskDefinition(task *TaskState, event Event) {
	task.Revision = event.TaskRevision
	task.Alias = stringPayload(event.Payload, "alias")
	task.Title = stringPayload(event.Payload, "title")
	task.Acceptance = stringSlicePayload(event.Payload, "acceptance")
	task.Dependencies = stringSlicePayload(event.Payload, "dependencies")
	task.PolicyRevision = firstNonEmpty(event.PolicyRevision, stringPayload(event.Payload, "policy_revision"))
	if source := firstNonEmpty(event.SourceDigest, stringPayload(event.Payload, "source_digest")); source != "" {
		task.SourceDigest = source
	}
}

func semanticDefinition(event Event) string {
	parts := []string{event.TaskID, fmt.Sprint(event.TaskRevision), stringPayload(event.Payload, "title"), strings.Join(stringSlicePayload(event.Payload, "acceptance"), "\x00"), strings.Join(stringSlicePayload(event.Payload, "dependencies"), "\x00"), event.PolicyRevision, event.SourceDigest}
	return strings.Join(parts, "\x01")
}

func semanticTransition(event Event) string {
	evidence := stringSlicePayload(event.Payload, "evidence_digests")
	sort.Strings(evidence)
	return strings.Join([]string{event.TaskID, fmt.Sprint(event.TaskRevision), stringPayload(event.Payload, "status"), event.PolicyRevision, firstNonEmpty(event.SourceDigest, stringPayload(event.Payload, "candidate_digest")), strings.Join(evidence, ",")}, "\x01")
}

func semanticPolicy(event Event) string {
	return strings.Join([]string{stringPayload(event.Payload, "new_revision"), stringPayload(event.Payload, "intent_digest"), stringPayload(event.Payload, "constraints_digest")}, "\x01")
}

func addConflict(target map[string]Conflict, kind, taskID string, heads []string, message string) {
	heads = uniqueNonEmpty(heads)
	sort.Strings(heads)
	id := conflictID(kind, taskID, heads)
	target[id] = Conflict{ID: id, Kind: kind, TaskID: taskID, Heads: heads, Message: message}
}

func conflictID(kind, taskID string, heads []string) string {
	sorted := append([]string{}, heads...)
	sort.Strings(sorted)
	sum := sha256.Sum256([]byte(kind + "\n" + taskID + "\n" + strings.Join(sorted, "\n")))
	return "conflict_" + hex.EncodeToString(sum[:16])
}

func eventIDs(events []Event) []string {
	ids := make([]string, 0, len(events))
	for _, event := range events {
		ids = append(ids, event.EventID)
	}
	return uniqueNonEmpty(ids)
}
func stringPayload(payload map[string]any, key string) string {
	value, _ := payload[key].(string)
	return strings.TrimSpace(value)
}
func stringSlicePayload(payload map[string]any, key string) []string {
	raw, _ := payload[key].([]any)
	if typed, ok := payload[key].([]string); ok {
		return append([]string{}, typed...)
	}
	values := []string{}
	for _, item := range raw {
		if value, ok := item.(string); ok && strings.TrimSpace(value) != "" {
			values = append(values, strings.TrimSpace(value))
		}
	}
	return values
}
func uniqueNonEmpty(values []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, value := range values {
		if value != "" && !seen[value] {
			seen[value] = true
			out = append(out, value)
		}
	}
	return out
}
func sameStringSet(a, b []string) bool {
	a = uniqueNonEmpty(a)
	b = uniqueNonEmpty(b)
	sort.Strings(a)
	sort.Strings(b)
	return strings.Join(a, "\x00") == strings.Join(b, "\x00")
}
func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
