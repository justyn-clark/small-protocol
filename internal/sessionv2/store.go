package sessionv2

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func IsWorkspace(baseDir string) bool {
	data, err := os.ReadFile(filepath.Join(baseDir, ".small", "profile.json"))
	if err != nil {
		return false
	}
	var marker struct {
		SmallVersion string `json:"small_version"`
	}
	return json.Unmarshal(data, &marker) == nil && marker.SmallVersion == ProfileVersion
}

func Load(baseDir string) (*Store, error) {
	profilePath := filepath.Join(baseDir, ".small", "profile.json")
	profileData, err := readRegularBounded(profilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrNotV2
		}
		return nil, err
	}
	var profile Profile
	if err := DecodeStrict(profileData, &profile); err != nil {
		return nil, fmt.Errorf("profile.json: %w", err)
	}
	if err := validateProfile(profile); err != nil {
		return nil, err
	}
	store := &Store{BaseDir: baseDir, Profile: profile, Sessions: map[string]Session{}, Events: map[string]Event{}, Receipts: map[string]Receipt{}}
	store.PolicyIntentDigest, err = policyFileDigest(filepath.Join(baseDir, ".small", "policy", "intent.small.yml"))
	if err != nil {
		return nil, fmt.Errorf("load policy intent: %w", err)
	}
	store.PolicyConstraintsDigest, err = policyFileDigest(filepath.Join(baseDir, ".small", "policy", "constraints.small.yml"))
	if err != nil {
		return nil, fmt.Errorf("load policy constraints: %w", err)
	}
	if err := loadJSONTree(filepath.Join(baseDir, ".small", "sessions"), func(path string, data []byte) error {
		var session Session
		if err := DecodeStrict(data, &session); err != nil {
			return err
		}
		if filepath.Base(path) != session.SessionID+".json" {
			return fmt.Errorf("session path does not match id %s", session.SessionID)
		}
		if err := validateSession(profile, session); err != nil {
			return err
		}
		if _, exists := store.Sessions[session.SessionID]; exists {
			return fmt.Errorf("duplicate session id %s", session.SessionID)
		}
		store.Sessions[session.SessionID] = session
		return nil
	}); err != nil {
		return nil, fmt.Errorf("load sessions: %w", err)
	}
	if err := loadJSONTree(filepath.Join(baseDir, ".small", "events"), func(path string, data []byte) error {
		var event Event
		if err := DecodeStrict(data, &event); err != nil {
			return err
		}
		if filepath.Base(path) != event.EventID+".json" || filepath.Base(filepath.Dir(path)) != event.SessionID {
			return fmt.Errorf("event path does not match session/event identity")
		}
		if existing, exists := store.Events[event.EventID]; exists {
			if existing.Digest != event.Digest {
				return fmt.Errorf("%w: event %s has digests %s and %s", ErrCorruption, event.EventID, existing.Digest, event.Digest)
			}
			return nil
		}
		store.Events[event.EventID] = event
		return nil
	}); err != nil {
		return nil, fmt.Errorf("load events: %w", err)
	}
	if err := loadJSONTree(filepath.Join(baseDir, ".small", "receipts", "sha256"), func(path string, data []byte) error {
		var receipt Receipt
		if err := DecodeStrict(data, &receipt); err != nil {
			return err
		}
		if filepath.Base(path) != receipt.Digest+".json" {
			return fmt.Errorf("receipt path does not match digest")
		}
		store.Receipts[receipt.Digest] = receipt
		return nil
	}); err != nil {
		return nil, fmt.Errorf("load receipts: %w", err)
	}
	if err := Validate(store); err != nil {
		return nil, err
	}
	return store, nil
}

