import { index, route } from "@react-router/dev/routes";

export default [
	route("/", "./modules/shell/routes/marketing-layout.route.tsx", [
		index("./modules/marketing/routes/home.route.tsx"),
		route("about", "./modules/marketing/routes/about.route.tsx"),
		route("spec", "./modules/marketing/routes/spec.route.tsx"),
		route("compliance", "./modules/marketing/routes/compliance.route.tsx"),
		route(
			"reference-workflow",
			"./modules/workflow/routes/reference-workflow.route.tsx",
		),
		route(
			"blog/ai-needs-execution",
			"./modules/marketing/routes/blog/ai-needs-execution.route.tsx",
		),
		route("pricing", "./modules/marketing/routes/pricing.route.tsx"),
		route("contact", "./modules/marketing/routes/contact.route.tsx"),
	]),
	route("/docs", "./modules/shell/routes/docs-layout.route.tsx", [
		index("./modules/docs/routes/docs-index.route.tsx"),
		route(":slug", "./modules/docs/routes/doc.route.tsx"),
	]),
	route("/protocol/small/v1", "./routes/protocol.small-v1.route.ts"),
	// Catch-all route for DevTools and other well-known requests
	route("*", "./routes/catch-all.route.tsx"),
];
