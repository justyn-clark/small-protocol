import { MDXProvider } from "@mdx-js/react";
import type { ReactNode } from "react";
import { mdxComponents } from "./mdx-components";

export function MDXRenderer({ children }: { children: ReactNode }) {
	return (
		<div className="prose prose-invert max-w-none">
			<MDXProvider components={mdxComponents}>{children}</MDXProvider>
		</div>
	);
}
