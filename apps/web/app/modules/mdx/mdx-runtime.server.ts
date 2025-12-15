import { compile } from "@mdx-js/mdx";
import * as React from "react";
import remarkGfm from "remark-gfm";
import { bundledLanguages, createHighlighter } from "shiki";
import { visit } from "unist-util-visit";
import { CodeBlock } from "./CodeBlock";
import { Mermaid } from "./Mermaid";

// Simple in-memory cache keyed by file path.
// In dev, mtime invalidation keeps it fresh.
// In prod, this is fine unless you hot-swap files at runtime.
type CacheEntry = {
	mtimeMs: number;
	Component: React.FC;
};

const mdxCache = new Map<string, CacheEntry>();

let highlighterPromise: Awaited<ReturnType<typeof createHighlighter>> | null =
	null;
async function getShiki() {
	if (!highlighterPromise) {
		highlighterPromise = await createHighlighter({
			themes: ["github-dark"] as any,
			langs: Object.keys(bundledLanguages) as any,
		});
	}
	return highlighterPromise;
}

function remarkCodeToComponents() {
	return async (tree: any, file: any) => {
		const highlighter = await getShiki();

		const tasks: Promise<void>[] = [];

		visit(tree, "code", (node: any, index, parent: any) => {
			if (!parent || typeof index !== "number") return;

			const lang = (node.lang || "").toString().trim();
			const code = (node.value || "").toString();

			// Mermaid special-case
			if (lang === "mermaid") {
				parent.children[index] = {
					type: "mdxJsxFlowElement",
					name: "Mermaid",
					attributes: [{ type: "mdxJsxAttribute", name: "code", value: code }],
					children: [],
				};
				return;
			}

			// Normal code -> Shiki HTML -> CodeBlock component
			tasks.push(
				(async () => {
					const html = highlighter.codeToHtml(code, {
						lang: lang || "txt",
						theme: "github-dark",
					} as any);

					parent.children[index] = {
						type: "mdxJsxFlowElement",
						name: "CodeBlock",
						attributes: [
							{ type: "mdxJsxAttribute", name: "html", value: html },
							{
								type: "mdxJsxAttribute",
								name: "lang",
								value: lang || "txt",
							},
						],
						children: [],
					};
				})(),
			);
		});

		if (tasks.length) await Promise.all(tasks);
	};
}

// Very small MDX -> React compiler.
// We also override fenced code rendering by converting MDX code blocks to HTML using Shiki.
// Easiest approach: expose a <CodeBlockHtml /> component and let MDX use it via a rehype transform later.
// For Run #2A, we do a pragmatic approach: keep inline code + pre styling, and add a helper export
// so we can use it from MDX content when needed.
export async function compileMdx(args: {
	cacheKey: string;
	mtimeMs: number;
	source: string;
}) {
	const cached = mdxCache.get(args.cacheKey);
	if (cached && cached.mtimeMs === args.mtimeMs) return cached;

	const compiled = await compile(args.source, {
		outputFormat: "function-body",
		development: process.env.NODE_ENV !== "production",
		remarkPlugins: [remarkCodeToComponents, remarkGfm],
	});

	const code = String(compiled);

	// Pass components in as named parameters to avoid redeclaration conflicts.
	// Keep the compiled body clean as specified.
	const wrappedCode = `${code}\nreturn MDXContent;`;

	// In development mode, MDX uses jsx-dev-runtime which exports jsxDEV
	// In production mode, it uses jsx-runtime which exports jsx/jsxs
	const isDevelopment = process.env.NODE_ENV !== "production";
	const runtime = isDevelopment
		? await import("react/jsx-dev-runtime")
		: await import("react/jsx-runtime");

	// eslint-disable-next-line no-new-func
	const fn = isDevelopment
		? new Function("_jsxRuntime", "CodeBlock", "Mermaid", wrappedCode)
		: new Function(
				"React",
				"jsx",
				"jsxs",
				"Fragment",
				"CodeBlock",
				"Mermaid",
				wrappedCode,
			);

	// Ensure we're passing actual functions
	if (typeof CodeBlock !== "function") {
		throw new Error(`CodeBlock must be a function, got: ${typeof CodeBlock}`);
	}
	if (typeof Mermaid !== "function") {
		throw new Error(`Mermaid must be a function, got: ${typeof Mermaid}`);
	}

	const ComponentResult = isDevelopment
		? fn(
				{
					Fragment: runtime.Fragment,
					jsxDEV: runtime.jsxDEV,
				},
				CodeBlock,
				Mermaid,
			)
		: fn(
				React,
				runtime.jsx,
				runtime.jsxs,
				runtime.Fragment,
				CodeBlock,
				Mermaid,
			);

	// Extract the actual component function - MDX may export it as default
	const Component =
		typeof ComponentResult === "function"
			? ComponentResult
			: ((ComponentResult as any)?.default ?? ComponentResult);

	if (typeof Component !== "function") {
		throw new Error(
			`Component must be a function, got: ${typeof Component}. ComponentResult: ${typeof ComponentResult}`,
		);
	}

	const entry: CacheEntry = { mtimeMs: args.mtimeMs, Component };
	mdxCache.set(args.cacheKey, entry);
	return entry;
}

export async function highlightCodeToHtml(args: {
	code: string;
	lang?: string;
}) {
	const highlighter = await getShiki();
	const lang =
		args.lang && highlighter.getLoadedLanguages().includes(args.lang as any)
			? args.lang
			: "txt";
	return highlighter.codeToHtml(args.code, { lang } as any);
}
