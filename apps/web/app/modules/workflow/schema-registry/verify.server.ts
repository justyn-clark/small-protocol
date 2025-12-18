import protocol from "~/modules/protocol/small.protocol.json";
import { getAjv } from "./registry.server";

const EXAMPLE_MANIFEST = {
	artifact: "track.audio",
	schema: "artifact.v1",
	version: 1,
};

const META_SCHEMA_ID = "https://json-schema.org/draft/2020-12/schema";

/**
 * Verify schema registry is properly initialized and all required schemas are present.
 * Validates the example manifest used by the playground.
 *
 * Throws with clear errors if anything is missing or invalid.
 */
export function verifySchemaRegistry(): void {
	const ajv = getAjv();
	const errors: string[] = [];

	// Verify meta-schema is registered
	const metaSchema = ajv.getSchema(META_SCHEMA_ID);
	if (!metaSchema) {
		errors.push(
			`Meta-schema not found: ${META_SCHEMA_ID}. Ensure draft2020-12.schema.json is synced.`,
		);
	}

	// Verify all SMALL schemas from protocol are registered
	const schemaPaths = Object.values(protocol.schemas);
	for (const schemaPath of schemaPaths) {
		// Extract schema name from path (e.g., "/schemas/small/v1/manifest.schema.json" -> "manifest")
		const schemaName = schemaPath.split("/").pop()?.replace(".schema.json", "");
		if (!schemaName) continue;

		// Construct expected $id (with localhost origin after sync)
		const expectedId = `http://localhost:5173${schemaPath}`;
		const schema = ajv.getSchema(expectedId);

		if (!schema) {
			errors.push(
				`Schema not found: ${expectedId}. Ensure ${schemaName}.schema.json is synced.`,
			);
		}
	}

	// Validate example manifest
	try {
		const manifestSchemaId = `http://localhost:5173${protocol.schemas.manifest}`;
		const validate = ajv.getSchema(manifestSchemaId);
		if (validate) {
			const valid = validate(EXAMPLE_MANIFEST);
			if (!valid) {
				errors.push(
					`Example manifest validation failed: ${JSON.stringify(validate.errors)}`,
				);
			}
		} else {
			errors.push(`Manifest schema not found: ${manifestSchemaId}`);
		}
	} catch (error) {
		errors.push(
			`Failed to validate example manifest: ${error instanceof Error ? error.message : String(error)}`,
		);
	}

	if (errors.length > 0) {
		throw new Error(
			`Schema registry verification failed:\n${errors.map((e) => `  - ${e}`).join("\n")}\n\nRun 'bun run sync-schemas' to sync schemas.`,
		);
	}
}
