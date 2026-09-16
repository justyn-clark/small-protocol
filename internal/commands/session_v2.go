package commands

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/justyn-clark/small-protocol/internal/sessionv2"
	"github.com/justyn-clark/small-protocol/internal/version"
	"github.com/spf13/cobra"
)

func sessionCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "session", Short: "Manage SMALL v2 execution sessions"}
	cmd.AddCommand(sessionStartCmd(), sessionListCmd(), sessionCloseCmd())
	return cmd
}

func sessionStartCmd() *cobra.Command {
	var dir, label, actor, model, parent, handoff, takeover, reason string
	var jsonOutput bool
	cmd := &cobra.Command{
		Use: "start", Short: "Start an execution session",
		RunE: func(cmd *cobra.Command, args []string) error {
			if dir == "" {
				dir = baseDir
			}
			dir = resolveArtifactsDir(dir)
			session, events, err := sessionv2.StartSession(dir, sessionv2.SessionStartOptions{
				Label: label, ActorLabel: actor, ModelLabel: model, ParentSessionID: parent,
				FromHandoff: handoff, TakeoverSessionID: takeover, TakeoverReason: reason,
				ToolVersion: version.GetVersion(),
			})
			if err != nil {
				return err
			}
			if jsonOutput {
				return writeJSONValue(map[string]any{"session": session, "events": events})
			}
			fmt.Printf("Started session %s (%s mode)\n", session.SessionID, session.SmallVersion)
			return nil
		},
	}
	cmd.Flags().StringVar(&dir, "dir", "", "Project directory")
	cmd.Flags().StringVar(&label, "label", "", "Human-readable session label")
	cmd.Flags().StringVar(&actor, "actor-label", "", "Human-readable actor label")
	cmd.Flags().StringVar(&model, "model-label", "", "Human-readable model label")
	cmd.Flags().StringVar(&parent, "parent-session", "", "Parent session id")
	cmd.Flags().StringVar(&handoff, "from-handoff", "", "Handoff event id used to resume")
	cmd.Flags().StringVar(&takeover, "takeover", "", "Active session id to supersede in solo mode")
	cmd.Flags().StringVar(&reason, "reason", "", "Required reason for takeover")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output JSON")
	return cmd
}

func sessionListCmd() *cobra.Command {
	var dir string
	var jsonOutput bool
	cmd := &cobra.Command{
		Use: "list", Short: "List sessions",
		RunE: func(cmd *cobra.Command, args []string) error {
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
			ids := make([]string, 0, len(store.Sessions))
			for id := range store.Sessions {
				ids = append(ids, id)
			}
			sort.Strings(ids)
			rows := make([]map[string]any, 0, len(ids))
			for _, id := range ids {
				s := store.Sessions[id]
				status := "active"
				if state.ClosedSessions[id] {
					status = "closed"
				}
				if state.SupersededSessions[id] {
					status = "superseded"
				}
				rows = append(rows, map[string]any{"session_id": id, "label": s.Label, "status": status, "created_at": s.CreatedAt, "observed_frontier": s.ObservedFrontier})
			}
			if jsonOutput {
				return writeJSONValue(map[string]any{"mode": state.Profile.Mode, "frontier": state.Frontier, "sessions": rows})
			}
			for _, row := range rows {
				fmt.Printf("%s\t%s\t%s\n", row["session_id"], row["status"], row["label"])
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&dir, "dir", "", "Project directory")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output JSON")
	return cmd
}

func sessionCloseCmd() *cobra.Command {
	var dir, sessionID, summary string
	var jsonOutput bool
	cmd := &cobra.Command{
		Use: "close", Short: "Record a narrative handoff and close a session",
		RunE: func(cmd *cobra.Command, args []string) error {
			if dir == "" {
				dir = baseDir
			}
			dir = resolveArtifactsDir(dir)
			resolved, err := sessionv2.ResolveSession(dir, sessionID)
			if err != nil {
				return err
			}
			if strings.TrimSpace(summary) == "" {
				return fmt.Errorf("--summary is required and must be non-whitespace")
			}
			events, err := sessionv2.CloseSession(dir, resolved, summary)
			if err != nil {
				return err
			}
			if jsonOutput {
				return writeJSONValue(map[string]any{"session_id": resolved, "events": events})
			}
			fmt.Printf("Closed session %s; handoff %s\n", resolved, events[0].EventID)
			return nil
		},
	}
	cmd.Flags().StringVar(&dir, "dir", "", "Project directory")
	cmd.Flags().StringVar(&sessionID, "session", "", "Session id (defaults to local active selection)")
	cmd.Flags().StringVar(&summary, "summary", "", "Narrative handoff summary")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output JSON")
	return cmd
}

func writeJSONValue(value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}
