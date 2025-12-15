import { bundledLanguages, createHighlighter } from "shiki";

let highlighterPromise: Awaited<ReturnType<typeof createHighlighter>> | null =
	null;

async function getHighlighter() {
	if (!highlighterPromise) {
		highlighterPromise = await createHighlighter({
			themes: ["github-dark"] as any,
			langs: ["json", "javascript", "typescript"] as any,
		});
	}
	return highlighterPromise;
}

export async function highlightCodeToHtml(args: {
	code: string;
	lang?: string;
}): Promise<string> {
	const highlighter = await getHighlighter();
	const lang = args.lang || "json";
	return highlighter.codeToHtml(args.code, {
		lang: lang as any,
		theme: "github-dark",
	} as any);
}
