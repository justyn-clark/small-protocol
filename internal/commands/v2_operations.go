package commands

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"
	"time"

	"github.com/justyn-clark/small-protocol/internal/sessionv2"
	"github.com/justyn-clark/small-protocol/internal/small"
)

func runV2Plan(baseDir, explicitSession string, reset bool, addTitle, doneRef, pendingRef, blockedRef, depends string) error {
	if reset {
		return fmt.Errorf("v2 authoritative history cannot be reset; create explicit task update events")
	}
	store, err := sessionv2.Load(baseDir)
	if err != nil {
		return err
	}
	state, err := sessionv2.Reduce(store)
	if err != nil {
		return err
	}
	operations := 0
	if strings.TrimSpace(addTitle) != "" {
		operations++
	}
	if doneRef != "" {
		operations++
	}
	if pendingRef != "" {
		operations++
	}
	if blockedRef != "" {
		operations++
	}
	if depends != "" {
		operations++
	}
	if operations > 1 {
		return fmt.Errorf("v2 plan accepts one mutation per invocation")
	}
	if operations == 0 {
		ids := sortedTaskIDs(state.Tasks)
		for _, id := range ids {
			task := state.Tasks[id]
			fmt.Printf("%s\t%s\t%s\t%s\n", task.Alias, task.ID, task.Status, task.Title)
		}
		return nil
	}
	sessionID, err := sessionv2.ResolveSession(baseDir, explicitSession)
	if err != nil {
		return err
	}
	frontier := state.Frontier
	if strings.TrimSpace(addTitle) != "" {
		taskID, err := sessionv2.NewID("task")
		if err != nil {
			return err
		}
		alias := nextV2Alias(state.Tasks)
		event, err := sessionv2.AppendEvent(baseDir, sessionID, "task_created", map[string]any{"alias": alias, "title": strings.TrimSpace(addTitle)}, sessionv2.AppendOptions{TaskID: taskID, TaskRevision: 1, ExpectedFrontier: frontier})
		if err != nil {
			return err
		}
		fmt.Printf("Added task %s (%s): %s [%s]\n", alias, taskID, strings.TrimSpace(addTitle), event.EventID)
		return nil
	}
	if depends != "" {
		parts := strings.SplitN(depends, ":", 2)
		if len(parts) != 2 {
			return fmt.Errorf("--depends format must be <task-id-or-alias>:<dependency-id-or-alias>")
		}
		task, err := resolveV2Task(state, parts[0])
		if err != nil {
			return err
		}
		dependency, err := resolveV2Task(state, parts[1])
		if err != nil {
			return err
		}
		deps := append([]string{}, task.Dependencies...)
		found := false
		for _, id := range deps {
			if id == dependency.ID {
				found = true
			}
		}
		if !found {
			deps = append(deps, dependency.ID)
		}
		sort.Strings(deps)
		event, err := sessionv2.AppendEvent(baseDir, sessionID, "task_updated", map[string]any{"alias": task.Alias, "title": task.Title, "acceptance": task.Acceptance, "dependencies": deps}, sessionv2.AppendOptions{TaskID: task.ID, TaskRevision: task.Revision + 1, PolicyRevision: task.PolicyRevision, SourceDigest: task.SourceDigest, ExpectedFrontier: frontier})
		if err != nil {
			return err
		}
		fmt.Printf("Updated %s dependencies [%s]\n", task.Alias, event.EventID)
		return nil
	}
	ref, status := pendingRef, "pending"
	if doneRef != "" {
		ref, status = doneRef, "completed"
	}
	if blockedRef != "" {
		ref, status = blockedRef, "blocked"
	}
	task, err := resolveV2Task(state, ref)
	if err != nil {
		return err
	}
	if status == "completed" && len(task.Acceptance) > 0 {
		return fmt.Errorf("task %s has acceptance criteria; use small checkpoint --evidence", task.Alias)
	}
	event, err := sessionv2.AppendEvent(baseDir, sessionID, "task_transitioned", map[string]any{"status": status, "reason": "small plan lifecycle flag"}, sessionv2.AppendOptions{TaskID: task.ID, TaskRevision: task.Revision, PolicyRevision: task.PolicyRevision, SourceDigest: task.SourceDigest, ExpectedFrontier: frontier})
	if err != nil {
		return err
	}
	fmt.Printf("Marked task %s as %s [%s]\n", task.Alias, status, event.EventID)
	return nil
}

