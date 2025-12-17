import type { Lifecycle } from "~/modules/workflow/types";
import { computeReplayId } from "./replayId";
import protocol from "~/modules/protocol/small.protocol.json";

// SMALL Guarantees: deterministic, immutableArtifacts
export function emitLifecycle(
	manifest: {
		artifact: string;
		schema: string;
		version: number;
		metadata?: Record<string, unknown>;
	},
	protocolVersion?: string,
): Lifecycle {
	const replayId = computeReplayId({
		protocolVersion: protocolVersion ?? protocol.version,
		manifest,
	});

	const now = new Date().toISOString();
	return [
		{
			type: "validated",
			at: now,
			replayId,
		},
		{
			type: "materialized",
			at: now,
			replayId,
		},
	];
}
