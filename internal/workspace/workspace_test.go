package workspace

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/justyn-clark/small-protocol/internal/small"
)

func TestLoadInvalidKindIncludesValidKinds(t *testing.T) {
	tmpDir := t.TempDir()
	smallDir := filepath.Join(tmpDir, small.SmallDir)
	if err := os.MkdirAll(smallDir, 0755); err != nil {
		t.Fatalf("failed to create %s: %v", smallDir, err)
	}

	content := fmt.Sprintf("small_version: %q\nkind: invalid-kind\n", small.ProtocolVersion)
	if err := os.WriteFile(filepath.Join(smallDir, "workspace.small.yml"), []byte(content), 0644); err != nil {
		t.Fatalf("failed to write workspace metadata: %v", err)
	}

	_, err := Load(tmpDir)
	if err == nil {
		t.Fatalf("expected error loading workspace metadata with invalid kind")
	}

	errMsg := err.Error()
	if !strings.Contains(errMsg, "valid kinds") {
		t.Fatalf("error should mention valid kinds, got %q", errMsg)
	}

	if !strings.Contains(errMsg, string(KindRepoRoot)) {
		t.Fatalf("error should include %q, got %q", KindRepoRoot, errMsg)
	}

	if !strings.Contains(errMsg, string(KindExamples)) {
		t.Fatalf("error should include %q, got %q", KindExamples, errMsg)
	}
}
