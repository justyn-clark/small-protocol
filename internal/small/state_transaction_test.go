package small

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestStateTransactionRecoversEveryPublicationBoundary(t *testing.T) {
	for _, boundary := range []string{"prepared", "publish-1", "publish-2", "committed"} {
		t.Run(boundary, func(t *testing.T) {
			base := t.TempDir()
			if err := os.MkdirAll(filepath.Join(base, SmallDir), 0o755); err != nil {
				t.Fatal(err)
			}
			first := filepath.Join(base, SmallDir, "plan.small.yml")
			second := filepath.Join(base, SmallDir, "progress.small.yml")
			if err := os.WriteFile(first, []byte("old-plan\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(second, []byte("old-progress\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			t.Setenv("SMALL_TEST_FAIL_TRANSACTION_AFTER", boundary)
			changed, err := WriteStateFiles(base, []StateFile{
				{Path: ".small/plan.small.yml", Data: []byte("new-plan\n")},
				{Path: ".small/progress.small.yml", Data: []byte("new-progress\n")},
			})
			if err == nil {
				t.Fatalf("expected injected failure at %s", boundary)
			}
			if changed {
				t.Fatal("incomplete transaction must not report clean success")
			}
			pending, pendingErr := PendingStateTransactions(base)
			if pendingErr != nil || len(pending) != 1 {
				t.Fatalf("expected one recoverable transaction, pending=%v err=%v", pending, pendingErr)
			}
			t.Setenv("SMALL_TEST_FAIL_TRANSACTION_AFTER", "")
			if err := RecoverStateTransactions(base); err != nil {
				t.Fatalf("recover: %v", err)
			}
			for path, want := range map[string]string{first: "new-plan\n", second: "new-progress\n"} {
				data, readErr := os.ReadFile(path)
				if readErr != nil || string(data) != want {
					t.Fatalf("%s after recovery = %q, %v; want %q", path, data, readErr, want)
				}
			}
		})
	}
}

func TestStateTransactionNoOpAndLocalLock(t *testing.T) {
	base := t.TempDir()
	if err := os.MkdirAll(filepath.Join(base, SmallDir), 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(base, SmallDir, "handoff.small.yml")
	if err := os.WriteFile(path, []byte("same\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	changed, err := WriteStateFiles(base, []StateFile{{Path: ".small/handoff.small.yml", Data: []byte("same\n")}})
	if err != nil || changed {
		t.Fatalf("equivalent write: changed=%v err=%v", changed, err)
	}

	err = WithStateLock(base, func() error {
		inner := WithStateLock(base, func() error { return nil })
		if !errors.Is(inner, ErrStateBusy) {
			t.Fatalf("nested writer error = %v, want ErrStateBusy", inner)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
