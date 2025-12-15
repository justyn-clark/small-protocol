import { statSync } from "node:fs";
import fs from "node:fs/promises";
import * as React from "react";
import { renderToString } from "react-dom/server";
import { CodeBlock } from "~/modules/mdx/CodeBlock";
import { MDXRenderer } from "~/modules/mdx/MDXRenderer";
import { Mermaid } from "~/modules/mdx/Mermaid";
import { compileMdx } from "~/modules/mdx/mdx-runtime.server";
import { slugToPath } from "./slugs";

export async function loadDoc(slug: string) {
	const file = slugToPath(slug);

	if (!file) {
		const fallback = `# Missing doc\n\nNo MDX file mapped for slug: ${slug}\n`;
		const compiled = await compileMdx({
			cacheKey: `missing:${slug}`,
			mtimeMs: 0,
			source: fallback,
		});
		const html = renderToString(
			React.createElement(MDXRenderer, {
				children: React.createElement(
					compiled.Component as React.ComponentType<{
						components?: {
							CodeBlock: typeof CodeBlock;
							Mermaid: typeof Mermaid;
						};
					}>,
					{
						components: {
							CodeBlock,
							Mermaid,
						},
					},
				),
			}),
		);
		return { slug, html };
	}

	const source = await fs.readFile(file, "utf8");
	const mtimeMs = statSync(file).mtimeMs;

	const compiled = await compileMdx({
		cacheKey: file,
		mtimeMs,
		source,
	});

	const html = renderToString(
		React.createElement(MDXRenderer, {
			children: React.createElement(
				compiled.Component as React.ComponentType<{
					components?: { CodeBlock: typeof CodeBlock; Mermaid: typeof Mermaid };
				}>,
				{
					components: {
						CodeBlock,
						Mermaid,
					},
				},
			),
		}),
	);

	return { slug, html };
}
