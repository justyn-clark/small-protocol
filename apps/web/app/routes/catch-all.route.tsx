import type { Route } from "./+types/catch-all";

export function meta() {
	return [
		{ title: "Not Found - React Router App" },
		{ name: "description", content: "Page not found" },
	];
}

export function loader({ request }: Route.LoaderArgs) {
	// Handle DevTools and other well-known requests silently
	if (
		request.url.includes(".well-known/") ||
		request.url.includes("devtools")
	) {
		return new Response(null, { status: 404 });
	}

	// For other 404s, return a proper response
	throw new Response("Not Found", { status: 404 });
}

export default function CatchAll() {
	return (
		<div className="flex min-h-screen items-center justify-center">
			<div className="text-center">
				<h1 className="text-4xl font-bold text-gray-900 dark:text-gray-100">
					404
				</h1>
				<p className="mt-2 text-gray-600 dark:text-gray-400">Page not found</p>
			</div>
		</div>
	);
}
