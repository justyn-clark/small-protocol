import type { Lifecycle } from "~/modules/workflow/types";

export function emitLifecycle(manifest: {
	artifact: string;
	schema: string;
}): Lifecycle {
	const now = new Date().toISOString();
	return [
		{
			type: "validated",
			at: now,
		},
		{
			type: "materialized",
			at: now,
		},
	];
}
