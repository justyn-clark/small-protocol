type Props = {
	html: string;
	lang?: string;
	title?: string;
};

export function CodeBlock({ html, lang, title }: Props) {
	return (
		<div className="my-6 overflow-hidden rounded-xl border border-white/10 bg-black/40">
			<div className="flex items-center justify-between border-b border-white/10 px-4 py-2">
				<div className="text-xs font-medium text-white/70">
					{title ?? (lang || "code")}
				</div>
			</div>

			{/* Shiki returns HTML with its own <pre><code> structure */}
			<div
				className="overflow-auto p-4 text-sm leading-6"
				dangerouslySetInnerHTML={{ __html: html }}
			/>
		</div>
	);
}
