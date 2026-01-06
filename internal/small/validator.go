package small

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/justyn-clark/small-protocol/internal/specembed"
	"github.com/santhosh-tekuri/jsonschema/v5"
	"gopkg.in/yaml.v3"
)

// SchemaSource indicates where schemas are loaded from.
type SchemaSource int

const (
	// SchemaSourceDisk loads schemas from disk (spec-dir or dev mode).
	SchemaSourceDisk SchemaSource = iota
	// SchemaSourceEmbedded loads schemas from embedded FS.
	SchemaSourceEmbedded
)

// SchemaConfig holds configuration for schema resolution.
type SchemaConfig struct {
	// SpecDir is an explicit path to the spec directory (contains small/v1.0.0/schemas/).
	// If set, this takes precedence over all other sources.
	SpecDir string

	// BaseDir is the directory to start searching for on-disk schemas.
	// Used for dev mode detection (finding spec/small/v1.0.0/schemas/).
	BaseDir string
}

// ResolvedSchema contains a loaded schema and metadata about its source.
type ResolvedSchema struct {
	Schema *jsonschema.Schema
	Source SchemaSource
	Path   string // Filesystem path or embedded path
}

var (
	schemaCache = make(map[string]*ResolvedSchema)
)

// getSchemaPath returns the filesystem path for an artifact schema.
func getSchemaPath(baseDir, artifactType string) string {
	return filepath.Join(baseDir, "spec", "small", "v1.0.0", "schemas", artifactType+".schema.json")
}

