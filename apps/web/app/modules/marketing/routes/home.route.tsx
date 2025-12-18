import { Link } from "react-router-dom";

export default function Home() {
	return (
		<div className="space-y-12">
			{/* Hero */}
			<div className="space-y-6">
				<h1 className="text-5xl font-bold tracking-tight">SMALL</h1>
				<p className="text-xl text-white/90">
					The Execution Model for Agent-Legible Systems
				</p>
				<p className="text-white/70 max-w-2xl">
					AI agents don't fail because models are weak.
					<br />
					They fail because the systems around them are implicit, mutable, and
					opaque.
				</p>
				<p className="text-white/70 max-w-2xl">
					SMALL replaces guesswork with execution contracts.
				</p>
				<p className="font-mono text-sm text-white/60">
					Schema → Manifest → Artifact → Lineage → Lifecycle
				</p>
			</div>

			{/* Sub-hero */}
			<div className="max-w-2xl space-y-4 border-t border-white/10 pt-8">
				<p className="text-white/80 leading-relaxed">
					SMALL is a minimal execution model that makes systems legible to
					machines. It defines how intent is declared, how state is created, how
					changes are traced, and how execution is governed - deterministically.
				</p>
				<p className="text-white/80 leading-relaxed">
					This is not a CMS.
					<br />
					It's the layer CMS-style systems are missing.
				</p>
			</div>

			{/* Five Primitives */}
			<div className="space-y-4 border-t border-white/10 pt-8">
				<ul className="space-y-2 text-white/80">
					<li>
						<strong className="text-white">Schema</strong> - what is allowed
					</li>
					<li>
						<strong className="text-white">Manifest</strong> - what is intended
					</li>
					<li>
						<strong className="text-white">Artifact</strong> - what exists
					</li>
					<li>
						<strong className="text-white">Lineage</strong> - where it came from
					</li>
					<li>
						<strong className="text-white">Lifecycle</strong> - what happened
						and what may happen next
					</li>
				</ul>
				<p className="text-sm text-white/60 mt-4">
					If a system cannot answer these questions explicitly, it is unsafe for
					agents.
				</p>
			</div>

			{/* Proof Statement */}
			<div className="border-t border-white/10 pt-8">
				<p className="text-white/80">
					SMALL already runs.
					<br />
					It enforces failure, rollback, and auditability by design.
				</p>
			</div>

			{/* CTAs */}
			<div className="space-y-3 border-t border-white/10 pt-8">
				<div>
					<Link
						to="/blog/ai-needs-execution"
						className="text-white/90 hover:text-white underline"
					>
						Read the Essay →
					</Link>
				</div>
				<div>
					<Link to="/spec" className="text-white/90 hover:text-white underline">
						View the Spec →
					</Link>
				</div>
				<div>
					<Link
						to="/reference-workflow"
						className="text-white/90 hover:text-white underline"
					>
						See the Reference Workflow →
					</Link>
				</div>
			</div>
		</div>
	);
}
