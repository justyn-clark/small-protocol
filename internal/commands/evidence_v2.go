package commands

import (
	"fmt"
	"strings"

	"github.com/justyn-clark/small-protocol/internal/sessionv2"
	"github.com/spf13/cobra"
)

func evidenceCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "evidence", Short: "Save and verify explicit v2 evidence receipts"}
	cmd.AddCommand(evidenceSaveCmd(), evidenceVerifyCmd())
	cmd.AddCommand(evidenceUnavailableCmd())
	return cmd
}

func evidenceUnavailableCmd() *cobra.Command {
	var dir, taskRef, reason string
	var jsonOutput bool
	cmd := &cobra.Command{Use: "declare-unavailable", Short: "Record that private or external evidence is unavailable here", RunE: func(cmd *cobra.Command, args []string) error {
		if dir == "" {
			dir = baseDir
		}
		dir = resolveArtifactsDir(dir)
		if strings.TrimSpace(reason) == "" {
			return fmt.Errorf("--reason is required")
		}
		store, err := sessionv2.Load(dir)
		if err != nil {
			return err
		}
		state, err := sessionv2.Reduce(store)
		if err != nil {
			return err
		}
		id, err := sessionv2.NewID("receipt")
		if err != nil {
			return err
		}
		receipt := sessionv2.Receipt{SmallVersion: sessionv2.ProfileVersion, ProjectID: store.Profile.ProjectID, ReceiptID: id, Strength: "narrative_assertion", Availability: "unavailable", Producer: map[string]any{"tool": "small evidence declare-unavailable"}, Metadata: map[string]any{"reason": strings.TrimSpace(reason)}}
		if taskRef != "" {
			task, err := resolveV2Task(state, taskRef)
			if err != nil {
				return err
			}
			receipt.TaskID = task.ID
			receipt.TaskRevision = task.Revision
			receipt.PolicyRevision = task.PolicyRevision
			receipt.SourceDigest = task.SourceDigest
		}
		receipt.Digest, err = sessionv2.DigestRecord(receipt)
		if err != nil {
			return err
		}
		if _, err := sessionv2.PublishReceipt(dir, receipt); err != nil {
			return err
		}
		if jsonOutput {
			return writeJSONValue(receipt)
		}
		fmt.Printf("Recorded unavailable evidence %s\n", receipt.Digest)
		return nil
	}}
	cmd.Flags().StringVar(&dir, "dir", "", "Project directory")
	cmd.Flags().StringVar(&taskRef, "task", "", "Task id or alias")
	cmd.Flags().StringVar(&reason, "reason", "", "Why evidence is unavailable in this checkout")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output JSON")
	return cmd
}

func evidenceSaveCmd() *cobra.Command {
	var dir, file, taskRef, validator, outcome string
	var jsonOutput bool
	cmd := &cobra.Command{Use: "save", Short: "Save a portable content-addressed evidence artifact", RunE: func(cmd *cobra.Command, args []string) error {
		if dir == "" {
			dir = baseDir
		}
		dir = resolveArtifactsDir(dir)
		if strings.TrimSpace(file) == "" {
			return fmt.Errorf("--file is required")
		}
		store, err := sessionv2.Load(dir)
		if err != nil {
			return err
		}
		state, err := sessionv2.Reduce(store)
		if err != nil {
			return err
		}
		receipt := sessionv2.Receipt{Validator: strings.TrimSpace(validator), Outcome: strings.TrimSpace(outcome), Producer: map[string]any{"tool": "small evidence save"}}
		if taskRef != "" {
			task, err := resolveV2Task(state, taskRef)
			if err != nil {
				return err
			}
			receipt.TaskID = task.ID
			receipt.TaskRevision = task.Revision
			receipt.PolicyRevision = task.PolicyRevision
			receipt.SourceDigest = task.SourceDigest
		}
		saved, err := sessionv2.SavePortableEvidence(dir, file, receipt)
		if err != nil {
			return err
		}
		if jsonOutput {
			return writeJSONValue(saved)
		}
		fmt.Printf("Saved evidence %s artifact %s\n", saved.Digest, saved.ArtifactDigest)
		return nil
	}}
	cmd.Flags().StringVar(&dir, "dir", "", "Project directory")
	cmd.Flags().StringVar(&file, "file", "", "Regular file to copy into portable evidence storage")
	cmd.Flags().StringVar(&taskRef, "task", "", "Task id or alias")
	cmd.Flags().StringVar(&validator, "validator", "", "Validator name")
	cmd.Flags().StringVar(&outcome, "outcome", "passed", "Validation outcome")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output JSON")
	return cmd
}

func evidenceVerifyCmd() *cobra.Command {
	var dir string
	var jsonOutput bool
	cmd := &cobra.Command{Use: "verify", Short: "Verify receipt and portable artifact availability", RunE: func(cmd *cobra.Command, args []string) error {
		if dir == "" {
			dir = baseDir
		}
		results, err := sessionv2.VerifyEvidence(resolveArtifactsDir(dir))
		if err != nil {
			return err
		}
		failed := false
		for _, result := range results {
			if result.Status != "verified" {
				failed = true
			}
		}
		if jsonOutput {
			if err := writeJSONValue(map[string]any{"profile": sessionv2.ProfileVersion, "receipts": results, "all_verified": !failed}); err != nil {
				return err
			}
		} else {
			for _, result := range results {
				fmt.Printf("%s\t%s\t%s\n", result.ReceiptDigest, result.Status, result.Reason)
			}
		}
		if failed {
			return fmt.Errorf("one or more evidence receipts are unavailable or invalid")
		}
		return nil
	}}
	cmd.Flags().StringVar(&dir, "dir", "", "Project directory")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output JSON")
	return cmd
}
