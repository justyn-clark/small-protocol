import { NavLink } from "react-router-dom";

export function Header() {
	return (
		<header className="sticky top-0 z-40 border-b border-white/10 bg-zinc-950/80 backdrop-blur">
			<div className="mx-auto flex max-w-7xl items-center justify-between px-4 py-3">
				<NavLink to="/" className="text-sm font-semibold tracking-tight">
					SMALL
				</NavLink>
				<nav className="flex items-center gap-6 text-sm">
					<NavLink
						to="/"
						className={({ isActive }) =>
							isActive ? "text-white" : "text-white/70 hover:text-white"
						}
					>
						Overview
					</NavLink>
					<NavLink
						to="/blog/ai-needs-execution"
						className={({ isActive }) =>
							isActive ? "text-white" : "text-white/70 hover:text-white"
						}
					>
						Essay
					</NavLink>
					<NavLink
						to="/spec"
						className={({ isActive }) =>
							isActive ? "text-white" : "text-white/70 hover:text-white"
						}
					>
						Spec
					</NavLink>
					<NavLink
						to="/compliance"
						className={({ isActive }) =>
							isActive ? "text-white" : "text-white/70 hover:text-white"
						}
					>
						Compliance
					</NavLink>
					<NavLink
						to="/reference-workflow"
						className={({ isActive }) =>
							isActive ? "text-white" : "text-white/70 hover:text-white"
						}
					>
						Reference Workflow
					</NavLink>
					<NavLink
						to="/about"
						className={({ isActive }) =>
							isActive ? "text-white" : "text-white/70 hover:text-white"
						}
					>
						About
					</NavLink>
				</nav>
			</div>
		</header>
	);
}
