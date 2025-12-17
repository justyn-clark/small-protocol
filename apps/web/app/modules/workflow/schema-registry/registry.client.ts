import Ajv2020, { type ErrorObject } from "ajv/dist/2020";
import addFormats from "ajv-formats";

let ajvInstance: Ajv2020 | null = null;
let schemasLoaded = false;

const SCHEMA_FILES = [
	"/schemas/meta/draft2020-12.schema.json",
	"/schemas/small/v1/artifact.schema.json",
	"/schemas/small/v1/lifecycle.schema.json",
	"/schemas/small/v1/lineage.schema.json",
	"/schemas/small/v1/manifest.schema.json",
	"/schemas/small/v1/schema.schema.json",
];

async function loadSchemas(): Promise<void> {
	if (schemasLoaded) {
		return;
	}

	if (!ajvInstance) {
		ajvInstance = new Ajv2020({
			strict: true,
			allErrors: true,
		});
		addFormats(ajvInstance);
	}

	const schemaPromises = SCHEMA_FILES.map(async (path) => {
		const response = await fetch(path);
		if (!response.ok) {
			throw new Error(
				`Failed to load schema at ${path}: ${response.statusText}`,
			);
		}
		const schema = await response.json();
		if (schema.$id) {
			ajvInstance!.addSchema(schema, schema.$id);
		}
	});

	await Promise.all(schemaPromises);
	schemasLoaded = true;
}

async function getAjv(): Promise<Ajv2020> {
	await loadSchemas();
	if (!ajvInstance) {
		throw new Error("AJV instance not initialized");
	}
	return ajvInstance;
}

export async function validateWith(
	schemaId: string,
	data: unknown,
): Promise<{ valid: boolean; errors: ErrorObject[] }> {
	const ajv = await getAjv();
	const validate = ajv.getSchema(schemaId);

	if (!validate) {
		throw new Error(
			`Schema not found: ${schemaId}. Ensure it's registered in the registry.`,
		);
	}

	const valid = validate(data);
	const isValid = typeof valid === "boolean" ? valid : false;
	return {
		valid: isValid,
		errors: validate.errors ?? [],
	};
}

export { getAjv };
