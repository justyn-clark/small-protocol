package sessionv2

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/justyn-clark/small-protocol/internal/small"
)

type SessionStartOptions struct {
	Label, ActorLabel, ModelLabel, ParentSessionID, FromHandoff, ToolVersion string
	TakeoverSessionID, TakeoverReason                                        string
}

type AppendOptions struct {
	TaskID, PolicyRevision, SourceDigest, ExpectedFrontier string
	TaskRevision                                           int64
	Parents                                                []string
	OccurredAt                                             time.Time
}

func Initialize(baseDir string, profile Profile, intent, constraints []byte) error {
	if profile.Mode == "" {
		profile.Mode = "solo"
	}
	if profile.IntentDigest == "" {
		profile.IntentDigest = "absent"
		if intent != nil {
			profile.IntentDigest = PolicyMaterialDigest(intent)
		}
	}
	if profile.ConstraintsDigest == "" {
		profile.ConstraintsDigest = "absent"
		if constraints != nil {
			profile.ConstraintsDigest = PolicyMaterialDigest(constraints)
		}
	}
	if err := validateProfile(profile); err != nil {
		return err
	}
	profileData, err := CanonicalJSON(profile)
	if err != nil {
		return err
	}
	profileData = append(profileData, '\n')
	files := []small.StateFile{{Path: ".small/profile.json", Data: profileData}}
	if intent != nil {
		files = append(files, small.StateFile{Path: ".small/policy/intent.small.yml", Data: intent})
	}
	if constraints != nil {
		files = append(files, small.StateFile{Path: ".small/policy/constraints.small.yml", Data: constraints})
	}
	_, err = small.WriteStateFiles(baseDir, files)
	return err
}

func StartSession(baseDir string, options SessionStartOptions) (Session, []Event, error) {
	var created Session
	var events []Event
	err := small.WithStateLock(baseDir, func() error {
		store, err := Load(baseDir)
		if err != nil {
			return err
		}
		state, err := Reduce(store)
		if err != nil {
			return err
		}
		active := ActiveSessions(store, state)
		if options.ParentSessionID != "" {
			if _, ok := store.Sessions[options.ParentSessionID]; !ok {
				return fmt.Errorf("parent session %s not found", options.ParentSessionID)
			}
		}
		if options.FromHandoff != "" {
			event, ok := store.Events[options.FromHandoff]
			if !ok || event.Kind != "handoff_recorded" {
				return fmt.Errorf("handoff event %s not found", options.FromHandoff)
			}
		}
		if state.Profile.Mode == "solo" && len(active) > 0 {
			if options.TakeoverSessionID == "" || strings.TrimSpace(options.TakeoverReason) == "" {
				return fmt.Errorf("solo mode already has active session %s; close it or use --takeover with --reason", strings.Join(active, ","))
			}
			found := false
			for _, id := range active {
				if id == options.TakeoverSessionID {
					found = true
				}
			}
			if !found {
				return fmt.Errorf("takeover target %s is not an active session", options.TakeoverSessionID)
			}
		}
		sessionID, err := NewID("session")
		if err != nil {
			return err
		}
		created = Session{SmallVersion: ProfileVersion, ProjectID: store.Profile.ProjectID, LineageID: store.Profile.LineageID, SessionID: sessionID, Label: strings.TrimSpace(options.Label), ActorLabel: strings.TrimSpace(options.ActorLabel), ModelLabel: strings.TrimSpace(options.ModelLabel), ParentSessionID: strings.TrimSpace(options.ParentSessionID), FromHandoff: strings.TrimSpace(options.FromHandoff), ObservedFrontier: state.Frontier, ToolVersion: options.ToolVersion, CreatedAt: time.Now().UTC().Format(time.RFC3339Nano)}
		created.Digest, err = DigestRecord(created)
		if err != nil {
			return err
		}
		firstID, err := NewID("event")
		if err != nil {
			return err
		}
		first := Event{SmallVersion: ProfileVersion, ProjectID: store.Profile.ProjectID, LineageID: store.Profile.LineageID, SessionID: sessionID, EventID: firstID, Kind: "session_started", Sequence: 1, Parents: EventHeads(store.Events), OccurredAt: time.Now().UTC().Format(time.RFC3339Nano), Payload: map[string]any{"label": created.Label, "observed_frontier": state.Frontier}}
		first.Digest, err = DigestRecord(first)
		if err != nil {
			return err
		}
		events = append(events, first)
		if options.TakeoverSessionID != "" {
			takeoverID, idErr := NewID("event")
			if idErr != nil {
				return idErr
			}
			takeover := Event{SmallVersion: ProfileVersion, ProjectID: store.Profile.ProjectID, LineageID: store.Profile.LineageID, SessionID: sessionID, EventID: takeoverID, Kind: "ownership_taken_over", Sequence: 2, PreviousEvent: first.EventID, OccurredAt: time.Now().UTC().Format(time.RFC3339Nano), Payload: map[string]any{"scope": "session", "prior_session": options.TakeoverSessionID, "owner_session": sessionID, "reason": strings.TrimSpace(options.TakeoverReason)}}
			takeover.Digest, err = DigestRecord(takeover)
			if err != nil {
				return err
			}
			events = append(events, takeover)
		}
		files := []small.StateFile{{Path: filepath.Join(".small", "sessions", created.SessionID+".json"), Data: append(mustCanonical(created), '\n')}}
		for _, event := range events {
			files = append(files, small.StateFile{Path: filepath.Join(".small", "events", event.SessionID, event.EventID+".json"), Data: append(mustCanonical(event), '\n')})
		}
		_, err = small.WriteStateFilesLocked(baseDir, files)
		return err
	})
	if err != nil {
		return Session{}, nil, err
	}
	if err := writeActiveSession(baseDir, created.SessionID); err != nil {
		return Session{}, nil, fmt.Errorf("session recorded but local selection failed: %w", err)
	}
	return created, events, nil
}

