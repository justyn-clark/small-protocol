package commands

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"testing"

	"github.com/justyn-clark/small-protocol/internal/small"
	"github.com/justyn-clark/small-protocol/internal/workspace"
)

func TestPlanWriteInitializesRunReplayIDAndHandoffMatches(t *testing.T) {
	tmpDir := t.TempDir()
	writeArtifacts(t, tmpDir, defaultArtifacts())
	mustSaveWorkspace(t, tmpDir, workspace.KindRepoRoot)

	plan := planCmd()
	plan.SetArgs([]string{"--dir", tmpDir, "--add", "Replay-bound task"})
	if err := plan.Execute(); err != nil {
		t.Fatalf("plan --add failed: %v", err)
	}

	replayID, err := workspace.RunReplayID(tmpDir)
	if err != nil {
		t.Fatalf("workspace replay id load failed: %v", err)
	}
	if replayID == "" {
		t.Fatal("expected workspace replay_id to be initialized after plan write")
	}

	progress, err := loadProgressData(tmpDir + "/.small/progress.small.yml")
	if err != nil {
		t.Fatalf("load progress failed: %v", err)
	}
	if len(progress.Entries) == 0 {
		t.Fatal("expected progress entries after plan --add")
	}
	last := progress.Entries[len(progress.Entries)-1]
	if last["task_id"] != "task-2" {
		t.Fatalf("expected last task_id task-2, got %v", last["task_id"])
	}
	if got := stringVal(last["replayId"]); got != replayID {
		t.Fatalf("task progress replayId = %q, want %q", got, replayID)
	}

	checkpoint := checkpointCmd()
	checkpoint.SetArgs([]string{"--dir", tmpDir, "--task", "task-2", "--status", "completed", "--evidence", "done"})
	if err := checkpoint.Execute(); err != nil {
		t.Fatalf("checkpoint failed: %v", err)
	}

	handoff := handoffCmd()
	handoff.SetArgs([]string{"--dir", tmpDir})
	if err := handoff.Execute(); err != nil {
		t.Fatalf("handoff failed: %v", err)
	}

	handoffArtifact, err := small.LoadArtifact(tmpDir, "handoff.small.yml")
	if err != nil {
		t.Fatalf("failed to load handoff artifact: %v", err)
	}
	replayRaw, _ := handoffArtifact.Data["replayId"].(map[string]any)
	if got := stringVal(replayRaw["value"]); got != replayID {
		t.Fatalf("handoff replayId = %q, want %q", got, replayID)
	}

	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe error: %v", err)
	}
	os.Stdout = w
	defer func() { os.Stdout = oldStdout }()

	var buf bytes.Buffer
	done := make(chan struct{})
	go func() {
		_, _ = io.Copy(&buf, r)
		_ = r.Close()
		close(done)
	}()

	status := statusCmd()
	status.SetArgs([]string{"--dir", tmpDir, "--json"})
	if err := status.Execute(); err != nil {
		t.Fatalf("status failed: %v", err)
	}
	_ = w.Close()
	<-done

	var out StatusOutput
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatalf("parse status json failed: %v", err)
	}
	if out.ReplayID != replayID {
		t.Fatalf("status replay_id = %q, want %q", out.ReplayID, replayID)
	}
}
