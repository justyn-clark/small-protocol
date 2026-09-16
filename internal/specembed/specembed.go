// Package specembed provides embedded SMALL protocol v1.0.0 and v2.0.0 schemas for runtime validation.
// This allows the CLI to validate artifacts without requiring the spec directory on disk.
//
// The schemas are copied from spec/small/v1.0.0/schemas/ during development.
// The authoritative source remains spec/small/v1.0.0/schemas/.
package specembed

import (
	"embed"
	"io/fs"
)

// schemas embeds the v1.0.0 JSON schema files.
//
//go:embed schemas/*.schema.json schemas/v2/*.schema.json
var schemas embed.FS

// FS returns the embedded filesystem containing the v1.0.0 schemas.
// Schema files are located at "schemas/<type>.schema.json".
func FS() fs.FS {
	return schemas
}

// SchemaPath returns the path to a schema file within the embedded FS.
func SchemaPath(artifactType string) string {
	return "schemas/" + artifactType + ".schema.json"
}

// ReadSchema reads a schema file from the embedded FS.
func ReadSchema(artifactType string) ([]byte, error) {
	return fs.ReadFile(schemas, SchemaPath(artifactType))
}

// ReadV2Schema reads a SMALL 2.0.0 session-profile schema.
func ReadV2Schema(recordType string) ([]byte, error) {
	return fs.ReadFile(schemas, "schemas/v2/"+recordType+".schema.json")
}