func AppendEvent(baseDir, sessionID, kind string, payload map[string]any, options AppendOptions) (Event, error) {
	created, _, err := AppendEventWithReceipts(baseDir, sessionID, kind, payload, options, nil)
	return created, err
}

// AppendEventWithReceipts publishes an event and its content-addressed evidence
// receipts in one recoverable transaction under the local writer lock.
func AppendEventWithReceipts(baseDir, sessionID, kind string, payload map[string]any, options AppendOptions, receipts []Receipt) (Event, []Receipt, error) {
	var created Event
	prepared := make([]Receipt, 0, len(receipts))
	err := small.WithStateLock(baseDir, func() error {
		store, err := Load(baseDir)
		if err != nil {
			return err
		}
		state, err := Reduce(store)
		if err != nil {
			return err
		}
		if options.ExpectedFrontier != "" && options.ExpectedFrontier != state.Frontier {
			return fmt.Errorf("%w: expected %s, actual %s", ErrStaleFrontier, options.ExpectedFrontier, state.Frontier)
		}
		if _, ok := store.Sessions[sessionID]; !ok {
			return fmt.Errorf("session %s not found", sessionID)
		}
		if state.ClosedSessions[sessionID] || state.SupersededSessions[sessionID] {
			return fmt.Errorf("session %s is not active", sessionID)
		}
		sequence, previous := nextSessionSequence(store.Events, sessionID)
		eventID, err := NewID("event")
		if err != nil {
			return err
		}
		occurred := options.OccurredAt
		if occurred.IsZero() {
			occurred = time.Now().UTC()
		}
		created = Event{SmallVersion: ProfileVersion, ProjectID: store.Profile.ProjectID, LineageID: store.Profile.LineageID, SessionID: sessionID, EventID: eventID, Kind: kind, Sequence: sequence, PreviousEvent: previous, Parents: uniqueStrings(append(EventHeads(store.Events), options.Parents...)), OccurredAt: occurred.UTC().Format(time.RFC3339Nano), TaskID: strings.TrimSpace(options.TaskID), TaskRevision: options.TaskRevision, PolicyRevision: firstValue(options.PolicyRevision, state.Profile.PolicyRevision), SourceDigest: strings.TrimSpace(options.SourceDigest), Payload: payload}
		created.Parents = removeString(created.Parents, created.PreviousEvent)
		created.Digest, err = DigestRecord(created)
		if err != nil {
			return err
		}
		files := []small.StateFile{{Path: filepath.Join(".small", "events", created.SessionID, created.EventID+".json"), Data: append(mustCanonical(created), '\n')}}
		for _, receipt := range receipts {
			if receipt.ReceiptID == "" {
				receipt.ReceiptID, err = NewID("receipt")
				if err != nil {
					return err
				}
			}
			receipt.SmallVersion = ProfileVersion
			receipt.ProjectID = store.Profile.ProjectID
			receipt.Digest, err = DigestRecord(receipt)
			if err != nil {
				return err
			}
			if err := validateReceipt(store.Profile, receipt); err != nil {
				return err
			}
			prepared = append(prepared, receipt)
			files = append(files, small.StateFile{Path: filepath.Join(".small", "receipts", "sha256", receipt.Digest+".json"), Data: append(mustCanonical(receipt), '\n')})
		}
		_, err = small.WriteStateFilesLocked(baseDir, files)
		return err
	})
	return created, prepared, err
}

