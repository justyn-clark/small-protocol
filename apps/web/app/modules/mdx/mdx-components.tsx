import type { ComponentProps } from "react";
import { CodeBlock } from "./CodeBlock";
import { Mermaid } from "./Mermaid";

function InlineCode(props: ComponentProps<"code">) {
	return (
		<code
			{...props}
			className={[
				"rounded-md bg-white/10 px-1.5 py-0.5 text-sm text-white",
				props.className ?? "",
			].join(" ")}
		/>
	);
}

function A(props: ComponentProps<"a">) {
	return (
		<a
			{...props}
			className={[
				"underline underline-offset-4 decoration-white/30 hover:decoration-white/70",
				props.className ?? "",
			].join(" ")}
		/>
	);
}

// NOTE: We no longer need to override <pre> because code fences will be transformed
// into <CodeBlock> / <Mermaid> at compile time. Keep <pre> default.

export const mdxComponents = {
	a: A,
	code: InlineCode,

	// Allow explicit usage in MDX too:
	CodeBlock,
	Mermaid,
};
