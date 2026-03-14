package commands

import (
	"fmt"
	"strings"

	"github.com/justyn-clark/small-protocol/internal/small/fixers"
)

func recordRuntimeLayoutReconcileEntry(artifactsDir string, result fixers.RuntimeLayoutFixResult) error {
	if len(result.Migrations) == 0 && len(result.Deduped) == 0 {
		return nil
	}

	parts := make([]string, 0, len(result.Migrations))
	for _, migration := range result.Migrations {
		parts = append(parts, fmt.Sprintf("%s->%s", migration.SourceRoot, migration.TargetRoot))
	}
	if len(result.Deduped) > 0 {
		parts = append(parts, fmt.Sprintf("deduped=%d", len(result.Deduped)))
	}

	entry := map[string]any{
		"task_id":  "meta/reconcile-runtime-layout",
		"status":   "completed",
		"evidence": fmt.Sprintf("Migrated legacy runtime layout to canonical stores (%d migration(s), %d deduped file(s))", len(result.Migrations), len(result.Deduped)),
		"notes":    strings.Join(parts, ", "),
	}

	if err := appendProgressEntry(artifactsDir, entry); err != nil {
		return fmt.Errorf("failed to append runtime layout reconcile progress entry: %w", err)
	}
	return nil
}
