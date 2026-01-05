package small

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/santhosh-tekuri/jsonschema/v5"
	"gopkg.in/yaml.v3"
)

var (
	schemaCache = make(map[string]*jsonschema.Schema)
)

func getSchemaPath(baseDir, artifactType string) string {
	// Try v1.0.0 first, fall back to v0.1 for backwards compatibility
	v1Path := filepath.Join(baseDir, "spec", "small", "v1.0.0", "schemas", artifactType+".schema.json")
	if _, err := os.Stat(v1Path); err == nil {
		return v1Path
	}
	// Fallback to v0.1 schemas during transition
	return filepath.Join(baseDir, "spec", "small", "v0.1", "schemas", artifactType+".schema.json")
}

func ValidateArtifact(artifact *Artifact, baseDir string) error {
	schemaPath := getSchemaPath(baseDir, artifact.Type)

	schema, ok := schemaCache[schemaPath]
	if !ok {
		var err error
		compiler := jsonschema.NewCompiler()
		compiler.Draft = jsonschema.Draft2020
		schema, err = compiler.Compile(schemaPath)
		if err != nil {
			return fmt.Errorf("failed to compile schema %s: %w", schemaPath, err)
		}
		schemaCache[schemaPath] = schema
	}

	yamlData, err := yaml.Marshal(artifact.Data)
	if err != nil {
		return fmt.Errorf("failed to marshal YAML: %w", err)
	}

	var jsonData interface{}
	if err := yaml.Unmarshal(yamlData, &jsonData); err != nil {
		return fmt.Errorf("failed to convert YAML to JSON: %w", err)
	}

	if err := schema.Validate(jsonData); err != nil {
		if validationError, ok := err.(*jsonschema.ValidationError); ok {
			return formatValidationError(validationError, artifact.Path)
		}
		return fmt.Errorf("validation failed for %s: %w", artifact.Path, err)
	}

	return nil
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

func ValidateAllArtifacts(artifacts map[string]*Artifact, baseDir string) []error {
	var errors []error

	for _, artifact := range artifacts {
		if err := ValidateArtifact(artifact, baseDir); err != nil {
			errors = append(errors, err)
		}
	}

	return errors
}

func YAMLToJSON(yamlData []byte) ([]byte, error) {
	var data interface{}
	if err := yaml.Unmarshal(yamlData, &data); err != nil {
		return nil, err
	}
	return json.Marshal(data)
}
