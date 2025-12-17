import type { Lineage } from "~/modules/workflow/types";
import { computeReplayId } from "./replayId";
import protocol from "~/modules/protocol/small.protocol.json";

// SMALL Guarantees: deterministic, noHiddenState
export function generateLineage(
	manifest: {
		artifact: string;
		schema: string;
		version: number;
		metadata?: Record<string, unknown>;
	},
	protocolVersion?: string,
): Lineage {
	const replayId = computeReplayId({
		protocolVersion: protocolVersion ?? protocol.version,
		manifest,
	});

	return {
		artifact: manifest.artifact,
		derivedFrom: manifest.schema,
		generatedAt: new Date().toISOString(),
		replayId,
	};
}
