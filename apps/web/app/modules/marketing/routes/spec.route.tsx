import { Link } from "react-router-dom";

export default function SpecRoute() {
	return (
		<div className="prose prose-invert max-w-3xl">
			<div className="">
				<h1 className="text-4xl font-bold tracking-tight">
					SMALL Specification
				</h1>
				<p className="text-white/80">
					SMALL defines the minimal execution model required for agent-legible
					systems through five core primitives.
				</p>
			</div>

			<div className="space-y-6 border-white/10">
				<div>
					<h2 className="text-2xl font-semibold tracking-tight mb-4">
						The Five Primitives
					</h2>
					<ul className="space-y-3 text-white/80">
						<li>
							<strong className="text-white">Schema</strong> - What is allowed
						</li>
						<li>
							<strong className="text-white">Manifest</strong> - What is
							intended
						</li>
						<li>
							<strong className="text-white">Artifact</strong> - What exists
						</li>
						<li>
							<strong className="text-white">Lineage</strong> - Where it came
							from
						</li>
						<li>
							<strong className="text-white">Lifecycle</strong> - What happened
							and what may happen next
						</li>
					</ul>
				</div>

				<div>
					<h2 className="text-2xl font-semibold tracking-tight mb-4">
						Documentation
					</h2>
					<p className="text-white/80 mb-4">
						Detailed specifications and reference documentation:
					</p>
					<ul className="space-y-2 text-white/80">
						<li>
							<Link
								to="/docs"
								className="text-white/90 hover:text-white underline"
							>
								SMALL Documentation →
							</Link>
						</li>
						<li>
							<Link
								to="/reference-workflow"
								className="text-white/90 hover:text-white underline"
							>
								Reference Workflow →
							</Link>
						</li>
					</ul>
				</div>

				<div>
					<h2 className="text-2xl font-semibold tracking-tight mb-4">
						Protocol
					</h2>
					<p className="text-white/80">
						SMALL is a versioned, machine-consumable protocol. The protocol
						contract is available at{" "}
						<a
							href="/protocol/small/v1"
							className="text-white/90 hover:text-white underline"
						>
							/protocol/small/v1
						</a>
						.
					</p>
				</div>
			</div>
		</div>
	);
}