func runV2Progress(baseDir, explicitSession, taskRef, status, evidence, notes, timestampAt, timestampAfter string, jsonOutput bool) error {
	status = strings.ToLower(strings.TrimSpace(status))
	if !isValidProgressStatus(status) {
		return fmt.Errorf("invalid status %q", status)
	}
	store, err := sessionv2.Load(baseDir)
	if err != nil {
		return err
	}
	state, err := sessionv2.Reduce(store)
	if err != nil {
		return err
	}
	task, err := resolveV2Task(state, taskRef)
	if err != nil {
		return err
	}
	sessionID, err := sessionv2.ResolveSession(baseDir, explicitSession)
	if err != nil {
		return err
	}
	occurred, err := v2Occurrence(timestampAt, timestampAfter)
	if err != nil {
		return err
	}
	payload := map[string]any{"status": status}
	if strings.TrimSpace(evidence) != "" {
		payload["evidence"] = strings.TrimSpace(evidence)
	}
	if strings.TrimSpace(notes) != "" {
		payload["notes"] = strings.TrimSpace(notes)
	}
	event, err := sessionv2.AppendEvent(baseDir, sessionID, "task_transitioned", payload, sessionv2.AppendOptions{TaskID: task.ID, TaskRevision: task.Revision, PolicyRevision: task.PolicyRevision, SourceDigest: task.SourceDigest, ExpectedFrontier: state.Frontier, OccurredAt: occurred})
	if err != nil {
		return err
	}
	if jsonOutput {
		return writeJSONValue(map[string]any{"profile": sessionv2.ProfileVersion, "event": event, "task_id": task.ID, "alias": task.Alias})
	}
	fmt.Printf("progress added: %s %s %s\n", task.Alias, status, event.EventID)
	return nil
}

func runV2Checkpoint(baseDir, explicitSession, taskRef, status, evidence, notes string, jsonOutput bool) error {
	status = strings.ToLower(strings.TrimSpace(status))
	if status != "completed" && status != "blocked" {
		return fmt.Errorf("invalid status %q (must be completed or blocked)", status)
	}
	if strings.TrimSpace(evidence) == "" {
		return fmt.Errorf("v2 checkpoint requires explicit --evidence")
	}
	store, err := sessionv2.Load(baseDir)
	if err != nil {
		return err
	}
	state, err := sessionv2.Reduce(store)
	if err != nil {
		return err
	}
	task, err := resolveV2Task(state, taskRef)
	if err != nil {
		return err
	}
	sessionID, err := sessionv2.ResolveSession(baseDir, explicitSession)
	if err != nil {
		return err
	}
	receipt, err := prepareNarrativeReceipt(store.Profile, task, strings.TrimSpace(evidence), strings.TrimSpace(notes), status)
	if err != nil {
		return err
	}
	payload := map[string]any{"status": status, "evidence": strings.TrimSpace(evidence), "evidence_digests": []string{receipt.Digest}}
	if strings.TrimSpace(notes) != "" {
		payload["notes"] = strings.TrimSpace(notes)
	}
	event, receipts, err := sessionv2.AppendEventWithReceipts(baseDir, sessionID, "task_transitioned", payload, sessionv2.AppendOptions{TaskID: task.ID, TaskRevision: task.Revision, PolicyRevision: task.PolicyRevision, SourceDigest: task.SourceDigest, ExpectedFrontier: state.Frontier}, []sessionv2.Receipt{receipt})
	if err != nil {
		return err
	}
	if jsonOutput {
		return writeJSONValue(map[string]any{"profile": sessionv2.ProfileVersion, "task_id": task.ID, "alias": task.Alias, "status": status, "event": event, "receipt": receipts[0], "validated": true})
	}
	fmt.Printf("checkpoint: %s -> %s [%s receipt:%s]\n", task.Alias, status, event.EventID, receipts[0].Digest)
	return nil
}

