package commands

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/justyn-clark/small-protocol/internal/sessionv2"
	"github.com/spf13/cobra"
)

type resolutionFile struct {
	Profile          string   `json:"profile"`
	ConflictID       string   `json:"conflict_id"`
	Heads            []string `json:"heads"`
	SelectedEvent    string   `json:"selected_event"`
	Reason           string   `json:"reason"`
	Resolver         string   `json:"resolver,omitempty"`
	SessionID        string   `json:"session_id,omitempty"`
	ExpectedFrontier string   `json:"expected_frontier"`
}

func reconcileCmd() *cobra.Command {
	var dir, applyPath, expected, explicitSession string
	var preview, jsonOutput bool
	cmd := &cobra.Command{
		Use: "reconcile", Short: "Preview or explicitly resolve SMALL v2 semantic conflicts",
		RunE: func(cmd *cobra.Command, args []string) error {
			if dir == "" {
				dir = baseDir
			}
			dir = resolveArtifactsDir(dir)
			if preview == (applyPath != "") {
				return fmt.Errorf("choose exactly one of --preview or --apply")
			}
			store, err := sessionv2.Load(dir)
			if err != nil {
				return err
			}
			state, err := sessionv2.Reduce(store)
			if err != nil {
				return err
			}
			if preview {
				next := []string{}
				for _, conflict := range state.Conflicts {
					if conflict.Kind == "policy_material" {
						next = append(next, fmt.Sprintf("restore the reviewed policy bytes selected by %s", strings.Join(conflict.Heads, ",")))
					} else {
						next = append(next, fmt.Sprintf("create a resolution for %s referencing all heads", conflict.ID))
					}
				}
				if len(next) == 0 {
					next = append(next, "no reconciliation required")
				}
				payload := map[string]any{"profile": sessionv2.ProfileVersion, "mode": state.Profile.Mode, "frontier": state.Frontier, "conflicts": state.Conflicts, "conflict_count": len(state.Conflicts), "next_steps": next}
				if jsonOutput {
					return writeJSONValue(payload)
				}
				fmt.Printf("frontier %s; conflicts %d\n", state.Frontier, len(state.Conflicts))
				for _, conflict := range state.Conflicts {
					fmt.Printf("%s\t%s\t%s\n", conflict.ID, conflict.Kind, conflict.Message)
				}
				return nil
			}
			if strings.TrimSpace(expected) == "" {
				return fmt.Errorf("--expect-state is required with --apply")
			}
			data, err := os.ReadFile(applyPath)
			if err != nil {
				return err
			}
			var resolution resolutionFile
			if err := sessionv2.DecodeStrict(data, &resolution); err != nil {
				return fmt.Errorf("resolution file: %w", err)
			}
			if resolution.Profile != sessionv2.ProfileVersion {
				return fmt.Errorf("resolution profile must be %s", sessionv2.ProfileVersion)
			}
			if expected != state.Frontier || resolution.ExpectedFrontier != expected {
				return fmt.Errorf("%w: expected %s, actual %s", sessionv2.ErrStaleFrontier, expected, state.Frontier)
			}
			var current *sessionv2.Conflict
			for i := range state.Conflicts {
				if state.Conflicts[i].ID == resolution.ConflictID {
					current = &state.Conflicts[i]
					break
				}
			}
			if current == nil {
				return fmt.Errorf("conflict %s is absent or already resolved", resolution.ConflictID)
			}
			if !equalStringSets(current.Heads, resolution.Heads) {
				return fmt.Errorf("resolution heads do not match current conflict")
			}
			selected := false
			for _, head := range current.Heads {
				if head == resolution.SelectedEvent {
					selected = true
				}
			}
			if !selected {
				return fmt.Errorf("selected_event must be one of the competing heads")
			}
			if strings.TrimSpace(resolution.Reason) == "" {
				return fmt.Errorf("resolution reason is required")
			}
			sessionID := explicitSession
			if sessionID == "" {
				sessionID = resolution.SessionID
			}
			sessionID, err = sessionv2.ResolveSession(dir, sessionID)
			if err != nil {
				return err
			}
			event, err := sessionv2.AppendEvent(dir, sessionID, "conflict_resolved", map[string]any{"conflict_id": current.ID, "heads": current.Heads, "selected_event": resolution.SelectedEvent, "expected_frontier": expected, "reason": strings.TrimSpace(resolution.Reason), "resolver": strings.TrimSpace(resolution.Resolver)}, sessionv2.AppendOptions{ExpectedFrontier: expected, Parents: current.Heads})
			if err != nil {
				return err
			}
			updated, err := sessionv2.Load(dir)
			if err != nil {
				return err
			}
			updatedState, err := sessionv2.Reduce(updated)
			if err != nil {
				return err
			}
			for _, conflict := range updatedState.Conflicts {
				if conflict.ID == current.ID {
					return fmt.Errorf("resolution event %s did not resolve conflict %s", event.EventID, current.ID)
				}
			}
			if jsonOutput {
				return writeJSONValue(map[string]any{"profile": sessionv2.ProfileVersion, "resolved": current.ID, "resolution_event": event, "frontier": updatedState.Frontier, "remaining_conflicts": updatedState.Conflicts})
			}
			fmt.Printf("Resolved %s with %s; frontier %s\n", current.ID, event.EventID, updatedState.Frontier)
			return nil
		},
	}
	cmd.Flags().StringVar(&dir, "dir", "", "Project directory")
	cmd.Flags().BoolVar(&preview, "preview", false, "Read-only conflict preview")
	cmd.Flags().StringVar(&applyPath, "apply", "", "Apply a reviewed JSON resolution file")
	cmd.Flags().StringVar(&expected, "expect-state", "", "Required current frontier digest")
	cmd.Flags().StringVar(&explicitSession, "session", "", "Resolver session id")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output JSON")
	return cmd
}

func equalStringSets(left, right []string) bool {
	left = append([]string{}, left...)
	right = append([]string{}, right...)
	sort.Strings(left)
	sort.Strings(right)
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}
