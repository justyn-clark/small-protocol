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
	Owner        string `yaml:"owner,omitempty"`
	Kind         Kind   `yaml:"kind"`
	CreatedAt    string `yaml:"created_at,omitempty"`
	UpdatedAt    string `yaml:"updated_at,omitempty"`
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

	now := time.Now().UTC().Format(time.RFC3339Nano)
	info := Info{
		SmallVersion: small.ProtocolVersion,
		Owner:        "agent",
		Kind:         kind,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	data, err := small.MarshalYAMLWithQuotedVersion(info)
	if err != nil {
		return fmt.Errorf("failed to marshal workspace metadata: %w", err)
	}

	if err := os.WriteFile(filepath.Join(smallDir, "workspace.small.yml"), data, 0644); err != nil {
		return fmt.Errorf("failed to write workspace metadata: %w", err)
	}

	return nil
}

// FixResult describes what was fixed in the workspace file.
type FixResult struct {
	Created          bool
	AddedOwner       bool
	AddedCreatedAt   bool
	AddedUpdatedAt   bool
	NormalizedFormat bool
}

// Fix creates or repairs workspace.small.yml.
// If force is true, it will overwrite all fields. Otherwise it only touches missing/invalid timestamps.
func Fix(baseDir string, kind Kind, force bool) (FixResult, error) {
	var result FixResult

	if !IsValidKind(kind) {
		return result, fmt.Errorf("cannot fix workspace metadata for invalid kind %q", kind)
	}

	smallDir := filepath.Join(baseDir, small.SmallDir)
	path := filepath.Join(smallDir, "workspace.small.yml")

	// Try to load existing workspace
	existing, loadErr := Load(baseDir)
	now := time.Now().UTC().Format(time.RFC3339Nano)

	if loadErr != nil {
		// File missing or invalid - create new
		if err := os.MkdirAll(smallDir, 0755); err != nil {
			return result, fmt.Errorf("failed to create .small directory: %w", err)
		}
		result.Created = true
		result.AddedOwner = true
		result.AddedCreatedAt = true
		result.AddedUpdatedAt = true

		info := Info{
			SmallVersion: small.ProtocolVersion,
			Owner:        "agent",
			Kind:         kind,
			CreatedAt:    now,
			UpdatedAt:    now,
		}

		data, err := small.MarshalYAMLWithQuotedVersion(info)
		if err != nil {
			return result, fmt.Errorf("failed to marshal workspace metadata: %w", err)
		}

		if err := os.WriteFile(path, data, 0644); err != nil {
			return result, fmt.Errorf("failed to write workspace metadata: %w", err)
		}

		return result, nil
	}

	// File exists, repair as needed
	info := existing
	needsWrite := false

	// Always set version to current
	if info.SmallVersion != small.ProtocolVersion {
		info.SmallVersion = small.ProtocolVersion
		result.NormalizedFormat = true
		needsWrite = true
	}

	// Add owner if missing
	if info.Owner == "" {
		info.Owner = "agent"
		result.AddedOwner = true
		needsWrite = true
	}

	// Add createdAt if missing
	if info.CreatedAt == "" {
		info.CreatedAt = now
		result.AddedCreatedAt = true
		needsWrite = true
	} else if !isValidRFC3339(info.CreatedAt) {
		// Normalize invalid timestamp
		info.CreatedAt = now
		result.AddedCreatedAt = true
		result.NormalizedFormat = true
		needsWrite = true
	}

	// Always update updatedAt (set to now)
	if info.UpdatedAt == "" || force {
		info.UpdatedAt = now
		result.AddedUpdatedAt = true
		needsWrite = true
	} else if !isValidRFC3339(info.UpdatedAt) {
		info.UpdatedAt = now
		result.AddedUpdatedAt = true
		result.NormalizedFormat = true
		needsWrite = true
	}

	// Update kind if force is set
	if force && info.Kind != kind {
		info.Kind = kind
		needsWrite = true
	}

	if !needsWrite {
		return result, nil
	}

	data, err := small.MarshalYAMLWithQuotedVersion(info)
	if err != nil {
		return result, fmt.Errorf("failed to marshal workspace metadata: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return result, fmt.Errorf("failed to write workspace metadata: %w", err)
	}

	return result, nil
}

// isValidRFC3339 checks if a timestamp string is valid RFC3339.
func isValidRFC3339(ts string) bool {
	_, err := time.Parse(time.RFC3339Nano, ts)
	if err != nil {
		_, err = time.Parse(time.RFC3339, ts)
	}
	return err == nil
}
