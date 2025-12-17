import { MDXProvider } from "@mdx-js/react";
import type { ComponentType } from "react";
import { mdxComponents } from "./mdx-components";

type Props = {
	Content: ComponentType<any>;
};

export function MDXRenderer({ Content }: Props) {
	return (
		<div className="prose prose-invert max-w-none">
			<MDXProvider components={mdxComponents}>
				<Content components={mdxComponents} />
			</MDXProvider>
		</div>
	);
}