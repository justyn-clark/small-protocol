import { compile } from "@mdx-js/mdx";
import * as React from "react";
import remarkGfm from "remark-gfm";
import { visit } from "unist-util-visit";
import { CodeBlock } from "./CodeBlock";
import { Mermaid } from "./Mermaid";

function parseFrontmatter(source: string): {
	frontmatter: Record<string, unknown>;
	content: string;
} {
	const frontmatterRegex = /^---\s*\n([\s\S]*?)\n---\s*\n([\s\S]*)$/;
	const match = source.match(frontmatterRegex);

	if (!match) {
		return { frontmatter: {}, content: source };
	}

	const [, frontmatterYaml, content] = match;
	const frontmatter: Record<string, unknown> = {};

	for (const line of frontmatterYaml.split("\n")) {
		const trimmed = line.trim();
		if (!trimmed || trimmed.startsWith("#")) continue;

		const colonIndex = trimmed.indexOf(":");
		if (colonIndex === -1) continue;

		const key = trimmed.slice(0, colonIndex).trim();
		let value: string | number | boolean = trimmed.slice(colonIndex + 1).trim();

		if (
			(value.startsWith('"') && value.endsWith('"')) ||
			(value.startsWith("'") && value.endsWith("'"))
		) {
			value = value.slice(1, -1);
		}

		if (value === "true") value = true;
		else if (value === "false") value = false;
		else if (/^\d+$/.test(value)) value = Number(value);
		else if (/^\d+\.\d+$/.test(value)) value = Number(value);

		frontmatter[key] = value;
	}

	return { frontmatter, content };
}

function remarkCodeToComponents() {
	return (tree: any) => {
		visit(tree, "code", (node: any, index, parent: any) => {
			if (!parent || typeof index !== "number") return;

			const lang = (node.lang || "").toString().trim().toLowerCase();
			const code = (node.value || "").toString();

			if (lang === "mermaid") {
				parent.children[index] = {
					type: "mdxJsxFlowElement",
					name: "Mermaid",
					attributes: [{ type: "mdxJsxAttribute", name: "code", value: code }],
					children: [],
				};
				return;
			}

			// For client-side, we'll render code blocks without syntax highlighting
			// The CodeBlock component can handle plain text via the code prop
			parent.children[index] = {
				type: "mdxJsxFlowElement",
				name: "CodeBlock",
				attributes: [
					{ type: "mdxJsxAttribute", name: "code", value: code },
					{
						type: "mdxJsxAttribute",
						name: "lang",
						value: lang || "txt",
					},
				],
				children: [],
			};
		});
	};
}

export async function compileMdxClient(source: string): Promise<{
	Component: React.FC;
	frontmatter: Record<string, unknown>;
}> {
	const { frontmatter, content } = parseFrontmatter(source);

	const compiled = await compile(content, {
		outputFormat: "function-body",
		development: process.env.NODE_ENV !== "production",
		remarkPlugins: [remarkCodeToComponents, remarkGfm],
	});

	const code = String(compiled);

	const wrappedCode = `
		const React = arguments[1];
		const CodeBlock = arguments[2];
		const Mermaid = arguments[3];
		${code}
	`;

	const isDevelopment = process.env.NODE_ENV !== "production";
	const runtime = isDevelopment
		? await import("react/jsx-dev-runtime")
		: await import("react/jsx-runtime");

	const fn = new Function(wrappedCode);

	const ComponentResult = fn(runtime, React, CodeBlock, Mermaid);

	const Component =
		typeof ComponentResult === "function"
			? ComponentResult
			: ((ComponentResult as any)?.default ?? ComponentResult);

	if (typeof Component !== "function") {
		throw new Error(
			`Component must be a function, got: ${typeof Component}. ComponentResult: ${typeof ComponentResult}`,
		);
	}

	return {
		Component,
		frontmatter,
	};
}
