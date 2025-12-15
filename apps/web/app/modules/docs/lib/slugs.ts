import path from "node:path";
import { fileURLToPath } from "node:url";

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

const ROOT = path.join(__dirname, "..", "content");

export function slugToPath(slug: string) {
	if (slug === "docs") return path.join(ROOT, "docs.mdx");
	if (slug === "primitives-v1") return path.join(ROOT, "primitives", "v1.mdx");
	if (slug === "glossary") return path.join(ROOT, "primitives", "glossary.mdx");
	if (slug === "faq") return path.join(ROOT, "primitives", "faq.mdx");
	if (slug === "diagrams") return path.join(ROOT, "diagrams.mdx");
	if (slug === "reference-workflow")
		return path.join(ROOT, "reference-workflow.mdx");
	if (slug === "api") return path.join(ROOT, "api.mdx");
	if (slug === "schemas") return path.join(ROOT, "schemas.mdx");

	return null;
}