func Validate(store *Store) error {
	if store == nil {
		return fmt.Errorf("nil v2 store")
	}
	for id, session := range store.Sessions {
		if err := validateSession(store.Profile, session); err != nil {
			return fmt.Errorf("session %s: %w", id, err)
		}
	}
	sequenceIndex := map[string]map[int64]string{}
	for id, event := range store.Events {
		if err := validateEvent(store.Profile, event); err != nil {
			return fmt.Errorf("event %s: %w", id, err)
		}
		if _, ok := store.Sessions[event.SessionID]; !ok {
			return fmt.Errorf("event %s references missing session %s", id, event.SessionID)
		}
		if sequenceIndex[event.SessionID] == nil {
			sequenceIndex[event.SessionID] = map[int64]string{}
		}
		if prior, exists := sequenceIndex[event.SessionID][event.Sequence]; exists {
			return fmt.Errorf("session chain fork: session %s sequence %d has events %s and %s", event.SessionID, event.Sequence, prior, id)
		}
		sequenceIndex[event.SessionID][event.Sequence] = id
		for _, parent := range append(append([]string{}, event.Parents...), event.PreviousEvent) {
			if parent == "" {
				continue
			}
			if _, ok := store.Events[parent]; !ok {
				return fmt.Errorf("event %s references missing parent %s", id, parent)
			}
		}
	}
	for sessionID, sequences := range sequenceIndex {
		for sequence, id := range sequences {
			event := store.Events[id]
			if sequence == 1 && event.PreviousEvent != "" {
				return fmt.Errorf("session %s first event has previous_event", sessionID)
			}
			if sequence > 1 {
				previousID, ok := sequences[sequence-1]
				if !ok || event.PreviousEvent != previousID {
					return fmt.Errorf("session %s sequence %d previous_event mismatch", sessionID, sequence)
				}
			}
		}
	}
	if err := validateAcyclic(store.Events); err != nil {
		return err
	}
	for digest, receipt := range store.Receipts {
		if err := validateReceipt(store.Profile, receipt); err != nil {
			return fmt.Errorf("receipt %s: %w", digest, err)
		}
	}
	return nil
}

func PublishSession(baseDir string, session Session) (bool, error) {
	digest, err := DigestRecord(session)
	if err != nil {
		return false, err
	}
	session.Digest = digest
	if err := validateSessionIdentity(session); err != nil {
		return false, err
	}
	return publishExclusive(filepath.Join(baseDir, ".small", "sessions", session.SessionID+".json"), session)
}

func PublishEvent(baseDir string, event Event) (bool, error) {
	digest, err := DigestRecord(event)
	if err != nil {
		return false, err
	}
	event.Digest = digest
	if err := validateEventIdentity(event); err != nil {
		return false, err
	}
	return publishExclusive(filepath.Join(baseDir, ".small", "events", event.SessionID, event.EventID+".json"), event)
}

func PublishReceipt(baseDir string, receipt Receipt) (bool, error) {
	if receipt.ReceiptID == "" {
		id, err := NewID("receipt")
		if err != nil {
			return false, err
		}
		receipt.ReceiptID = id
	}
	digest, err := DigestRecord(receipt)
	if err != nil {
		return false, err
	}
	receipt.Digest = digest
	return publishExclusive(filepath.Join(baseDir, ".small", "receipts", "sha256", digest+".json"), receipt)
}

func publishExclusive(path string, value any) (bool, error) {
	data, err := CanonicalJSON(value)
	if err != nil {
		return false, err
	}
	data = append(data, '\n')
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return false, err
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		if !os.IsExist(err) {
			return false, err
		}
		existing, readErr := readRegularBounded(path)
		if readErr != nil {
			return false, readErr
		}
		if strings.TrimSpace(string(existing)) == strings.TrimSpace(string(data)) {
			return false, nil
		}
		return false, fmt.Errorf("%w: immutable path %s already contains different bytes", ErrCorruption, path)
	}
	if _, err := file.Write(data); err != nil {
		_ = file.Close()
		return false, err
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		return false, err
	}
	if err := file.Close(); err != nil {
		return false, err
	}
	dir, err := os.Open(filepath.Dir(path))
	if err != nil {
		return false, err
	}
	defer dir.Close()
	if err := dir.Sync(); err != nil {
		return false, err
	}
	return true, nil
}

func Frontier(events map[string]Event) string {
	lines := make([]string, 0, len(events))
	for id, event := range events {
		lines = append(lines, id+":"+event.Digest)
	}
	sort.Strings(lines)
	sum := sha256.Sum256([]byte(strings.Join(lines, "\n")))
	return hex.EncodeToString(sum[:])
}

