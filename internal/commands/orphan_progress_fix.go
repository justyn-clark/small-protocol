package commands

import (
	"fmt"
	"strings"

	"github.com/justyn-clark/small-protocol/internal/small/fixers"
)

func orphanProgressScopeLabel(replayID string) string {
	scopeLabel := strings.TrimSpace(replayID)
	if scopeLabel == "" {
		return "unknown"
	}
	return scopeLabel
}

func recordOrphanProgressReconcileEntry(artifactsDir string, result fixers.OrphanProgressFixResult) error {
	if len(result.Rewrites) == 0 {
		return nil
	}

	scopeLabel := orphanProgressScopeLabel(result.ReplayID)

	hashes := make([]string, 0, len(result.Rewrites))
	for _, rewrite := range result.Rewrites {
		hashes = append(hashes, fmt.Sprintf("%s:%s", rewrite.OriginalTaskID, rewrite.Hash))
	}

	entry := map[string]any{
		"task_id":  "meta/reconcile-plan",
		"status":   "completed",
		"evidence": fmt.Sprintf("Rewrote orphan progress task_ids for replayId scope %s (operational=%d historical=%d unknown=%d)", scopeLabel, result.Counts.Operational, result.Counts.Historical, result.Counts.Unknown),
		"notes":    fmt.Sprintf("original hashes: %s", strings.Join(hashes, ", ")),
	}

	if err := appendProgressEntry(artifactsDir, entry); err != nil {
		return fmt.Errorf("failed to append reconcile progress entry: %w", err)
	}

	return nil
}
