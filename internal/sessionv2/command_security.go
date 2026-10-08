package sessionv2

import (
	"path/filepath"
	"sort"

	"github.com/justyn-clark/small-protocol/internal/small"
)

// CommandSecurityViolations never changes events, receipts, or profile state.
func CommandSecurityViolations(store *Store) []small.InvariantViolation {
	ids := make([]string, 0, len(store.Events))
	for id := range store.Events {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	var violations []small.InvariantViolation
	for _, id := range ids {
		event := store.Events[id]
		if event.Kind != "command_recorded" {
			continue
		}
		path := filepath.Join(store.BaseDir, ".small", "events", event.SessionID, event.EventID+".json")
		if _, marked := event.Payload["command_summary_version"]; marked && event.SourceDigest != event.Payload["command_sha256"] {
			violations = append(violations, small.InvariantViolation{File: path, Message: "command_sha256 does not match event source_digest"})
		}
		violations = append(violations, small.CheckCommandRecord(store.BaseDir, path, event.Payload)...)
	}
	return violations
}
