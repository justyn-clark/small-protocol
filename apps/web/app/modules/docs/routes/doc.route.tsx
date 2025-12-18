import { useEffect, useState } from "react";
import type { LoaderFunctionArgs } from "react-router-dom";
import { useLoaderData } from "react-router-dom";
import { MDXRenderer } from "~/modules/mdx/MDXRenderer";
import { compileMdxClient } from "~/modules/mdx/mdx-runtime.client";
import { loadDoc } from "~/modules/docs/lib/content.server";

export async function loader({ params }: LoaderFunctionArgs) {
	const slug = params.slug ?? "docs";
	return loadDoc(slug);
}

export function shouldRevalidate() {
	return process.env.NODE_ENV === "development";
}

export default function DocRoute() {
	const data = useLoaderData() as {
		slug: string;
		source: string;
		frontmatter?: Record<string, unknown>;
	};
	const [Component, setComponent] = useState<React.ComponentType<any> | null>(
		null,
	);

	useEffect(() => {
		compileMdxClient(data.source).then((compiled) => {
			setComponent(() => compiled.Component);
		});
	}, [data.source]);

	if (!Component) {
		return <div>Loading...</div>;
	}

	return (
		<div className="prose prose-invert max-w-none prose-headings:scroll-mt-24">
			<MDXRenderer Content={Component} />
		</div>
	);
}
