export default function About() {
	return (
		<div className="space-y-12 max-w-3xl">
			{/* About SMALL */}
			<div className="space-y-4">
				<h1 className="text-4xl font-bold tracking-tight">About SMALL</h1>
				<p className="text-white/80 leading-relaxed">
					SMALL is a formal execution model designed to make AI systems legible,
					deterministic, and safe to operate at scale.
				</p>
				<p className="text-white/80 leading-relaxed">
					It was created to address a recurring failure pattern across AI
					platforms, CMS systems, and internal tools: implicit state, invisible
					intent, and ungovernable execution.
				</p>
				<p className="text-white/80 leading-relaxed">
					SMALL is intentionally minimal.
					<br />
					Its goal is not to add features, but to make systems explainable.
				</p>
			</div>

			{/* About the Creator */}
			<div className="space-y-4 border-t border-white/10 pt-8">
				<h2 className="text-2xl font-semibold tracking-tight">
					About the Creator
				</h2>
				<p className="text-white/80 leading-relaxed">
					Justin Clark is a systems-focused engineer and founder with deep
					experience building production platforms across content,
					infrastructure, and AI-adjacent systems.
				</p>
				<p className="text-white/80 leading-relaxed">His work spans:</p>
				<ul className="list-disc list-inside space-y-1 text-white/80 ml-4">
					<li>CMS and workflow systems</li>
					<li>infrastructure and execution tooling</li>
					<li>
						AI-enabled platforms constrained by real-world governance
						requirements
					</li>
				</ul>
				<p className="text-white/80 leading-relaxed">
					SMALL emerged from repeatedly seeing the same failures - and
					rebuilding the same scaffolding - across otherwise unrelated systems.
				</p>
			</div>

			{/* About Justyn Clark Network */}
			<div className="space-y-4 border-t border-white/10 pt-8">
				<h2 className="text-2xl font-semibold tracking-tight">
					About Justyn Clark Network
				</h2>
				<p className="text-white/80 leading-relaxed">
					Justyn Clark Network (JCN) is a development studio focused on
					execution-first systems at the intersection of software, AI, and
					media.
				</p>
				<p className="text-white/80 leading-relaxed">JCN builds:</p>
				<ul className="list-disc list-inside space-y-1 text-white/80 ml-4">
					<li>infrastructure-grade platforms</li>
					<li>schema-driven systems</li>
					<li>tools designed for determinism, auditability, and longevity</li>
				</ul>
				<p className="text-white/80 leading-relaxed">
					SMALL is a flagship system created and maintained by JCN.
				</p>
			</div>

			{/* Attribution Line */}
			<div className="border-t border-white/10 pt-8">
				<p className="text-sm text-white/60 italic">
					SMALL is open in spirit, formal in design, and governed by explicit
					execution semantics.
				</p>
			</div>
		</div>
	);
}
