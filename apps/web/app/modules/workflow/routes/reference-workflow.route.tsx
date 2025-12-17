import { useEffect, useState } from "react";
import { CodeBlock } from "~/modules/mdx/CodeBlock";
import { emitLifecycle } from "~/modules/workflow/core/emitLifecycle";
import { generateLineage } from "~/modules/workflow/core/generateLineage";
import { validateManifest } from "~/modules/workflow/core/validateManifest";
import { computeReplayId } from "~/modules/workflow/core/replayId";
import { highlightCodeToHtml } from "~/modules/workflow/lib/highlight";

const EXAMPLE_MANIFEST = `{
  "artifact": "track.audio",
  "schema": "artifact.v1",
  "version": 1
}`;

type ProtocolData = {
	protocol: string;
	version: string;
	primitives: string[];
	rules: Record<string, boolean>;
	schemas: Record<string, string>;
};

export default function ReferenceWorkflowRoute() {
	const [manifestInput, setManifestInput] = useState(EXAMPLE_MANIFEST);
	const [validationResult, setValidationResult] = useState<{
		ok: boolean;
		errors: Array<{ message?: string; instancePath?: string }>;
	} | null>(null);
	const [lineageHtml, setLineageHtml] = useState<string | null>(null);
	const [lifecycleHtml, setLifecycleHtml] = useState<string | null>(null);
	const [replayIdHtml, setReplayIdHtml] = useState<string | null>(null);
	const [isProcessing, setIsProcessing] = useState(false);
	const [protocolData, setProtocolData] = useState<ProtocolData | null>(null);
	const [protocolJsonExpanded, setProtocolJsonExpanded] = useState(false);
	const [protocolJsonHtml, setProtocolJsonHtml] = useState<string | null>(null);

	useEffect(() => {
		async function fetchProtocol() {
			try {
				const response = await fetch("/protocol/small/v1");
				const data: ProtocolData = await response.json();
				setProtocolData(data);
				const jsonStr = JSON.stringify(data, null, 2);
				const html = await highlightCodeToHtml({ code: jsonStr, lang: "json" });
				setProtocolJsonHtml(html);
			} catch (error) {
				console.error("Failed to fetch protocol:", error);
			}
		}
		fetchProtocol();
	}, []);

	const handleValidate = async () => {
		setIsProcessing(true);
		setValidationResult(null);
		setLineageHtml(null);
		setLifecycleHtml(null);
		setReplayIdHtml(null);

		try {
			const parsed = JSON.parse(manifestInput);
			const result = await validateManifest(parsed);

			setValidationResult(result);

			if (result.ok && protocolData) {
				const protocolVersion = protocolData.version;

				// Compute replay ID
				const replayId = await computeReplayId({
					protocolVersion,
					manifest: parsed,
				});
				const replayIdHighlighted = await highlightCodeToHtml({
					code: replayId,
					lang: "text",
				});
				setReplayIdHtml(replayIdHighlighted);

				// Generate lineage and lifecycle with protocol version
				const lineageData = generateLineage(parsed, protocolVersion);
				const lifecycleData = emitLifecycle(parsed, protocolVersion);

				const lineageJson = JSON.stringify(lineageData, null, 2);
				const lifecycleJson = JSON.stringify(lifecycleData, null, 2);

				const [lineageHighlighted, lifecycleHighlighted] = await Promise.all([
					highlightCodeToHtml({ code: lineageJson, lang: "json" }),
					highlightCodeToHtml({ code: lifecycleJson, lang: "json" }),
				]);

				setLineageHtml(lineageHighlighted);
				setLifecycleHtml(lifecycleHighlighted);
			}
		} catch (error) {
			setValidationResult({
				ok: false,
				errors: [
					{
						message:
							error instanceof Error ? error.message : "Invalid JSON format",
					},
				],
			});
		} finally {
			setIsProcessing(false);
		}
	};

	return (
		<div className="space-y-8">
			<div className="space-y-4">
				<div className="flex items-center justify-between">
					<h2 className="text-xl font-semibold">SMALL Playground</h2>
					{protocolData && (
						<div className="flex items-center gap-2 rounded-lg border border-white/10 bg-black/40 px-3 py-1.5 text-sm">
							<span className="text-white/60">Protocol:</span>
							<span className="font-mono font-semibold text-white">
								{protocolData.protocol} v{protocolData.version}
							</span>
						</div>
					)}
				</div>

				<div className="rounded-lg border border-blue-500/20 bg-blue-500/10 p-4 text-sm text-blue-200">
					<p className="font-medium">Deterministic Replay</p>
					<p className="mt-1 text-blue-300/80">
						This playground is stateless and deterministic. No data is
						persisted. All outputs are computed from inputs using SMALL
						primitives.
					</p>
				</div>

				{protocolData && protocolJsonHtml && (
					<div className="space-y-2">
						<button
							onClick={() => setProtocolJsonExpanded(!protocolJsonExpanded)}
							className="flex w-full items-center justify-between rounded-lg border border-white/10 bg-black/40 p-3 text-left text-sm hover:bg-black/60"
						>
							<span className="font-medium">Raw Protocol JSON</span>
							<span className="text-white/60">
								{protocolJsonExpanded ? "▼" : "▶"}
							</span>
						</button>
						{protocolJsonExpanded && (
							<CodeBlock html={protocolJsonHtml} lang="json" title="Protocol" />
						)}
					</div>
				)}

				<h2 className="text-xl font-semibold">Manifest Input</h2>
				<textarea
					value={manifestInput}
					onChange={(e) => setManifestInput(e.target.value)}
					className="w-full rounded-lg border border-white/10 bg-black/40 p-4 font-mono text-sm text-white focus:border-white/20 focus:outline-none"
					rows={8}
					placeholder="Paste your manifest JSON here..."
				/>
				<button
					onClick={handleValidate}
					disabled={isProcessing}
					className="rounded-lg bg-white px-4 py-2 text-sm font-semibold text-zinc-950 disabled:opacity-50 disabled:cursor-not-allowed hover:bg-white/90"
				>
					{isProcessing ? "Processing..." : "Validate"}
				</button>
			</div>

			{validationResult && (
				<div className="space-y-6">
					<div>
						<h2 className="mb-4 text-xl font-semibold">Validation Result</h2>
						{validationResult.ok ? (
							<div className="flex items-center gap-2 text-green-400">
								<span className="text-lg">✅</span>
								<span className="font-medium">Valid</span>
							</div>
						) : (
							<div className="space-y-2">
								<div className="flex items-center gap-2 text-red-400">
									<span className="text-lg">❌</span>
									<span className="font-medium">Invalid</span>
								</div>
								{validationResult.errors.length > 0 && (
									<ul className="ml-6 list-disc space-y-1 text-sm text-red-300">
										{validationResult.errors.map((error, idx) => (
											<li key={idx}>
												{error.instancePath && (
													<span className="font-mono">
														{error.instancePath}
													</span>
												)}{" "}
												{error.message}
											</li>
										))}
									</ul>
								)}
							</div>
						)}
					</div>

					{validationResult.ok && replayIdHtml && (
						<div>
							<h2 className="mb-4 text-xl font-semibold">Replay ID</h2>
							<CodeBlock html={replayIdHtml} lang="text" title="Replay ID" />
						</div>
					)}

					{validationResult.ok && lineageHtml && lifecycleHtml && (
						<>
							<div>
								<h2 className="mb-4 text-xl font-semibold">Lineage</h2>
								<CodeBlock html={lineageHtml} lang="json" title="Lineage" />
							</div>

							<div>
								<h2 className="mb-4 text-xl font-semibold">Lifecycle</h2>
								<CodeBlock html={lifecycleHtml} lang="json" title="Lifecycle" />
							</div>
						</>
					)}
				</div>
			)}
		</div>
	);
}
