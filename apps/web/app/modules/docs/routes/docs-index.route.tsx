import { Link } from "react-router-dom";

export default function DocsIndex() {
	return (
		<div className="space-y-6">
			<p className="max-w-2xl text-white/70">
				Canonical specification and reference workflow for agent-legible content
				infrastructure.
			</p>
			<div className="flex gap-3">
				<Link
					to="/docs/primitives-v1"
					className="rounded-lg bg-white px-4 py-2 text-sm font-semibold text-zinc-950"
				>
					Read Primitive Spec v1
				</Link>
				<Link
					to="/docs/reference-workflow"
					className="rounded-lg border border-white/15 px-4 py-2 text-sm font-semibold text-white/90 hover:bg-white/5"
				>
					Run reference workflow
				</Link>
			</div>
		</div>
	);
}