type PolicyMaterial struct {
	Intent, Constraints       []byte
	HasIntent, HasConstraints bool
}

func RevisePolicy(baseDir, sessionID, expectedFrontier, reason, requester string, material PolicyMaterial) (Event, error) {
	var created Event
	err := small.WithStateLock(baseDir, func() error {
		store, err := Load(baseDir)
		if err != nil {
			return err
		}
		state, err := Strict(store)
		if err != nil {
			return err
		}
		if expectedFrontier == "" || state.Frontier != expectedFrontier {
			return fmt.Errorf("%w: expected %s, actual %s", ErrStaleFrontier, expectedFrontier, state.Frontier)
		}
		if strings.TrimSpace(reason) == "" {
			return fmt.Errorf("policy revision reason is required")
		}
		if _, ok := store.Sessions[sessionID]; !ok || state.ClosedSessions[sessionID] || state.SupersededSessions[sessionID] {
			return fmt.Errorf("session %s is not active", sessionID)
		}
		intentDigest := "absent"
		if material.HasIntent {
			intentDigest = PolicyMaterialDigest(material.Intent)
		}
		constraintsDigest := "absent"
		if material.HasConstraints {
			constraintsDigest = PolicyMaterialDigest(material.Constraints)
		}
		newRevision := PolicyRevision(intentDigest, constraintsDigest)
		sequence, previous := nextSessionSequence(store.Events, sessionID)
		eventID, err := NewID("event")
		if err != nil {
			return err
		}
		created = Event{SmallVersion: ProfileVersion, ProjectID: store.Profile.ProjectID, LineageID: store.Profile.LineageID, SessionID: sessionID, EventID: eventID, Kind: "policy_changed", Sequence: sequence, PreviousEvent: previous, Parents: EventHeads(store.Events), OccurredAt: time.Now().UTC().Format(time.RFC3339Nano), PolicyRevision: state.Profile.PolicyRevision, Payload: map[string]any{"prior_revision": state.Profile.PolicyRevision, "new_revision": newRevision, "intent_digest": intentDigest, "constraints_digest": constraintsDigest, "reason": strings.TrimSpace(reason), "requester": strings.TrimSpace(requester)}}
		created.Parents = removeString(created.Parents, created.PreviousEvent)
		created.Digest, err = DigestRecord(created)
		if err != nil {
			return err
		}
		files := []small.StateFile{{Path: filepath.Join(".small", "events", sessionID, eventID+".json"), Data: append(mustCanonical(created), '\n')}}
		if material.HasIntent {
			files = append(files, small.StateFile{Path: filepath.Join(".small", "policy", "intent.small.yml"), Data: material.Intent})
		}
		if material.HasConstraints {
			files = append(files, small.StateFile{Path: filepath.Join(".small", "policy", "constraints.small.yml"), Data: material.Constraints})
		}
		_, err = small.WriteStateFilesLocked(baseDir, files)
		return err
	})
	return created, err
}