// findSpecDir looks for on-disk schemas starting from baseDir and walking up.
// Returns the repo root containing spec/small/v1.0.0/schemas, or empty string if not found.
func findSpecDir(baseDir string) string {
	dir := baseDir
	for {
		specPath := filepath.Join(dir, "spec", "small", "v1.0.0", "schemas")
		if info, err := os.Stat(specPath); err == nil && info.IsDir() {
			return dir
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

// compileSchemaFromDisk compiles a schema from a filesystem path.
func compileSchemaFromDisk(schemaPath string) (*jsonschema.Schema, error) {
	compiler := jsonschema.NewCompiler()
	compiler.Draft = jsonschema.Draft2020
	return compiler.Compile(schemaPath)
}

// compileSchemaFromEmbedded compiles a schema from the embedded filesystem.
func compileSchemaFromEmbedded(artifactType string) (*jsonschema.Schema, error) {
	schemaBytes, err := specembed.ReadSchema(artifactType)
	if err != nil {
		return nil, fmt.Errorf("failed to read embedded schema for %s: %w", artifactType, err)
	}

	// Use a stable URL for the embedded schema
	schemaURL := fmt.Sprintf("small://v1.0.0/schemas/%s.schema.json", artifactType)

	compiler := jsonschema.NewCompiler()
	compiler.Draft = jsonschema.Draft2020

	if err := compiler.AddResource(schemaURL, strings.NewReader(string(schemaBytes))); err != nil {
		return nil, fmt.Errorf("failed to add embedded schema resource: %w", err)
	}

	return compiler.Compile(schemaURL)
}

// LoadSchema resolves and compiles a schema for the given artifact type.
// Resolution order:
//  1. If config.SpecDir is set, load from that directory
//  2. Else if on-disk schemas found (dev mode), load from disk
//  3. Else load from embedded schemas
func LoadSchema(artifactType string, config SchemaConfig) (*ResolvedSchema, error) {
	// Build cache key based on config
	cacheKey := fmt.Sprintf("%s:%s:%s", artifactType, config.SpecDir, config.BaseDir)

	if cached, ok := schemaCache[cacheKey]; ok {
		return cached, nil
	}

	var resolved *ResolvedSchema

	// 1. Check explicit --spec-dir
	if config.SpecDir != "" {
		schemaPath := filepath.Join(config.SpecDir, "small", "v1.0.0", "schemas", artifactType+".schema.json")
		if _, err := os.Stat(schemaPath); err == nil {
			schema, err := compileSchemaFromDisk(schemaPath)
			if err != nil {
				return nil, fmt.Errorf("failed to compile schema from spec-dir: %w", err)
			}
			resolved = &ResolvedSchema{
				Schema: schema,
				Source: SchemaSourceDisk,
				Path:   schemaPath,
			}
		} else {
			return nil, fmt.Errorf("schema not found at --spec-dir path: %s", schemaPath)
		}
	}

	// 2. Check for on-disk schemas (dev mode)
	if resolved == nil && config.BaseDir != "" {
		if repoRoot := findSpecDir(config.BaseDir); repoRoot != "" {
			schemaPath := getSchemaPath(repoRoot, artifactType)
			schema, err := compileSchemaFromDisk(schemaPath)
			if err != nil {
				return nil, fmt.Errorf("failed to compile on-disk schema: %w", err)
			}
			resolved = &ResolvedSchema{
				Schema: schema,
				Source: SchemaSourceDisk,
				Path:   schemaPath,
			}
		}
	}

	// 3. Fall back to embedded schemas
	if resolved == nil {
		schema, err := compileSchemaFromEmbedded(artifactType)
		if err != nil {
			return nil, err
		}
		resolved = &ResolvedSchema{
			Schema: schema,
			Source: SchemaSourceEmbedded,
			Path:   specembed.SchemaPath(artifactType),
		}
	}

	schemaCache[cacheKey] = resolved
	return resolved, nil
}

// ValidateArtifactWithConfig validates an artifact using the given schema config.
func ValidateArtifactWithConfig(artifact *Artifact, config SchemaConfig) error {
	resolved, err := LoadSchema(artifact.Type, config)
	if err != nil {
		return err
	}

	yamlData, err := yaml.Marshal(artifact.Data)
	if err != nil {
		return fmt.Errorf("failed to marshal YAML: %w", err)
	}

	var jsonData interface{}
	if err := yaml.Unmarshal(yamlData, &jsonData); err != nil {
		return fmt.Errorf("failed to convert YAML to JSON: %w", err)
	}

	if err := resolved.Schema.Validate(jsonData); err != nil {
		if validationError, ok := err.(*jsonschema.ValidationError); ok {
			return formatValidationError(validationError, artifact.Path)
		}
		return fmt.Errorf("validation failed for %s: %w", artifact.Path, err)
	}

	return nil
}

// ValidateArtifact validates an artifact against its JSON schema.
// This is the legacy function for backward compatibility.
// Deprecated: Use ValidateArtifactWithConfig for explicit schema resolution control.
func ValidateArtifact(artifact *Artifact, baseDir string) error {
	return ValidateArtifactWithConfig(artifact, SchemaConfig{BaseDir: baseDir})
}

func formatValidationError(err *jsonschema.ValidationError, filePath string) error {
	if len(err.Causes) == 0 {
		return fmt.Errorf("%s: %s", filePath, err.Message)
	}

	var messages []string
	for _, cause := range err.Causes {
		messages = append(messages, fmt.Sprintf("  %s: %s", cause.InstanceLocation, cause.Message))
	}

	return fmt.Errorf("%s:\n%s", filePath, fmt.Sprint(messages))
}

// ValidateAllArtifactsWithConfig validates all artifacts using the given schema config.
func ValidateAllArtifactsWithConfig(artifacts map[string]*Artifact, config SchemaConfig) []error {
	var errors []error

	for _, artifact := range artifacts {
		if err := ValidateArtifactWithConfig(artifact, config); err != nil {
			errors = append(errors, err)
		}
	}

	return errors
}

// ValidateAllArtifacts validates all artifacts against their JSON schemas.
// Deprecated: Use ValidateAllArtifactsWithConfig for explicit schema resolution control.
func ValidateAllArtifacts(artifacts map[string]*Artifact, baseDir string) []error {
	return ValidateAllArtifactsWithConfig(artifacts, SchemaConfig{BaseDir: baseDir})
}

func YAMLToJSON(yamlData []byte) ([]byte, error) {
	var data interface{}
	if err := yaml.Unmarshal(yamlData, &data); err != nil {
		return nil, err
	}
	return json.Marshal(data)
}

// DescribeSchemaResolution returns a human-readable description of how schemas are resolved.
func DescribeSchemaResolution(config SchemaConfig) string {
	if config.SpecDir != "" {
		return fmt.Sprintf("using schemas from --spec-dir: %s", config.SpecDir)
	}

	if config.BaseDir != "" {
		if repoRoot := findSpecDir(config.BaseDir); repoRoot != "" {
			return fmt.Sprintf("using on-disk schemas from: %s (dev mode)", filepath.Join(repoRoot, "spec", "small", "v1.0.0", "schemas"))
		}
	}

	return "using embedded v1.0.0 schemas"
}
