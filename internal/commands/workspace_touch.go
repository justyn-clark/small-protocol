package commands

import (
	"fmt"

	"github.com/justyn-clark/small-protocol/internal/workspace"
)

func touchWorkspaceUpdatedAt(baseDir string) error {
	_, err := workspace.TouchUpdatedAt(baseDir)
	if err != nil {
		return fmt.Errorf("failed to update workspace.updated_at: %w", err)
	}
	return nil
}