func PolicyMaterialDigest(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func PolicyRevision(intentDigest, constraintsDigest string) string {
	sum := sha256.Sum256([]byte("intent:" + intentDigest + "\nconstraints:" + constraintsDigest))
	return "policy_" + hex.EncodeToString(sum[:16])
}

func policyFileDigest(path string) (string, error) {
	data, err := readRegularBounded(path)
	if os.IsNotExist(err) {
		return "absent", nil
	}
	if err != nil {
		return "", err
	}
	return PolicyMaterialDigest(data), nil
}

func validateProfile(profile Profile) error {
	if profile.SmallVersion != ProfileVersion {
		return fmt.Errorf("unsupported profile %q", profile.SmallVersion)
	}
	if profile.ProjectID == "" || profile.LineageID == "" || profile.PolicyRevision == "" {
		return fmt.Errorf("profile identity and policy revision are required")
	}
	if profile.Mode != "solo" && profile.Mode != "collaborative" {
		return fmt.Errorf("invalid mode %q", profile.Mode)
	}
	return nil
}

func validateSession(profile Profile, session Session) error {
	if session.SmallVersion != ProfileVersion || session.ProjectID != profile.ProjectID || session.LineageID != profile.LineageID {
		return fmt.Errorf("session profile/project/lineage mismatch")
	}
	if err := validateSessionIdentity(session); err != nil {
		return err
	}
	want, err := DigestRecord(session)
	if err != nil {
		return err
	}
	if want != session.Digest {
		return fmt.Errorf("digest mismatch: want %s got %s", want, session.Digest)
	}
	return nil
}

func validateSessionIdentity(session Session) error {
	if session.SessionID == "" || session.ObservedFrontier == "" || session.ToolVersion == "" || session.CreatedAt == "" {
		return fmt.Errorf("session identity, frontier, tool version, and creation time are required")
	}
	return nil
}

func validateEvent(profile Profile, event Event) error {
	if event.SmallVersion != ProfileVersion || event.ProjectID != profile.ProjectID || event.LineageID != profile.LineageID {
		return fmt.Errorf("event profile/project/lineage mismatch")
	}
	if err := validateEventIdentity(event); err != nil {
		return err
	}
	want, err := DigestRecord(event)
	if err != nil {
		return err
	}
	if want != event.Digest {
		return fmt.Errorf("digest mismatch: want %s got %s", want, event.Digest)
	}
	return nil
}

func validateEventIdentity(event Event) error {
	if event.SessionID == "" || event.EventID == "" || event.Sequence < 1 || event.OccurredAt == "" || event.Payload == nil {
		return fmt.Errorf("event identity, sequence, occurrence annotation, and payload are required")
	}
	if !AllowedEventKinds[event.Kind] {
		return fmt.Errorf("unknown event kind %q", event.Kind)
	}
	if event.Sequence == 1 && event.PreviousEvent != "" {
		return fmt.Errorf("first event cannot have previous_event")
	}
	if event.Sequence > 1 && event.PreviousEvent == "" {
		return fmt.Errorf("sequence %d requires previous_event", event.Sequence)
	}
	return nil
}

func validateReceipt(profile Profile, receipt Receipt) error {
	if receipt.SmallVersion != ProfileVersion || receipt.ProjectID != profile.ProjectID {
		return fmt.Errorf("receipt profile/project mismatch")
	}
	if receipt.Strength != "narrative_assertion" && receipt.Strength != "cli_captured" && receipt.Strength != "verified_artifact" {
		return fmt.Errorf("invalid evidence strength %q", receipt.Strength)
	}
	if receipt.Availability != "unavailable" && receipt.Availability != "stale" && receipt.Availability != "failed" && receipt.Availability != "verified" {
		return fmt.Errorf("invalid evidence availability %q", receipt.Availability)
	}
	want, err := DigestRecord(receipt)
	if err != nil {
		return err
	}
	if want != receipt.Digest {
		return fmt.Errorf("digest mismatch")
	}
	return nil
}

func validateAcyclic(events map[string]Event) error {
	state := map[string]uint8{}
	var visit func(string) error
	visit = func(id string) error {
		if state[id] == 1 {
			return fmt.Errorf("event graph cycle at %s", id)
		}
		if state[id] == 2 {
			return nil
		}
		state[id] = 1
		event := events[id]
		for _, parent := range append(append([]string{}, event.Parents...), event.PreviousEvent) {
			if parent != "" {
				if err := visit(parent); err != nil {
					return err
				}
			}
		}
		state[id] = 2
		return nil
	}
	for id := range events {
		if err := visit(id); err != nil {
			return err
		}
	}
	return nil
}

func loadJSONTree(root string, visit func(string, []byte) error) error {
	info, err := os.Stat(root)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("%s is not a directory", root)
	}
	return filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("symlink is not allowed in authoritative state: %s", path)
		}
		if entry.IsDir() {
			return nil
		}
		if filepath.Ext(path) != ".json" {
			return fmt.Errorf("unexpected non-JSON authoritative file: %s", path)
		}
		data, err := readRegularBounded(path)
		if err != nil {
			return err
		}
		return visit(path, data)
	})
}

func readRegularBounded(path string) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return nil, fmt.Errorf("not a regular file: %s", path)
	}
	if info.Size() > MaxRecordBytes {
		return nil, fmt.Errorf("record exceeds %d-byte limit: %s", MaxRecordBytes, path)
	}
	return os.ReadFile(path)
}

func IsConflict(err error) bool { return errors.Is(err, ErrConflict) }
