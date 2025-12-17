import { useEffect, useState } from "react";
import { createHighlighter } from "shiki";

type Props = {
	html?: string;
	code?: string;
	lang?: string;
	title?: string;
};

function labelFrom({ title, lang }: { title?: string; lang?: string }) {
	const raw = (title ?? lang ?? "code").trim();
	return raw.length ? raw : "code";
}

let highlighterPromise: Awaited<ReturnType<typeof createHighlighter>> | null =
	null;

async function getShiki() {
	if (!highlighterPromise) {
		highlighterPromise = createHighlighter({
			themes: ["github-dark"],
			langs: [
				"json",
				"javascript",
				"typescript",
				"jsx",
				"tsx",
				"bash",
				"shell",
				"yaml",
				"markdown",
				"text",
				"plaintext",
			],
		});
	}
	return highlighterPromise;
}

export function CodeBlock({ html, code, lang, title }: Props) {
	const label = labelFrom({ title, lang });
	const [highlightedHtml, setHighlightedHtml] = useState<string | null>(null);

	useEffect(() => {
		if (html) {
			setHighlightedHtml(html);
			return;
		}

		if (!code) {
			setHighlightedHtml(null);
			return;
		}

		let cancelled = false;

		(async () => {
			try {
				const highlighter = await getShiki();
				if (cancelled) return;

				const html = highlighter.codeToHtml(code, {
					lang: lang || "text",
					theme: "github-dark",
				});

				if (!cancelled) {
					setHighlightedHtml(html);
				}
			} catch (error) {
				console.error("Failed to highlight code:", error);
				if (!cancelled) {
					setHighlightedHtml(null);
				}
			}
		})();

		return () => {
			cancelled = true;
		};
	}, [html, code, lang]);

	return (
		<div className="my-8 overflow-hidden rounded-2xl border border-white/10 bg-white/5">
			<div className="flex items-center justify-between border-b border-white/10 px-4 py-2">
				<div className="flex items-center gap-2">
					<span className="text-[11px] font-medium uppercase tracking-wide text-white/60">
						{label}
					</span>
				</div>

				{/* Optional: a subtle "copy" affordance later */}
				<div className="text-[11px] text-white/40"> </div>
			</div>

			{highlightedHtml ? (
				<div
					className="overflow-auto p-4 text-sm leading-6 [--shiki-color-background:transparent] [&_.shiki]:bg-transparent [&_.shiki]:p-0"
					dangerouslySetInnerHTML={{ __html: highlightedHtml }}
				/>
			) : (
				<pre className="overflow-auto p-4 text-sm leading-6">
					<code>{code || ""}</code>
				</pre>
			)}
		</div>
	);
}
