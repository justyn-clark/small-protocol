package small

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io/fs"
	"strings"
)

// Availability is independent of strict policy and credential inspection. The
// report deliberately excludes command text, user-authored metadata and errors
// that might contain sensitive values.
type CommandProofAvailability struct {
	Location             string `json:"location"`
	Required             bool   `json:"required"`
	Availability         string `json:"availability"`
	DisplayDigestMatches bool   `json:"display_digest_matches"`
}

type CommandProofInventory struct {
	Coverage             string                     `json:"coverage"`
	Total                int                        `json:"total"`
	Available            int                        `json:"available"`
	Missing              int                        `json:"missing"`
	Invalid              int                        `json:"invalid"`
	Unreferenced         int                        `json:"unreferenced"`
	RequiredUnavailable  int                        `json:"required_unavailable"`
	DisplayDigestMatches int                        `json:"display_digest_matches"`
	Records              []CommandProofAvailability `json:"records"`
}

func NewCommandProofInventory() CommandProofInventory {
	return CommandProofInventory{Coverage: "complete", Records: []CommandProofAvailability{}}
}

func commandProofRequired(record map[string]any) bool {
	summary, _ := record["command_summary"].(string)
	_, marked := record["command_summary_version"]
	legacyCandidate := len(summary) == DefaultCommandSummaryCap && strings.HasSuffix(summary, "...") && insecureCommandSummary(summary)
	return marked || legacyCandidate || commandHTTPLiteralsForLint(summary) != summary
}

func (inventory *CommandProofInventory) Add(baseDir, location string, record map[string]any) {
	inventory.add(baseDir, location, record, nil)
}

func (inventory *CommandProofInventory) AddEvent(baseDir, location string, record map[string]any, sourceDigest string) {
	inventory.add(baseDir, location, record, &sourceDigest)
}

func (inventory *CommandProofInventory) add(baseDir, location string, record map[string]any, sourceDigest *string) {
	summary, _ := record["command_summary"].(string)
	if summary == "" {
		summary, _ = record["command"].(string)
	}
	ref, _ := record["command_ref"].(string)
	digest, _ := record["command_sha256"].(string)
	if summary == "" && ref == "" && digest == "" && !commandProofRequired(record) {
		return
	}
	item := CommandProofAvailability{Location: location, Required: commandProofRequired(record), Availability: "unreferenced"}
	if summary != "" && commandDigestPattern.MatchString(digest) {
		sum := sha256.Sum256([]byte(summary))
		item.DisplayDigestMatches = hex.EncodeToString(sum[:]) == digest
	}
	if ref != "" || digest != "" || item.Required {
		command, err := ReadCommandProof(baseDir, ref, digest)
		switch {
		case errors.Is(err, fs.ErrNotExist):
			item.Availability = "missing"
		case err != nil:
			item.Availability = "invalid"
		case len(commandProofBindingErrors(record, command)) > 0:
			item.Availability = "invalid"
		default:
			item.Availability = "available"
		}
	}
	if sourceDigest != nil && *sourceDigest != digest {
		item.Availability = "invalid"
	}
	inventory.Total++
	switch item.Availability {
	case "available":
		inventory.Available++
	case "missing":
		inventory.Missing++
	case "invalid":
		inventory.Invalid++
	case "unreferenced":
		inventory.Unreferenced++
	}
	if item.Required && item.Availability != "available" {
		inventory.RequiredUnavailable++
	}
	if item.DisplayDigestMatches {
		inventory.DisplayDigestMatches++
	}
	if inventory.Available != inventory.Total {
		inventory.Coverage = "incomplete"
	}
	inventory.Records = append(inventory.Records, item)
}
