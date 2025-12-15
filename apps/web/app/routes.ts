import { index, route } from "@react-router/dev/routes";

export default [
	route("/", "./modules/shell/routes/marketing-layout.route.tsx", [
		index("./modules/marketing/routes/home.route.tsx"),
		route("pricing", "./modules/marketing/routes/pricing.route.tsx"),
		route("about", "./modules/marketing/routes/about.route.tsx"),
		route("contact", "./modules/marketing/routes/contact.route.tsx"),
	]),
	route("/docs", "./modules/shell/routes/docs-layout.route.tsx", [
		index("./modules/docs/routes/docs-index.route.tsx"),
		route(
			"reference-workflow",
			"./modules/workflow/routes/reference-workflow.route.tsx",
		),
		route(":slug", "./modules/docs/routes/doc.route.tsx"),
	]),
	// Catch-all route for DevTools and other well-known requests
	route("*", "./routes/catch-all.route.tsx"),
];
