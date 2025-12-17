import { useEffect, useId, useMemo, useRef, useState } from "react";

type Props = {
	code: string;
	title?: string;
};

type MermaidModule = any;
type MermaidRenderResult =
	| string
	| { svg?: string; bindFunctions?: (el: Element) => void };

let mermaidInit = false;

function safeId(raw: string) {
	// React 18/19 useId can include ":" which can confuse Mermaid / selectors.
	return raw.replace(/[:]/g, "-");
}

function resolveMermaid(mod: MermaidModule) {
	// Mermaid packages/export shapes vary. This is the least-bad “works in practice” chain.
	return mod?.default ?? mod?.mermaid ?? mod;
}

function extractSvg(rendered: MermaidRenderResult): {
	svg: string;
	bind?: (el: Element) => void;
} {
	if (typeof rendered === "string") return { svg: rendered };
	return { svg: rendered?.svg ?? "", bind: rendered?.bindFunctions };
}

function hardenSvg(svgEl: SVGElement) {
	// Ensure it actually displays and is responsive inside a constrained container.
	svgEl.style.display = "block";
	svgEl.style.maxWidth = "100%";
	svgEl.style.width = "100%";
	svgEl.style.height = "auto";

	// Mermaid sometimes sets explicit width/height; these can fight responsiveness.
	svgEl.removeAttribute("width");
	svgEl.removeAttribute("height");

	// Keep aspect ratio reasonable.
	svgEl.setAttribute("preserveAspectRatio", "xMinYMin meet");
}

export function Mermaid({ code, title }: Props) {
	const reactId = useId();
	const id = useMemo(() => `m-${safeId(reactId)}`, [reactId]);

	const containerRef = useRef<HTMLDivElement | null>(null);
	const [error, setError] = useState<string | null>(null);

	const normalized = useMemo(() => code.trim(), [code]);

	useEffect(() => {
		let cancelled = false;

		async function run() {
			try {
				const mod = await import("mermaid");
				const mermaid = resolveMermaid(mod);

				if (
					!mermaid ||
					typeof mermaid.initialize !== "function" ||
					typeof mermaid.render !== "function"
				) {
					throw new Error(
						"Mermaid module did not resolve to a usable API (initialize/render missing).",
					);
				}

				if (!mermaidInit) {
					mermaid.initialize({
						startOnLoad: false,
						theme: "dark",
						// strict can block SVG/labels in some cases; this is docs rendering, not a bank vault.
						securityLevel: "loose",
						fontFamily: "ui-sans-serif, system-ui, -apple-system",
						sequence: { showSequenceNumbers: false },
						flowchart: { useMaxWidth: true, htmlLabels: true },
					});
					mermaidInit = true;
				}

				const container = containerRef.current;
				if (!container) return;

				// Clear prior render (fast refresh / rerenders)
				container.innerHTML = "";

				// Optional validation (gives better errors)
				if (typeof mermaid.parse === "function") {
					await mermaid.parse(normalized);
				}

				const rendered: MermaidRenderResult = await mermaid.render(
					id,
					normalized,
				);
				const { svg, bind } = extractSvg(rendered);

				if (!svg || !svg.includes("<svg")) {
					throw new Error("Mermaid render returned no SVG.");
				}

				if (cancelled) return;

				container.innerHTML = svg;

				// Mermaid v11: wire up interactions/events
				if (typeof bind === "function") {
					bind(container);
				}

				const svgEl = container.querySelector("svg") as SVGElement | null;
				if (svgEl) hardenSvg(svgEl);

				setError(null);
			} catch (e: any) {
				if (cancelled) return;
				setError(
					typeof e === "string"
						? e
						: e?.message
							? String(e.message)
							: "Mermaid render failed",
				);
			}
		}

		run();
		return () => {
			cancelled = true;
		};
	}, [id, normalized]);

	return (
		<div className="not-prose my-8 overflow-hidden rounded-2xl border border-white/10 bg-white/5">
			<div className="flex items-center justify-between border-b border-white/10 px-4 py-2">
				<span className="text-[11px] font-medium uppercase tracking-wide text-white/60">
					{title ?? "Diagram"}
				</span>
			</div>

			<div className="px-4 py-4">
				<div
					ref={containerRef}
					className="mx-auto max-w-3xl overflow-auto [&_svg]:max-w-full [&_svg]:h-auto [&_svg]:w-full"
				/>

				{error ? (
					<div className="mt-4 rounded-xl border border-red-500/30 bg-red-500/10 p-3 text-sm text-white/80">
						<div className="text-xs font-semibold uppercase tracking-wide text-red-200/90">
							Mermaid error
						</div>
						<div className="mt-1 text-sm text-white/80">{error}</div>
					</div>
				) : null}

				<details className="mt-4">
					<summary className="cursor-pointer text-xs text-white/60 hover:text-white/80">
						View diagram source
					</summary>
					<pre className="mt-3 overflow-auto rounded-xl border border-white/10 bg-black/40 p-4 text-xs leading-5 text-white/80">
						<code>{normalized}</code>
					</pre>
				</details>
			</div>
		</div>
	);
}