func runV2Apply(baseDir, explicitSession, taskRef, command string, dryRun, autoCheckpoint bool, acceptanceEvidence string, handoff, jsonOutput bool) error {
	store, err := sessionv2.Load(baseDir)
	if err != nil {
		return err
	}
	state, err := sessionv2.Reduce(store)
	if err != nil {
		return err
	}
	var task sessionv2.TaskState
	if strings.TrimSpace(taskRef) != "" {
		task, err = resolveV2Task(state, taskRef)
		if err != nil {
			return err
		}
	}
	if autoCheckpoint && task.ID == "" {
		return fmt.Errorf("--auto-checkpoint requires --task")
	}
	if autoCheckpoint && len(task.Acceptance) > 0 && strings.TrimSpace(acceptanceEvidence) == "" {
		return fmt.Errorf("--auto-checkpoint requires --acceptance-evidence for task %s", task.Alias)
	}
	if dryRun {
		result := applyOutput{Command: small.SummarizeCommand(command, small.DefaultCommandSummaryCap), TaskID: task.ID, ChildOutcome: "not_executed", DryRun: true}
		if jsonOutput {
			return printApplyJSON(result)
		}
		fmt.Printf("Dry-run: would execute %s\n", command)
		return nil
	}
	sessionID, err := sessionv2.ResolveSession(baseDir, explicitSession)
	if err != nil {
		return err
	}
	child := exec.Command("sh", "-lc", command)
	child.Dir = baseDir
	output, childErr := child.CombinedOutput()
	exitCode := 0
	outcome := "succeeded"
	if childErr != nil {
		outcome = "failed"
		exitCode = 1
		if ee, ok := childErr.(*exec.ExitError); ok {
			exitCode = ee.ExitCode()
		}
	}
	if !jsonOutput && len(output) > 0 {
		_, _ = os.Stdout.Write(output)
	}
	commandSum := sha256.Sum256([]byte(command))
	source := hex.EncodeToString(commandSum[:])
	bounded := string(output)
	truncated := false
	if len(bounded) > 4000 {
		bounded = bounded[:4000]
		truncated = true
	}
	receiptID, err := sessionv2.NewID("receipt")
	if err != nil {
		return err
	}
	receipt := sessionv2.Receipt{SmallVersion: sessionv2.ProfileVersion, ProjectID: store.Profile.ProjectID, ReceiptID: receiptID, Strength: "cli_captured", Availability: "verified", Producer: map[string]any{"tool": "small apply", "command_sha256": source}, TaskID: task.ID, TaskRevision: task.Revision, PolicyRevision: task.PolicyRevision, SourceDigest: source, Outcome: outcome, Metadata: map[string]any{"exit_code": exitCode, "bounded_output": bounded, "output_truncated": truncated}}
	receipt.Digest, err = sessionv2.DigestRecord(receipt)
	if err != nil {
		return err
	}
	payload := map[string]any{"command_summary": small.SummarizeCommand(command, small.DefaultCommandSummaryCap), "command_sha256": source, "outcome": outcome, "exit_code": exitCode, "evidence_digests": []string{receipt.Digest}, "task_acceptance_unchanged": true}
	event, receipts, persistErr := sessionv2.AppendEventWithReceipts(baseDir, sessionID, "command_recorded", payload, sessionv2.AppendOptions{TaskID: task.ID, TaskRevision: task.Revision, PolicyRevision: task.PolicyRevision, SourceDigest: source, ExpectedFrontier: state.Frontier}, []sessionv2.Receipt{receipt})
	if persistErr != nil {
		if jsonOutput {
			_ = writeJSONValue(map[string]any{"profile": sessionv2.ProfileVersion, "child_outcome": outcome, "child_exit_code": exitCode, "child_executed": true, "evidence_recorded": false, "error_code": "executed_but_unrecorded"})
			return reportedJSONError{cause: fmt.Errorf("command executed with exit code %d but durable evidence recording failed; command was not retried: %w", exitCode, persistErr)}
		}
		return fmt.Errorf("command executed with exit code %d but durable evidence recording failed; command was not retried: %w", exitCode, persistErr)
	}
	checkpointed := false
	if autoCheckpoint {
		checkpointEvidence := strings.TrimSpace(acceptanceEvidence)
		if checkpointEvidence == "" {
			checkpointEvidence = fmt.Sprintf("Explicit acceptance of captured command result exit_code=%d", exitCode)
		}
		checkpointStatus := "completed"
		if childErr != nil {
			checkpointStatus = "blocked"
		}
		if err := runV2Checkpoint(baseDir, sessionID, task.ID, checkpointStatus, checkpointEvidence, "small apply --auto-checkpoint", false); err != nil {
			return fmt.Errorf("command evidence recorded but checkpoint failed: %w", err)
		}
		checkpointed = true
	}
	if handoff && childErr == nil {
		current, err := sessionv2.Load(baseDir)
		if err != nil {
			return err
		}
		clean, err := sessionv2.Strict(current)
		if err != nil {
			return fmt.Errorf("command evidence recorded but handoff refused: %w", err)
		}
		if _, err := sessionv2.AppendEvent(baseDir, sessionID, "handoff_recorded", map[string]any{"summary": fmt.Sprintf("Command completed successfully: %s", small.SummarizeCommand(command, small.DefaultCommandSummaryCap)), "authoritative": true}, sessionv2.AppendOptions{ExpectedFrontier: clean.Frontier}); err != nil {
			return fmt.Errorf("command evidence recorded but handoff failed: %w", err)
		}
	}
	if jsonOutput {
		_ = writeJSONValue(map[string]any{"profile": sessionv2.ProfileVersion, "command": small.SummarizeCommand(command, small.DefaultCommandSummaryCap), "task_id": task.ID, "child_outcome": outcome, "child_exit_code": exitCode, "child_executed": true, "evidence_recorded": true, "checkpointed": checkpointed, "event": event.EventID, "receipt": receipts[0].Digest})
	}
	if childErr != nil {
		err := fmt.Errorf("child command failed with exit code %d after evidence was recorded", exitCode)
		if jsonOutput {
			return reportedJSONError{cause: err}
		}
		return err
	}
	if !jsonOutput {
		fmt.Printf("Command completed successfully (exit code: %d); evidence %s\n", exitCode, receipts[0].Digest)
	}
	return nil
}

