package commands

import (
	"fmt"

	"github.com/justyn-clark/small-protocol/internal/workspace"
)

// enforceWorkspaceScope ensures the artifact directory matches the requested scope.
func enforceWorkspaceScope(artifactsDir string, scope workspace.Scope) error {
	info, err := workspace.Load(artifactsDir)
	if err != nil {
		return err
	}

	if !scope.Allows(info.Kind) {
		return fmt.Errorf("workspace kind %q is not allowed for scope %s", info.Kind, scope)
	}

	return nil
}
