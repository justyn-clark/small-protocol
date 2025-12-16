import Ajv2020, { type ErrorObject } from "ajv/dist/2020";
import addFormats from "ajv-formats";
import manifestSchema from "../schemas/manifest.schema.json";

// SMALL Guarantees: explicitContractsOnly, deterministic
const ajv = new Ajv2020({
	strict: true,
	allErrors: true,
});
addFormats(ajv);

export function validateManifest(manifest: unknown): {
	ok: boolean;
	errors: ErrorObject[];
} {
	const validate = ajv.compile(manifestSchema);
	const ok = validate(manifest);
	return {
		ok,
		errors: validate.errors ?? [],
	};
}
