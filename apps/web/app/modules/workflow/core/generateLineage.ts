import type { Lineage } from "~/modules/workflow/types";

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
