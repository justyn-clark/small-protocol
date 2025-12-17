import protocol from "~/modules/protocol/small.protocol.json";
import { canonicalJson, sha256 } from "../lib/hash";

export const SUPPORTED_PROTOCOL_VERSIONS = ["1.0.0"];

export interface ComputeReplayIdParams {
	protocolVersion?: string;
	manifest: {
		artifact: string;
		schema: string;
		version: number;
		metadata?: Record<string, unknown>;
	};
}

/**
 * Compute deterministic replay ID from protocol version and manifest.
 *
 * Hash format: sha256("SMALL|" + protocolVersion + "|" + canonicalJson(manifest))
 *
 * This ensures:
 * - Same manifest + same protocolVersion → same replayId
 * - Different protocolVersion → different replayId
 * - Different manifest → different replayId
 *
 * @param params - Object containing protocolVersion (optional, defaults to protocol version) and manifest
 * @returns Base64url-encoded SHA256 hash
 */
export async function computeReplayId({
	protocolVersion,
	manifest,
}: ComputeReplayIdParams): Promise<string> {
	// Default to protocol version if not provided
	const version = protocolVersion ?? protocol.version;

	// Validate protocol version
	if (!SUPPORTED_PROTOCOL_VERSIONS.includes(version)) {
		throw new Error(
			`Unsupported protocol version: ${version}. Supported versions: ${SUPPORTED_PROTOCOL_VERSIONS.join(", ")}`,
		);
	}

	// Compute canonical JSON representation of manifest
	const manifestJson = canonicalJson(manifest);

	// Hash format: "SMALL|" + protocolVersion + "|" + canonicalJson(manifest)
	const hashInput = `SMALL|${version}|${manifestJson}`;

	return sha256(hashInput);
}
