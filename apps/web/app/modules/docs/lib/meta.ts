import { DOCS_NAV } from "./nav";

export function getDocMeta(slug: string) {
	if (slug === "docs")
		return {
			title: "Overview",
			description: "The spec and reference workflow.",
		};

	for (const s of DOCS_NAV) {
		for (const i of s.items) {
			if (i.slug === slug) {
				return { title: i.title, description: i.description };
			}
		}
	}

	return { title: "Docs" };
}
