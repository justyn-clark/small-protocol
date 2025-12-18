import { compile } from "@mdx-js/mdx";
import * as React from "react";
import remarkGfm from "remark-gfm";
import { bundledLanguages, createHighlighter } from "shiki";
import { visit } from "unist-util-visit";
import { CodeBlock } from "./CodeBlock";
import { Mermaid } from "./Mermaid";

type CacheEntry = {
	mtimeMs: number;
	Component: React.FC;
	frontmatter?: Record<string, unknown>;
};

export type Frontmatter = Record<string, unknown>;

const mdxCache = new Map<string, CacheEntry>();

/**
 * Frontmatter: intentionally tiny parser.
 * Supports: key: value, booleans, numbers, quoted strings.
 * Not a full YAML parser (by design).
 */
function parseFrontmatter(source: string): {
	frontmatter: Frontmatter;
	content: string;
} {
	const frontmatterRegex = /^---\s*\n([\s\S]*?)\n---\s*\n?([\s\S]*)$/;
	const match = source.match(frontmatterRegex);

	if (!match) return { frontmatter: {}, content: source };

	const [, frontmatterYaml, content] = match;
	const frontmatter: Frontmatter = {};

	for (const line of frontmatterYaml.split("\n")) {
		const trimmed = line.trim();
		if (!trimmed || trimmed.startsWith("#")) continue;

		const colonIndex = trimmed.indexOf(":");
		if (colonIndex === -1) continue;

		const key = trimmed.slice(0, colonIndex).trim();
		let raw = trimmed.slice(colonIndex + 1).trim();

		// Empty values allowed (treat as empty string)
		if (!raw) {
			frontmatter[key] = "";
			continue;
		}

		// Remove surrounding quotes
		if (
			(raw.startsWith('"') && raw.endsWith('"')) ||
			(raw.startsWith("'") && raw.endsWith("'"))
		) {
			raw = raw.slice(1, -1);
			frontmatter[key] = raw;
			continue;
		}

		// Parse booleans
		if (raw === "true") {
			frontmatter[key] = true;
			continue;
		}
		if (raw === "false") {
			frontmatter[key] = false;
			continue;
		}

		// Parse numbers
		if (/^-?\d+$/.test(raw)) {
			frontmatter[key] = Number(raw);
			continue;
		}
		if (/^-?\d+\.\d+$/.test(raw)) {
			frontmatter[key] = Number(raw);
			continue;
		}

		frontmatter[key] = raw;
	}

	return { frontmatter, content };
}

// ---- Shiki (singleton) ----

type ShikiHighlighter = Awaited<ReturnType<typeof createHighlighter>>;

let shiki: ShikiHighlighter | null = null;
async function getShiki(): Promise<ShikiHighlighter> {
	if (shiki) return shiki;

	shiki = await createHighlighter({
		themes: ["github-dark"] as any,
		langs: Object.keys(bundledLanguages) as any,
	});

	return shiki;
}

function normalizeLang(lang: unknown): string {
	return (lang ?? "").toString().trim().toLowerCase();
}

function stripBackgroundStyles(html: string): string {
	return html.replace(/style="([^"]*)"/g, (_, styles) => {
		const cleaned = styles
			.split(";")
			.filter((s: string) => {
				const prop = s.split(":")[0].trim().toLowerCase();
				return prop !== "background" && prop !== "background-color";
			})
			.join(";");
		return cleaned ? `style="${cleaned}"` : "";
	});
}

async function codeToHtmlSafe(
	highlighter: ShikiHighlighter,
	code: string,
	lang: string,
) {
	const loaded = new Set(
		highlighter.getLoadedLanguages().map((l) => String(l).toLowerCase()),
	);

	const safeLang = loaded.has(lang) ? lang : "txt";

	const html = highlighter.codeToHtml(code, {
		lang: safeLang,
		theme: "github-dark",
	} as any);

	return stripBackgroundStyles(html);
}

// ---- Remark plugin: transform fenced code blocks -> components ----

function remarkCodeToComponents() {
	return async (tree: any) => {
		const highlighter = await getShiki();
		const tasks: Promise<void>[] = [];

		visit(tree, "code", (node: any, index: number | undefined, parent: any) => {
			if (!parent || typeof index !== "number") return;

			const lang = normalizeLang(node.lang);
			const code = (node.value ?? "").toString();

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

			// Normal code -> Shiki -> CodeBlock
			tasks.push(
				(async () => {
					const html = await codeToHtmlSafe(highlighter, code, lang || "txt");

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

// ---- MDX compiler ----

export async function compileMdx(args: {
	cacheKey: string;
	mtimeMs: number;
	source: string;
}) {
	const cached = mdxCache.get(args.cacheKey);
	if (cached && cached.mtimeMs === args.mtimeMs) {
		return {
			Component: cached.Component,
			frontmatter: cached.frontmatter ?? {},
		};
	}

	const { frontmatter, content } = parseFrontmatter(args.source);

	const compiled = await compile(content, {
		outputFormat: "function-body",
		development: process.env.NODE_ENV !== "production",
		remarkPlugins: [remarkCodeToComponents, remarkGfm],
	});

	const fnBody = String(compiled);

	// runtime is arguments[0] (MDX expects this)
	// React is arguments[1] (needed by some MDX output)
	// components are arguments[2] (we inject CodeBlock + Mermaid)
	const wrappedCode = `
		const React = arguments[1];
		const { CodeBlock, Mermaid } = arguments[2] || {};
		${fnBody}
	`;

	const isDev = process.env.NODE_ENV !== "production";
	const runtime = isDev
		? await import("react/jsx-dev-runtime")
		: await import("react/jsx-runtime");

	// eslint-disable-next-line no-new-func
	const fn = new Function(wrappedCode);

	// Provide components as a single object to avoid “Identifier already declared” issues.
	const result = fn(runtime, React, { CodeBlock, Mermaid });

	const Component =
		typeof result === "function"
			? result
			: ((result as any)?.default ?? result);

	if (typeof Component !== "function") {
		throw new Error(
			`MDX compile produced non-component. Got: ${typeof Component}`,
		);
	}

	mdxCache.set(args.cacheKey, {
		mtimeMs: args.mtimeMs,
		Component,
		frontmatter,
	});

	return { Component, frontmatter };
}

// ---- Public helper (used elsewhere) ----

export async function highlightCodeToHtml(args: {
	code: string;
	lang?: string;
}) {
	const highlighter = await getShiki();
	const lang = normalizeLang(args.lang) || "txt";
	return codeToHtmlSafe(highlighter, args.code, lang);
}
