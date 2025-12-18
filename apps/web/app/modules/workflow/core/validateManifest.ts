import type { ErrorObject } from "ajv/dist/2020";
import { validateWith } from "../schema-registry/registry.client";

// SMALL Guarantees: explicitContractsOnly, deterministic
const MANIFEST_SCHEMA_ID =
	"http://localhost:5173/schemas/small/v1/manifest.schema.json";

export async function validateManifest(manifest: unknown): Promise<{
	ok: boolean;
	errors: ErrorObject[];
}> {
	const result = await validateWith(MANIFEST_SCHEMA_ID, manifest);
	return {
		ok: result.valid,
		errors: result.errors,
	};
}
