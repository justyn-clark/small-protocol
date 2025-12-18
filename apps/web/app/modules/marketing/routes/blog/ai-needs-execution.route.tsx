import { statSync } from "node:fs";
import fs from "node:fs/promises";
import path from "node:path";
import { fileURLToPath } from "node:url";
import { useEffect, useState } from "react";
import type { LoaderFunctionArgs } from "react-router-dom";
import { useLoaderData } from "react-router-dom";
import { MDXRenderer } from "~/modules/mdx/MDXRenderer";
import { compileMdxClient } from "~/modules/mdx/mdx-runtime.client";

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

export async function loader({ request }: LoaderFunctionArgs) {
	// Resolve path relative to workspace root (up from apps/web/app/modules/marketing/routes/blog)
	const workspaceRoot = path.resolve(__dirname, "../../../../../../..");
	const filePath = path.join(
		workspaceRoot,
		"data_pages_copy",
		"Why Agents Fail Without SMALL - BLOG_ESSAY.md",
	);

	const source = await fs.readFile(filePath, "utf8");
	const mtimeMs = statSync(filePath).mtimeMs;

	return {
		source,
		mtimeMs,
	};
}

export function shouldRevalidate() {
	return process.env.NODE_ENV === "development";
}

export default function EssayRoute() {
	const data = useLoaderData() as {
		source: string;
		mtimeMs: number;
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
		<div className="prose prose-invert max-w-3xl prose-headings:scroll-mt-24">
			<MDXRenderer Content={Component} />
		</div>
	);
}
