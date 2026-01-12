package workspace

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/justyn-clark/small-protocol/internal/small"
	"gopkg.in/yaml.v3"
)

// Kind describes the type of workspace.
type Kind string

const (
	KindRepoRoot Kind = "repo-root"
	KindExamples Kind = "examples"
)

var knownKinds = []Kind{
	KindRepoRoot,
	KindExamples,
}

// Scope controls which workspace kinds are permitted via CLI flags.
type Scope string

const (
	ScopeRoot     Scope = "root"
	ScopeExamples Scope = "examples"
	ScopeAny      Scope = "any"
)

// Info describes the workspace metadata declared inside .small/workspace.small.yml.
type Info struct {
	SmallVersion string `yaml:"small_version"`
	Kind         Kind   `yaml:"kind"`
	CreatedAt    string `yaml:"created_at,omitempty"`
}

// ParseScope converts a CLI value into a Scope value.
func ParseScope(value string) (Scope, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "root":
		return ScopeRoot, nil
	case "examples":
		return ScopeExamples, nil
	case "any":
		return ScopeAny, nil
	default:
		return "", fmt.Errorf("invalid workspace scope: %s", value)
	}
}

// Allows reports whether the scope accepts the provided kind.
func (s Scope) Allows(kind Kind) bool {
	switch s {
	case ScopeAny:
		return true
	case ScopeRoot:
		return kind == KindRepoRoot
	case ScopeExamples:
		return kind == KindExamples
	default:
		return false
	}
}

// IsValidKind reports whether the workspace kind is known.
func IsValidKind(kind Kind) bool {
	for _, known := range knownKinds {
		if kind == known {
			return true
		}
	}
	return false
}

func validKindListString() string {
	parts := make([]string, len(knownKinds))
	for i, kind := range knownKinds {
		parts[i] = strconv.Quote(string(kind))
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

// Load reads workspace metadata from the .small directory.
func Load(baseDir string) (Info, error) {
	path := filepath.Join(baseDir, small.SmallDir, "workspace.small.yml")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Info{}, fmt.Errorf("workspace metadata missing: %s", path)
		}
		return Info{}, fmt.Errorf("failed to read workspace metadata: %w", err)
	}

	var info Info
	if err := yaml.Unmarshal(data, &info); err != nil {
		return Info{}, fmt.Errorf("failed to parse workspace metadata: %w", err)
	}

	if !IsValidKind(info.Kind) {
		return Info{}, fmt.Errorf("invalid workspace kind %q; valid kinds: %s", info.Kind, validKindListString())
	}

	return info, nil
}

// Save writes workspace metadata to the .small directory.
func Save(baseDir string, kind Kind) error {
	if !IsValidKind(kind) {
		return fmt.Errorf("cannot save workspace metadata for invalid kind %q", kind)
	}

	smallDir := filepath.Join(baseDir, small.SmallDir)
	if err := os.MkdirAll(smallDir, 0755); err != nil {
		return fmt.Errorf("failed to create .small directory: %w", err)
	}

	info := Info{
		SmallVersion: small.ProtocolVersion,
		Kind:         kind,
		CreatedAt:    time.Now().UTC().Format(time.RFC3339Nano),
	}

	data, err := yaml.Marshal(info)
	if err != nil {
		return fmt.Errorf("failed to marshal workspace metadata: %w", err)
	}

	if err := os.WriteFile(filepath.Join(smallDir, "workspace.small.yml"), data, 0644); err != nil {
		return fmt.Errorf("failed to write workspace metadata: %w", err)
	}

	return nil
}
