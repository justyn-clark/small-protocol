package commands

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"

	"github.com/justyn-clark/small-protocol/internal/sessionv2"
	"github.com/justyn-clark/small-protocol/internal/small"
	"github.com/spf13/cobra"
)

func proofCmd() *cobra.Command {
	parent := &cobra.Command{Use: "proof", Short: "Inspect captured command proof availability"}
	var dir string
	var jsonOutput bool
	status := &cobra.Command{
		Use:   "status",
		Short: "Report exact command proof coverage without changing files",
		Long:  "Read-only command proof inventory for v1/v2. Availability verifies local exact bytes and receipt binding, not credential safety. Incomplete legacy coverage does not change strict policy. No proof is reconstructed, copied, or disclosed.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			report, err := buildCommandProofInventory(dir)
			if err != nil {
				return err
			}
			if jsonOutput {
				return json.NewEncoder(cmd.OutOrStdout()).Encode(report)
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Command proof coverage: %s\nCaptures: %d; available: %d; missing: %d; invalid: %d; unreferenced: %d\nRequired but unavailable: %d\nDisplay bytes match digest: %d (not a secret review or permission to restore)\nAvailability is separate from strict policy and credential inspection.\n", report.Coverage, report.Total, report.Available, report.Missing, report.Invalid, report.Unreferenced, report.RequiredUnavailable, report.DisplayDigestMatches)
			return err
		},
	}
	status.Flags().StringVar(&dir, "dir", ".", "Workspace containing .small/ artifacts")
	status.Flags().BoolVar(&jsonOutput, "json", false, "Output availability counts and receipt locations as JSON")
	parent.AddCommand(status)
	return parent
}

func buildCommandProofInventory(dir string) (small.CommandProofInventory, error) {
	base := resolveArtifactsDir(dir)
	report := small.NewCommandProofInventory()
	if sessionv2.IsWorkspace(base) {
		store, err := sessionv2.Load(base)
		if err != nil {
			return report, err
		}
		ids := make([]string, 0, len(store.Events))
		for id := range store.Events {
			ids = append(ids, id)
		}
		sort.Strings(ids)
		for _, id := range ids {
			event := store.Events[id]
			if event.Kind == "command_recorded" {
				location := filepath.ToSlash(filepath.Join(".small", "events", event.SessionID, event.EventID+".json"))
				report.AddEvent(base, location, event.Payload, event.SourceDigest)
			}
		}
		return report, nil
	}
	artifact, err := small.LoadArtifact(base, "progress.small.yml")
	if err != nil {
		return report, err
	}
	entries, ok := artifact.Data["entries"].([]any)
	if !ok {
		return report, fmt.Errorf("progress entries must be an array")
	}
	for index, entry := range entries {
		record, ok := entry.(map[string]any)
		if !ok {
			return report, fmt.Errorf("progress entries[%d] must be an object", index)
		}
		report.Add(base, fmt.Sprintf(".small/progress.small.yml:entries[%d]", index), record)
	}
	return report, nil
}
