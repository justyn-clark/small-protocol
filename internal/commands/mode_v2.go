package commands

import (
	"fmt"
	"strings"

	"github.com/justyn-clark/small-protocol/internal/sessionv2"
	"github.com/spf13/cobra"
)

func modeCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "mode", Short: "Inspect or change the SMALL v2 execution mode"}
	cmd.AddCommand(modeShowCmd(), modeSetCmd())
	return cmd
}

func modeShowCmd() *cobra.Command {
	var dir string
	var jsonOutput bool
	cmd := &cobra.Command{Use: "show", Short: "Show effective execution mode", RunE: func(cmd *cobra.Command, args []string) error {
		if dir == "" {
			dir = baseDir
		}
		store, err := sessionv2.Load(resolveArtifactsDir(dir))
		if err != nil {
			return err
		}
		state, err := sessionv2.Reduce(store)
		if err != nil {
			return err
		}
		if jsonOutput {
			return writeJSONValue(map[string]any{"mode": state.Profile.Mode, "frontier": state.Frontier, "active_sessions": sessionv2.ActiveSessions(store, state)})
		}
		fmt.Println(state.Profile.Mode)
		return nil
	}}
	cmd.Flags().StringVar(&dir, "dir", "", "Project directory")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output JSON")
	return cmd
}

func modeSetCmd() *cobra.Command {
	var dir, expected, reason, sessionID, requester string
	var jsonOutput bool
	cmd := &cobra.Command{Use: "set <solo|collaborative>", Args: cobra.ExactArgs(1), Short: "Change mode with compare-and-swap protection", RunE: func(cmd *cobra.Command, args []string) error {
		mode := strings.TrimSpace(args[0])
		if mode != "solo" && mode != "collaborative" {
			return fmt.Errorf("mode must be solo or collaborative")
		}
		if strings.TrimSpace(expected) == "" {
			return fmt.Errorf("--expect-state is required")
		}
		if strings.TrimSpace(reason) == "" {
			return fmt.Errorf("--reason is required")
		}
		if dir == "" {
			dir = baseDir
		}
		dir = resolveArtifactsDir(dir)
		store, err := sessionv2.Load(dir)
		if err != nil {
			return err
		}
		state, err := sessionv2.Reduce(store)
		if err != nil {
			return err
		}
		if state.Frontier != expected {
			return fmt.Errorf("%w: expected %s, actual %s", sessionv2.ErrStaleFrontier, expected, state.Frontier)
		}
		if state.Profile.Mode == mode {
			if jsonOutput {
				return writeJSONValue(map[string]any{"mode": mode, "frontier": state.Frontier, "changed": false})
			}
			fmt.Println(mode)
			return nil
		}
		if mode == "solo" && len(sessionv2.ActiveSessions(store, state)) > 1 {
			return fmt.Errorf("cannot enter solo mode while multiple sessions are active")
		}
		resolved, err := sessionv2.ResolveSession(dir, sessionID)
		if err != nil {
			return err
		}
		event, err := sessionv2.AppendEvent(dir, resolved, "mode_changed", map[string]any{"prior_mode": state.Profile.Mode, "new_mode": mode, "reason": strings.TrimSpace(reason), "requester": strings.TrimSpace(requester)}, sessionv2.AppendOptions{ExpectedFrontier: expected})
		if err != nil {
			return err
		}
		if jsonOutput {
			return writeJSONValue(map[string]any{"mode": mode, "changed": true, "event": event})
		}
		fmt.Printf("Mode changed to %s (%s)\n", mode, event.EventID)
		return nil
	}}
	cmd.Flags().StringVar(&dir, "dir", "", "Project directory")
	cmd.Flags().StringVar(&expected, "expect-state", "", "Required current frontier digest")
	cmd.Flags().StringVar(&reason, "reason", "", "Required reason")
	cmd.Flags().StringVar(&sessionID, "session", "", "Session id")
	cmd.Flags().StringVar(&requester, "requester", "", "Requester label")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output JSON")
	return cmd
}
