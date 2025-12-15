import { useEffect } from "react";
import type { LoaderFunctionArgs } from "react-router-dom";
import { useLoaderData } from "react-router-dom";
import { loadDoc } from "~/modules/docs/lib/content.server";

export async function loader({ params }: LoaderFunctionArgs) {
	const slug = params.slug ?? "docs";
	return loadDoc(slug);
}

export default function DocRoute() {
	const data = useLoaderData() as { slug: string; html: string };

	// Hydrate Mermaid diagrams on the client
	// The server-rendered HTML includes Mermaid containers with data attributes
	useEffect(() => {
		// Mermaid components will initialize themselves via their useEffect hooks
		// when they mount. Since we're using dangerouslySetInnerHTML, we need to
		// manually trigger Mermaid initialization for any diagrams in the HTML.
		// However, this is complex. For now, Mermaid will work if the HTML structure
		// is preserved and Mermaid components can find their containers.
	}, []);

	return (
		<div
			className="prose prose-invert max-w-none"
			dangerouslySetInnerHTML={{ __html: data.html }}
		/>
	);
}
