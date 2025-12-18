import { fileURLToPath } from "node:url";
import Ajv2020, { type ErrorObject } from "ajv/dist/2020";
import addFormats from "ajv-formats";
import { existsSync, readdirSync, readFileSync } from "fs";
import { dirname, join } from "path";

declare global {
	// eslint-disable-next-line no-var
	var __smallAjv: Ajv2020 | undefined;
}

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);
// Resolve public/schemas relative to the app directory
// From: apps/web/app/modules/workflow/schema-registry/
// To: apps/web/public/schemas (up 4 levels: schema-registry -> workflow -> modules -> app -> web)
const SCHEMAS_ROOT = join(__dirname, "../../../../public/schemas");

const META_SCHEMA_ID = "https://json-schema.org/draft/2020-12/schema";

function getAjv(): Ajv2020 {
	if (globalThis.__smallAjv) {
		return globalThis.__smallAjv;
	}

	const ajv = new Ajv2020({
		strict: true,
		allErrors: true,
		addUsedSchema: false, // Prevent errors on accidental re-adds
	});

	addFormats(ajv);

	// Load and register meta-schema
	const metaSchemaPath = join(SCHEMAS_ROOT, "meta/draft2020-12.schema.json");
	if (existsSync(metaSchemaPath)) {
		const metaSchema = JSON.parse(readFileSync(metaSchemaPath, "utf-8"));
		// Only register if not already available
		if (!ajv.getSchema(META_SCHEMA_ID)) {
			ajv.addSchema(metaSchema, META_SCHEMA_ID);
		}
	} else {
		throw new Error(
			`Meta-schema not found at ${metaSchemaPath}. Run 'bun run sync-schemas' first.`,
		);
	}

	// Load and register all SMALL v1 schemas
	const smallV1Path = join(SCHEMAS_ROOT, "small/v1");
	if (existsSync(smallV1Path)) {
		const schemaFiles = readdirSync(smallV1Path).filter((f) =>
			f.endsWith(".schema.json"),
		);

		for (const file of schemaFiles) {
			const schemaPath = join(smallV1Path, file);
			const schema = JSON.parse(readFileSync(schemaPath, "utf-8"));
			if (schema.$id) {
				// Only register if not already available
				if (!ajv.getSchema(schema.$id)) {
					ajv.addSchema(schema);
				}
			}
		}
	} else {
		throw new Error(
			`SMALL v1 schemas not found at ${smallV1Path}. Run 'bun run sync-schemas' first.`,
		);
	}

	globalThis.__smallAjv = ajv;
	return ajv;
}

export function validateWith(
	schemaId: string,
	data: unknown,
): { valid: boolean; errors: ErrorObject[] } {
	const ajv = getAjv();
	const validate = ajv.getSchema(schemaId);

	if (!validate) {
		throw new Error(
			`Schema not found: ${schemaId}. Ensure it's registered in the registry.`,
		);
	}

	const valid = validate(data);
	// validate is synchronous since we don't use loadSchema
	const isValid = typeof valid === "boolean" ? valid : false;
	return {
		valid: isValid,
		errors: validate.errors ?? [],
	};
}

export { getAjv };
