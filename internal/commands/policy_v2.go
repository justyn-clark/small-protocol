package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/justyn-clark/small-protocol/internal/sessionv2"
	"github.com/spf13/cobra"
)

func policyCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "policy", Short: "Inspect or revise SMALL v2 human-owned policy"}
	cmd.AddCommand(policyShowCmd(), policyReviseCmd())
	return cmd
}

func policyShowCmd() *cobra.Command {
	var dir string
	var jsonOutput bool
	cmd := &cobra.Command{Use: "show", RunE: func(cmd *cobra.Command, args []string) error {
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
		value := map[string]any{"profile": sessionv2.ProfileVersion, "policy_revision": state.Profile.PolicyRevision, "frontier": state.Frontier, "intent_digest": store.PolicyIntentDigest, "constraints_digest": store.PolicyConstraintsDigest, "conflicts": state.Conflicts}
		if jsonOutput {
			return writeJSONValue(value)
		}
		fmt.Printf("policy %s\nfrontier %s\n", state.Profile.PolicyRevision, state.Frontier)
		return nil
	}}
	cmd.Flags().StringVar(&dir, "dir", "", "Project directory")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output JSON")
	return cmd
}

func policyReviseCmd() *cobra.Command {
	var dir, intentPath, constraintsPath, expected, reason, requester, sessionID string
	var jsonOutput bool
	cmd := &cobra.Command{Use: "revise", Short: "Atomically record reviewed policy material", RunE: func(cmd *cobra.Command, args []string) error {
		if intentPath == "" && constraintsPath == "" {
			return fmt.Errorf("at least one of --intent or --constraints is required")
		}
		if expected == "" {
			return fmt.Errorf("--expect-state is required")
		}
		if strings.TrimSpace(reason) == "" {
			return fmt.Errorf("--reason is required")
		}
		if dir == "" {
			dir = baseDir
		}
		dir = resolveArtifactsDir(dir)
		resolved, err := sessionv2.ResolveSession(dir, sessionID)
		if err != nil {
			return err
		}
		material, err := loadPolicyMaterial(dir, intentPath, constraintsPath)
		if err != nil {
			return err
		}
		event, err := sessionv2.RevisePolicy(dir, resolved, expected, reason, requester, material)
		if err != nil {
			return err
		}
		store, err := sessionv2.Load(dir)
		if err != nil {
			return err
		}
		state, err := sessionv2.Reduce(store)
		if err != nil {
			return err
		}
		if jsonOutput {
			return writeJSONValue(map[string]any{"profile": sessionv2.ProfileVersion, "policy_revision": state.Profile.PolicyRevision, "frontier": state.Frontier, "event": event})
		}
		fmt.Printf("Policy revised to %s (%s)\n", state.Profile.PolicyRevision, event.EventID)
		return nil
	}}
	cmd.Flags().StringVar(&dir, "dir", "", "Project directory")
	cmd.Flags().StringVar(&intentPath, "intent", "", "Reviewed intent file; omit to retain current material")
	cmd.Flags().StringVar(&constraintsPath, "constraints", "", "Reviewed constraints file; omit to retain current material")
	cmd.Flags().StringVar(&expected, "expect-state", "", "Required current frontier digest")
	cmd.Flags().StringVar(&reason, "reason", "", "Required review rationale")
	cmd.Flags().StringVar(&requester, "requester", "", "Requester label")
	cmd.Flags().StringVar(&sessionID, "session", "", "Session id")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output JSON")
	return cmd
}

func loadPolicyMaterial(base, intentPath, constraintsPath string) (sessionv2.PolicyMaterial, error) {
	read := func(explicit, current string) ([]byte, bool, error) {
		path := explicit
		if path == "" {
			path = current
		}
		data, err := os.ReadFile(path)
		if os.IsNotExist(err) && explicit == "" {
			return nil, false, nil
		}
		if err != nil {
			return nil, false, err
		}
		if len(data) > sessionv2.MaxRecordBytes {
			return nil, false, fmt.Errorf("policy file exceeds %d bytes", sessionv2.MaxRecordBytes)
		}
		return data, true, nil
	}
	intent, hasIntent, err := read(intentPath, filepath.Join(base, ".small", "policy", "intent.small.yml"))
	if err != nil {
		return sessionv2.PolicyMaterial{}, err
	}
	constraints, hasConstraints, err := read(constraintsPath, filepath.Join(base, ".small", "policy", "constraints.small.yml"))
	if err != nil {
		return sessionv2.PolicyMaterial{}, err
	}
	return sessionv2.PolicyMaterial{Intent: intent, Constraints: constraints, HasIntent: hasIntent, HasConstraints: hasConstraints}, nil
}
