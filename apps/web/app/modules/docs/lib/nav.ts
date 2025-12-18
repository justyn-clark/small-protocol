export type DocItem = { slug: string; title: string; description?: string };

export type DocSection = { id: string; title: string; items: DocItem[] };

export const DOCS_NAV: DocSection[] = [
	{
		id: "overview",
		title: "Overview",
		items: [
			{ slug: "docs", title: "Specification Overview" },
			{ slug: "small-v1", title: "Protocol Version v1" },
			{ slug: "reference-workflow", title: "Reference Workflow" },
		],
	},
	{
		id: "primitives",
		title: "Primitives",
		items: [
			{ slug: "primitives-v1", title: "Primitive Specification v1" },
			{ slug: "glossary", title: "Glossary" },
			{ slug: "faq", title: "FAQ" },
		],
	},
	{
		id: "architecture",
		title: "Architecture",
		items: [
			{ slug: "diagrams", title: "Canonical Diagrams" },
			{ slug: "small-model", title: "Execution Model" },
		],
	},
	{
		id: "contracts",
		title: "Contracts",
		items: [
			{ slug: "api", title: "OpenAPI Surface" },
			{ slug: "schemas", title: "JSON Schemas" },
		],
	},
	{
		id: "context",
		title: "Context",
		items: [
			{ slug: "small-is-the-new-cms", title: "SMALL Is the New CMS" },
		],
	},
];