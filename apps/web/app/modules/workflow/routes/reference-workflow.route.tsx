import { useState } from "react";
import { CodeBlock } from "~/modules/mdx/CodeBlock";
import { emitLifecycle } from "~/modules/workflow/core/emitLifecycle";
import { generateLineage } from "~/modules/workflow/core/generateLineage";
import { validateManifest } from "~/modules/workflow/core/validateManifest";
import { highlightCodeToHtml } from "~/modules/workflow/lib/highlight";

const EXAMPLE_MANIFEST = `{
  "artifact": "track.audio",
  "schema": "artifact.v1",
  "version": 1
}`;

export default function ReferenceWorkflowRoute() {
	const [manifestInput, setManifestInput] = useState(EXAMPLE_MANIFEST);
	const [validationResult, setValidationResult] = useState<{
		ok: boolean;
		errors: Array<{ message?: string; instancePath?: string }>;
	} | null>(null);
	const [lineageHtml, setLineageHtml] = useState<string | null>(null);
	const [lifecycleHtml, setLifecycleHtml] = useState<string | null>(null);
	const [isProcessing, setIsProcessing] = useState(false);

	const handleValidate = async () => {
		setIsProcessing(true);
		setValidationResult(null);
		setLineageHtml(null);
		setLifecycleHtml(null);

		try {
			const parsed = JSON.parse(manifestInput);
			const result = validateManifest(parsed);

			setValidationResult(result);

			if (result.ok) {
				const lineageData = generateLineage(parsed);
				const lifecycleData = emitLifecycle(parsed);

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