func CloseSession(baseDir, sessionID, summary string) ([]Event, error) {
	if strings.TrimSpace(summary) == "" {
		return nil, fmt.Errorf("session close summary must be non-whitespace")
	}
	store, err := Load(baseDir)
	if err != nil {
		return nil, err
	}
	frontier := Frontier(store.Events)
	handoff, err := AppendEvent(baseDir, sessionID, "handoff_recorded", map[string]any{"summary": strings.TrimSpace(summary), "authoritative": true}, AppendOptions{ExpectedFrontier: frontier})
	if err != nil {
		return nil, err
	}
	store, err = Load(baseDir)
	if err != nil {
		return nil, err
	}
	closed, err := AppendEvent(baseDir, sessionID, "session_closed", map[string]any{"handoff_event": handoff.EventID}, AppendOptions{ExpectedFrontier: Frontier(store.Events), Parents: []string{handoff.EventID}})
	if err != nil {
		return nil, err
	}
	_ = clearActiveSessionIf(baseDir, sessionID)
	return []Event{handoff, closed}, nil
}

func ResolveSession(baseDir, explicit string) (string, error) {
	if strings.TrimSpace(explicit) != "" {
		return strings.TrimSpace(explicit), nil
	}
	data, err := os.ReadFile(filepath.Join(baseDir, ".small-cache", "active-session"))
	if err != nil {
		return "", fmt.Errorf("%w: pass --session or run small session start", ErrAmbiguous)
	}
	id := strings.TrimSpace(string(data))
	if id == "" {
		return "", fmt.Errorf("%w: active session pointer is empty", ErrAmbiguous)
	}
	return id, nil
}

func ActiveSessions(store *Store, state State) []string {
	active := []string{}
	for id := range store.Sessions {
		if !state.ClosedSessions[id] && !state.SupersededSessions[id] {
			active = append(active, id)
		}
	}
	sort.Strings(active)
	return active
}

func EventHeads(events map[string]Event) []string {
	referenced := map[string]bool{}
	for _, event := range events {
		if event.PreviousEvent != "" {
			referenced[event.PreviousEvent] = true
		}
		for _, parent := range event.Parents {
			referenced[parent] = true
		}
	}
	heads := []string{}
	for id := range events {
		if !referenced[id] {
			heads = append(heads, id)
		}
	}
	sort.Strings(heads)
	return heads
}

func nextSessionSequence(events map[string]Event, sessionID string) (int64, string) {
	var max int64
	previous := ""
	for _, event := range events {
		if event.SessionID == sessionID && event.Sequence > max {
			max = event.Sequence
			previous = event.EventID
		}
	}
	return max + 1, previous
}

func writeActiveSession(baseDir, sessionID string) error {
	path := filepath.Join(baseDir, ".small-cache", "active-session")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(sessionID+"\n"), 0o600)
}
func clearActiveSessionIf(baseDir, sessionID string) error {
	path := filepath.Join(baseDir, ".small-cache", "active-session")
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if strings.TrimSpace(string(data)) == sessionID {
		return os.Remove(path)
	}
	return nil
}
func mustCanonical(value any) []byte {
	data, err := CanonicalJSON(value)
	if err != nil {
		panic(err)
	}
	return data
}
func uniqueStrings(values []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" && !seen[value] {
			seen[value] = true
			out = append(out, value)
		}
	}
	sort.Strings(out)
	return out
}
func removeString(values []string, target string) []string {
	out := values[:0]
	for _, value := range values {
		if value != target {
			out = append(out, value)
		}
	}
	return out
}
func firstValue(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
