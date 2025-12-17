import { statSync } from "node:fs";
import fs from "node:fs/promises";
import { compileMdx } from "~/modules/mdx/mdx-runtime.server";
import { slugToPath } from "./slugs";
import { validateProtocolDoc } from "./validate-protocol-docs";

export async function loadDoc(slug: string) {
	const file = slugToPath(slug);

	if (!file) {
		const fallback = `# Missing doc\n\nNo MDX file mapped for slug: ${slug}\n`;
		const compiled = await compileMdx({
			cacheKey: `missing:${slug}`,
			mtimeMs: 0,
			source: fallback,
		});
		return { slug, source: fallback, frontmatter: compiled.frontmatter ?? {} };
	}

	const source = await fs.readFile(file, "utf8");
	const mtimeMs = statSync(file).mtimeMs;

	const compiled = await compileMdx({
		cacheKey: file,
		mtimeMs,
		source,
	});

	// Validate protocol frontmatter at build time
	validateProtocolDoc(compiled.frontmatter ?? {}, file);

	// Return the source content so it can be compiled on the client
	// Functions can't be serialized in React Router loader data
	// This allows React hooks (like useEffect in Mermaid) to run properly
	return { slug, source, frontmatter: compiled.frontmatter ?? {} };
}
