package small

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// RunIndexEntry represents a single run lineage entry.
type RunIndexEntry struct {
	ReplayID         string `yaml:"replayId"`
	Timestamp        string `yaml:"timestamp"`
	GitSHA           string `yaml:"git_sha,omitempty"`
	Summary          string `yaml:"summary,omitempty"`
	PreviousReplayID string `yaml:"previous_replay_id,omitempty"`
	Reason           string `yaml:"reason"`
}

// RunIndex is an append-only ledger for run lineage.
type RunIndex struct {
	SmallVersion string          `yaml:"small_version"`
	Entries      []RunIndexEntry `yaml:"entries"`
}

// AppendRunIndexEntry appends an entry to .small/runs/index.small.yml.
func AppendRunIndexEntry(baseDir string, entry RunIndexEntry) error {
	if strings.TrimSpace(entry.ReplayID) == "" {
		return fmt.Errorf("run index entry requires replayId")
	}
	if strings.TrimSpace(entry.Timestamp) == "" {
		return fmt.Errorf("run index entry requires timestamp")
	}
	if strings.TrimSpace(entry.Reason) == "" {
		return fmt.Errorf("run index entry requires reason")
	}

	smallDir := filepath.Join(baseDir, SmallDir)
	runsDir := filepath.Join(smallDir, "runs")
	path := filepath.Join(runsDir, "index.small.yml")

	index := RunIndex{
		SmallVersion: ProtocolVersion,
	}

	if data, err := os.ReadFile(path); err == nil {
		if err := yaml.Unmarshal(data, &index); err != nil {
			return fmt.Errorf("failed to parse run index: %w", err)
		}
		if index.SmallVersion == "" {
			index.SmallVersion = ProtocolVersion
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("failed to read run index: %w", err)
	}

	if entry.PreviousReplayID == "" && len(index.Entries) > 0 {
		entry.PreviousReplayID = index.Entries[len(index.Entries)-1].ReplayID
	}

	index.Entries = append(index.Entries, entry)

	if err := os.MkdirAll(runsDir, 0755); err != nil {
		return fmt.Errorf("failed to create runs directory: %w", err)
	}

	data, err := MarshalYAMLWithQuotedVersion(&index)
	if err != nil {
		return fmt.Errorf("failed to marshal run index: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write run index: %w", err)
	}
	return nil
}