func resolveV2Task(state sessionv2.State, ref string) (sessionv2.TaskState, error) {
	ref = strings.TrimSpace(ref)
	if task, ok := state.Tasks[ref]; ok {
		return task, nil
	}
	matches := []sessionv2.TaskState{}
	for _, task := range state.Tasks {
		if task.Alias == ref {
			matches = append(matches, task)
		}
	}
	if len(matches) == 1 {
		return matches[0], nil
	}
	if len(matches) > 1 {
		return sessionv2.TaskState{}, fmt.Errorf("task alias %s is ambiguous; use opaque task id", ref)
	}
	return sessionv2.TaskState{}, fmt.Errorf("task %s not found", ref)
}
func sortedTaskIDs(tasks map[string]sessionv2.TaskState) []string {
	ids := make([]string, 0, len(tasks))
	for id := range tasks {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}
func nextV2Alias(tasks map[string]sessionv2.TaskState) string {
	used := map[string]bool{}
	for _, task := range tasks {
		used[task.Alias] = true
	}
	for i := 1; ; i++ {
		alias := fmt.Sprintf("task-%d", i)
		if !used[alias] {
			return alias
		}
	}
}
func v2Occurrence(at, after string) (time.Time, error) {
	if strings.TrimSpace(at) != "" {
		parsed, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(at))
		if err != nil {
			return time.Time{}, fmt.Errorf("invalid --at: %w", err)
		}
		return parsed, nil
	}
	now := time.Now().UTC()
	if strings.TrimSpace(after) != "" {
		parsed, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(after))
		if err != nil {
			return time.Time{}, fmt.Errorf("invalid --after: %w", err)
		}
		if !now.After(parsed) {
			now = parsed.Add(time.Nanosecond)
		}
	}
	return now, nil
}
func prepareNarrativeReceipt(profile sessionv2.Profile, task sessionv2.TaskState, evidence, notes, outcome string) (sessionv2.Receipt, error) {
	id, err := sessionv2.NewID("receipt")
	if err != nil {
		return sessionv2.Receipt{}, err
	}
	receipt := sessionv2.Receipt{SmallVersion: sessionv2.ProfileVersion, ProjectID: profile.ProjectID, ReceiptID: id, Strength: "narrative_assertion", Availability: "verified", Producer: map[string]any{"tool": "small checkpoint"}, TaskID: task.ID, TaskRevision: task.Revision, PolicyRevision: task.PolicyRevision, SourceDigest: task.SourceDigest, Outcome: outcome, Metadata: map[string]any{"evidence": evidence, "notes": notes}}
	receipt.Digest, err = sessionv2.DigestRecord(receipt)
	return receipt, err
}
