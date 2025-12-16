import type { Lineage } from "~/modules/workflow/types";
// import { SMALL_GUARANTEES } from "./guarantees";

// SMALL Guarantees: deterministic, noHiddenState
export function generateLineage(manifest: {
	artifact: string;
	schema: string;
}): Lineage {
	return {
		artifact: manifest.artifact,
		derivedFrom: manifest.schema,
		generatedAt: new Date().toISOString(),
	};
}
