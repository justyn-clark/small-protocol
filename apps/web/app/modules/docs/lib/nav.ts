export type DocItem = { slug: string; title: string; description?: string };

export type DocSection = { id: string; title: string; items: DocItem[] };

export const DOCS_NAV: DocSection[] = [
	{
		id: "getting-started",
		title: "Getting started",
		items: [
			{ slug: "docs", title: "Overview" },
			{ slug: "small-v1", title: "SMALL v1" },
			{ slug: "reference-workflow", title: "Reference workflow" },
		],
	},
	{
		id: "primitives",
		title: "Primitives",
		items: [
			{ slug: "primitives-v1", title: "Primitive spec v1" },
			{ slug: "glossary", title: "Glossary" },
			{ slug: "faq", title: "FAQ" },
		],
	},
	{
		id: "architecture",
		title: "Architecture",
		items: [
			{ slug: "diagrams", title: "Canonical diagrams" },
			{ slug: "small-model", title: "SMALL Model" },
		],
	},
	{
		id: "contracts",
		title: "Contracts",
		items: [
			{ slug: "api", title: "OpenAPI surface" },
			{ slug: "schemas", title: "JSON Schemas" },
		],
	},
	{
		id: "manifesto",
		title: "Manifesto",
		items: [{ slug: "small-is-the-new-cms", title: "SMALL Is the New CMS" }],
	},
];
