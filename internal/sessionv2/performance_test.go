package sessionv2

import (
	"fmt"
	"os"
	"runtime"
	"strconv"
	"testing"
	"time"
)

// TestReducerScale is opt-in because 100k records are a release measurement,
// not a useful cost on every unit-test invocation. Run with
// SMALL_PERF_EVENTS=10000 and SMALL_PERF_EVENTS=100000.
func TestReducerScale(t *testing.T) {
	raw := os.Getenv("SMALL_PERF_EVENTS")
	if raw == "" {
		t.Skip("set SMALL_PERF_EVENTS for scale measurement")
	}
	count, err := strconv.Atoi(raw)
	if err != nil || count < 1 {
		t.Fatalf("invalid SMALL_PERF_EVENTS=%q", raw)
	}
	profile := Profile{SmallVersion: ProfileVersion, ProjectID: "perf_project", LineageID: "perf_lineage", Mode: "collaborative", PolicyRevision: "policy_1"}
	session := Session{SmallVersion: ProfileVersion, ProjectID: profile.ProjectID, LineageID: profile.LineageID, SessionID: "perf_session", ObservedFrontier: Frontier(nil), ToolVersion: "performance-test", CreatedAt: "2026-01-01T00:00:00Z"}
	session.Digest, _ = DigestRecord(session)
	store := &Store{Profile: profile, Sessions: map[string]Session{session.SessionID: session}, Events: make(map[string]Event, count), Receipts: map[string]Receipt{}}
	previous := ""
	for i := 0; i < count; i++ {
		id := fmt.Sprintf("event_%09d", i)
		event := Event{SmallVersion: ProfileVersion, ProjectID: profile.ProjectID, LineageID: profile.LineageID, SessionID: session.SessionID, EventID: id, Kind: "command_recorded", Sequence: int64(i + 1), PreviousEvent: previous, OccurredAt: "2026-01-01T00:00:00Z", Payload: map[string]any{"outcome": "succeeded"}}
		event.Digest, _ = DigestRecord(event)
		store.Events[id] = event
		previous = id
	}
	runtime.GC()
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	started := time.Now()
	state, err := Reduce(store)
	elapsed := time.Since(started)
	runtime.ReadMemStats(&after)
	if err != nil {
		t.Fatal(err)
	}
	if state.EventCount != count || len(state.Conflicts) != 0 {
		t.Fatalf("state events=%d conflicts=%d", state.EventCount, len(state.Conflicts))
	}
	t.Logf("SMALL_PERF events=%d elapsed=%s alloc_delta_bytes=%d total_alloc_delta_bytes=%d frontier=%s", count, elapsed, int64(after.Alloc)-int64(before.Alloc), after.TotalAlloc-before.TotalAlloc, state.Frontier)
}
