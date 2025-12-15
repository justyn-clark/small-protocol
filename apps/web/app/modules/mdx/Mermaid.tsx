import { useEffect, useId, useMemo, useRef, useState } from "react";

type Props = {
	code: string;
	title?: string;
};

let mermaidInit = false;

export function Mermaid({ code, title }: Props) {
	const id = useId();
	const containerRef = useRef<HTMLDivElement | null>(null);
	const [error, setError] = useState<string | null>(null);

	const normalized = useMemo(() => code.trim(), [code]);

	useEffect(() => {
		let cancelled = false;

		async function run() {
			try {
				// @ts-expect-error - mermaid types will be available after bun install
				// eslint-disable-next-line @typescript-eslint/no-unsafe-assignment, @typescript-eslint/no-unsafe-member-access
				const mermaidModule = await import("mermaid");
				// eslint-disable-next-line @typescript-eslint/no-unsafe-assignment, @typescript-eslint/no-unsafe-member-access
				const mermaid = mermaidModule.default as {
					initialize: (config: {
						startOnLoad: boolean;
						theme: string;
						securityLevel: string;
						fontFamily: string;
					}) => void;
					render: (id: string, definition: string) => Promise<{ svg: string }>;
				};

				if (!mermaidInit) {
					mermaid.initialize({
						startOnLoad: false,
						theme: "dark",
						securityLevel: "strict",
						fontFamily: "ui-sans-serif, system-ui, -apple-system",
					});
					mermaidInit = true;
				}

				const { svg } = await mermaid.render(`mermaid-${id}`, normalized);

				if (cancelled) return;
				if (containerRef.current) {
					containerRef.current.innerHTML = svg;
				}
				setError(null);
			} catch (e: any) {
				if (cancelled) return;
				setError(e?.message ?? "Mermaid render failed");
			}
		}

		run();
		return () => {
			cancelled = true;
		};
	}, [id, normalized]);

	return (
		<div className="my-6 overflow-hidden rounded-xl border border-white/10 bg-white/5">
			<div className="flex items-center justify-between border-b border-white/10 px-4 py-2">
				<div className="text-xs font-medium text-white/70">
					{title ?? "Diagram"}
				</div>
			</div>

			<div className="p-4">
				{/* Client-render target */}
				<div ref={containerRef} />

				{/* SSR + error fallback */}
				{error ? (
					<div className="mt-4 rounded-lg border border-red-500/30 bg-red-500/10 p-3 text-sm text-white/80">
						<div className="font-semibold">Mermaid error</div>
						<div className="mt-1 opacity-90">{error}</div>
					</div>
				) : null}

				{/* Keep source available (nice for debugging + "spec credibility") */}
				<details className="mt-4">
					<summary className="cursor-pointer text-xs text-white/60 hover:text-white/80">
						View diagram source
					</summary>
					<pre className="mt-2 overflow-auto rounded-lg border border-white/10 bg-black/40 p-3 text-xs text-white/80">
						<code>{normalized}</code>
					</pre>
				</details>
			</div>
		</div>
	);
}
