import Ajv, { type ErrorObject } from "ajv";
import addFormats from "ajv-formats";
import manifestSchema from "../schemas/manifest.schema.json";

const ajv = new Ajv({ allErrors: true });
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
