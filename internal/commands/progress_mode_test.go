package commands

import "testing"

func TestResolveProgressMode(t *testing.T) {
	t.Setenv(progressModeEnvVar, "")
	if got := resolveProgressMode(); got != progressModeSignal {
		t.Fatalf("resolveProgressMode() default = %q, want signal", got)
	}

	t.Setenv(progressModeEnvVar, "audit")
	if got := resolveProgressMode(); got != progressModeAudit {
		t.Fatalf("resolveProgressMode() audit = %q, want audit", got)
	}

	t.Setenv(progressModeEnvVar, "invalid")
	if got := resolveProgressMode(); got != progressModeSignal {
		t.Fatalf("resolveProgressMode() invalid = %q, want signal", got)
	}
}

func TestShouldEmitProgress(t *testing.T) {
	if shouldEmitProgress(progressEventApplyStart, "task-1", progressModeSignal) {
		t.Fatal("signal mode should not emit apply start")
	}
	if !shouldEmitProgress(progressEventApplyComplete, "task-1", progressModeSignal) {
		t.Fatal("signal mode should emit task apply completion")
	}
	if shouldEmitProgress(progressEventApplyComplete, "meta/reconcile-plan", progressModeSignal) {
		t.Fatal("signal mode should not emit non-task apply completion")
	}
	if !shouldEmitProgress(progressEventApplyStart, "meta/reconcile-plan", progressModeAudit) {
		t.Fatal("audit mode should emit apply start")
	}
}
