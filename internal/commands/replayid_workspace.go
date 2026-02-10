package commands

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/justyn-clark/small-protocol/internal/small"
	"github.com/justyn-clark/small-protocol/internal/workspace"
)

func currentWorkspaceRunReplayID(artifactsDir string) (string, error) {
	replayID, err := workspace.RunReplayID(artifactsDir)
	if err != nil {
		if isWorkspaceMissingError(err) {
			return "", nil
		}
		return "", err
	}
	return strings.TrimSpace(replayID), nil
}

func ensureWorkspaceRunReplayID(artifactsDir string) (string, error) {
	replayID, err := currentWorkspaceRunReplayID(artifactsDir)
	if err != nil {
		return "", err
	}
	if replayID != "" {
		return replayID, nil
	}

	generated, err := generateReplayId(filepath.Join(artifactsDir, small.SmallDir), "")
	if err != nil {
		return "", fmt.Errorf("replayId error: %w", err)
	}
	if err := workspace.SetRunReplayID(artifactsDir, generated.Value); err != nil {
		if !isWorkspaceMissingError(err) {
			return "", fmt.Errorf("failed to persist workspace run replay_id: %w", err)
		}
	}
	return generated.Value, nil
}

func setWorkspaceRunReplayIDIfPresent(artifactsDir, replayID string) error {
	if strings.TrimSpace(replayID) == "" {
		return nil
	}
	if err := workspace.SetRunReplayID(artifactsDir, replayID); err != nil {
		if isWorkspaceMissingError(err) {
			return nil
		}
		return err
	}
	return nil
}

func isWorkspaceMissingError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "workspace metadata missing:")
}
