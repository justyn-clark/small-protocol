package specembed

import (
	"testing"
)

func TestReadSchema(t *testing.T) {
	artifactTypes := []string{"intent", "constraints", "plan", "progress", "handoff"}

	for _, artifactType := range artifactTypes {
		t.Run(artifactType, func(t *testing.T) {
			data, err := ReadSchema(artifactType)
			if err != nil {
				t.Fatalf("failed to read %s schema: %v", artifactType, err)
			}
			if len(data) == 0 {
				t.Errorf("schema for %s is empty", artifactType)
			}
			// Basic sanity check that it's valid JSON
			if data[0] != '{' {
				t.Errorf("schema for %s doesn't start with '{': got %c", artifactType, data[0])
			}
		})
	}
}

func TestSchemaPath(t *testing.T) {
	tests := []struct {
		artifactType string
		want         string
	}{
		{"intent", "schemas/intent.schema.json"},
		{"constraints", "schemas/constraints.schema.json"},
		{"plan", "schemas/plan.schema.json"},
		{"progress", "schemas/progress.schema.json"},
		{"handoff", "schemas/handoff.schema.json"},
	}

	for _, tt := range tests {
		t.Run(tt.artifactType, func(t *testing.T) {
			got := SchemaPath(tt.artifactType)
			if got != tt.want {
				t.Errorf("SchemaPath(%s) = %s, want %s", tt.artifactType, got, tt.want)
			}
		})
	}
}
