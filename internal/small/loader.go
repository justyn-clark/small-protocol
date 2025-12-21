package small

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const (
	SmallDir = ".small"
)

var (
	CanonicalFiles = []string{
		"intent.small.yml",
		"constraints.small.yml",
		"plan.small.yml",
		"progress.small.yml",
		"handoff.small.yml",
	}
)

type Artifact struct {
	Data map[string]interface{}
	Path string
	Type string
}

func LoadArtifact(baseDir, filename string) (*Artifact, error) {
	path := filepath.Join(baseDir, SmallDir, filename)

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", path, err)
	}

	var artifact map[string]interface{}
	if err := yaml.Unmarshal(data, &artifact); err != nil {
		return nil, fmt.Errorf("failed to parse YAML in %s: %w", path, err)
	}

	artifactType := filename[:len(filename)-len(".small.yml")]

	return &Artifact{
		Data: artifact,
		Path: path,
		Type: artifactType,
	}, nil
}

func LoadAllArtifacts(baseDir string) (map[string]*Artifact, error) {
	artifacts := make(map[string]*Artifact)

	for _, filename := range CanonicalFiles {
		artifact, err := LoadArtifact(baseDir, filename)
		if err != nil {
			return nil, err
		}
		artifacts[artifact.Type] = artifact
	}

	return artifacts, nil
}

func SaveArtifact(baseDir, filename string, data map[string]interface{}) error {
	smallDir := filepath.Join(baseDir, SmallDir)
	if err := os.MkdirAll(smallDir, 0755); err != nil {
		return fmt.Errorf("failed to create .small directory: %w", err)
	}

	path := filepath.Join(smallDir, filename)

	yamlData, err := yaml.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal YAML: %w", err)
	}

	if err := os.WriteFile(path, yamlData, 0644); err != nil {
		return fmt.Errorf("failed to write %s: %w", path, err)
	}

	return nil
}

func ArtifactExists(baseDir, filename string) bool {
	path := filepath.Join(baseDir, SmallDir, filename)
	_, err := os.Stat(path)
	return err == nil
}
