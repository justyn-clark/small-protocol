package migratev2

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/justyn-clark/small-protocol/internal/sessionv2"
)

func TestMigrationDeterministicLossPreservingAndIdempotent(t *testing.T) {
	left := t.TempDir()
	right := t.TempDir()
	leftOriginals := writeLegacyFixture(t, left)
	writeLegacyFixture(t, right)
	leftPlan, err := Preview(left, "shared-test-baseline", "")
	if err != nil {
		t.Fatal(err)
	}
	rightPlan, err := Preview(right, "shared-test-baseline", "solo")
	if err != nil {
		t.Fatal(err)
	}
	if leftPlan.ExpectedInputDigest != rightPlan.ExpectedInputDigest || leftPlan.ProjectID != rightPlan.ProjectID || leftPlan.ImportID != rightPlan.ImportID {
		t.Fatalf("independent previews differ:\n%#v\n%#v", leftPlan, rightPlan)
	}
	if leftPlan.Mode != "solo" {
		t.Fatalf("default mode = %q", leftPlan.Mode)
	}
	result, err := Apply(left, leftPlan)
	if err != nil {
		t.Fatal(err)
	}
	if result.Idempotent {
		t.Fatal("first migration reported idempotent")
	}
	store, err := sessionv2.Load(left)
	if err != nil {
		t.Fatal(err)
	}
	state, err := sessionv2.Strict(store)
	if err != nil {
		t.Fatal(err)
	}
	if state.Profile.Mode != "solo" || state.EventCount == 0 || len(state.Tasks) != 1 {
		t.Fatalf("state = %#v", state)
	}
	for path, want := range leftOriginals {
		got, err := os.ReadFile(filepath.Join(left, ".small", "imports", "v1", leftPlan.ImportID, "originals", path))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, want) {
			t.Fatalf("archived %s changed", path)
		}
	}
	again, err := Apply(left, leftPlan)
	if err != nil {
		t.Fatal(err)
	}
	if !again.Idempotent || again.Frontier != result.Frontier {
		t.Fatalf("second result = %#v", again)
	}
}

func TestMigrationRejectsStaleInput(t *testing.T) {
	base := t.TempDir()
	writeLegacyFixture(t, base)
	plan, err := Preview(base, "stale-test", "solo")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(base, ".small", "extra"), []byte("changed\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Apply(base, plan); !errors.Is(err, sessionv2.ErrStaleFrontier) {
		t.Fatalf("error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(base, ".small", "plan.small.yml")); err != nil {
		t.Fatalf("legacy source not retained: %v", err)
	}
}

func TestMigrationRecoversEveryPublishedBoundary(t *testing.T) {
	for _, stage := range []string{"prepared", "backup", "publish"} {
		t.Run(stage, func(t *testing.T) {
			base := t.TempDir()
			writeLegacyFixture(t, base)
			plan, err := Preview(base, "fault-"+stage, "solo")
			if err != nil {
				t.Fatal(err)
			}
			t.Setenv("SMALL_TEST_FAIL_MIGRATION_AFTER", stage)
			if _, err := Apply(base, plan); err == nil {
				t.Fatal("fault did not interrupt migration")
			}
			t.Setenv("SMALL_TEST_FAIL_MIGRATION_AFTER", "")
			result, err := Apply(base, plan)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := sessionv2.Load(base); err != nil {
				t.Fatal(err)
			}
			if stage != "prepared" && !result.Idempotent {
				t.Fatalf("recovered result = %#v", result)
			}
			archived, err := os.ReadFile(filepath.Join(base, ".small", "imports", "v1", plan.ImportID, "originals", "plan.small.yml"))
			if err != nil || len(archived) == 0 {
				t.Fatalf("archived plan: %v", err)
			}
		})
	}
}

func writeLegacyFixture(t *testing.T, base string) map[string][]byte {
	t.Helper()
	files := map[string][]byte{
		"intent.small.yml":      []byte("small_version: \"1.0.0\"\nowner: human\nintent: preserve me\n"),
		"constraints.small.yml": []byte("small_version: \"1.0.0\"\nowner: human\nconstraints: []\n"),
		"plan.small.yml":        []byte("small_version: \"1.0.0\"\nowner: agent\ntasks:\n  - id: task-1\n    title: Keep identity\n    acceptance: [verified]\n    status: completed\n"),
		"progress.small.yml":    []byte("small_version: \"1.0.0\"\nowner: agent\nentries:\n  - timestamp: 2026-01-01T00:00:00.000000000Z\n    task_id: task-1\n    status: completed\n    evidence: exact bytes\n"),
		"handoff.small.yml":     []byte("small_version: \"1.0.0\"\nowner: agent\nsummary: |\n  Authored narrative.\n  Resume carefully.\nresume:\n  next_steps: []\nlinks: []\nreplayId:\n  value: baseline-replay\n  source: test\n"),
	}
	for path, data := range files {
		full := filepath.Join(base, ".small", path)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return files
}
